package concat

/*
#cgo pkg-config: libavformat libavcodec libavutil
#include "concat.h"

#include <stddef.h>
#include <stdlib.h>
#include <libavutil/common.h>
*/
import "C"
import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

var formatPriorities = map[string]int{
	".ts":  100,
	".mkv": 50,
	".mp4": 20,
	".avi": 10,
	".m4a": 1,
	"mp3":  0,
}

func getFormatPriority(ext string) int {
	priority, ok := formatPriorities[ext]
	if !ok {
		return -1
	}
	return priority
}

type Option func(*Options)

type Options struct {
	audioOnly    int
	numbered     bool
	ignoreSingle bool
}

func WithAudioOnly() Option {
	return func(o *Options) {
		o.audioOnly = 1
	}
}

// IgnoreExtension forces the concatenation on files without taking account of the extension.
//
// TS files are prioritized.
//
// Example: 1.ts, 1.mp4, 2.ts -> 1.mp4 will be skipped.
func IgnoreExtension() Option {
	return func(o *Options) {
		o.numbered = true
	}
}

// IgnoreSingle file. This is useful when the file has already been remux.
func IgnoreSingle() Option {
	return func(o *Options) {
		o.ignoreSingle = true
	}
}

func applyOptions(opts []Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Do concat multiple video streams.
func Do(output string, inputs []string, opts ...Option) error {
	o := applyOptions(opts)

	if o.ignoreSingle && len(inputs) <= 1 {
		return nil
	}

	inputsC := C.malloc(C.size_t(len(inputs)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	// convert the C array to a Go Array so we can index it
	inputsCIndexable := (*[1<<30 - 1]*C.char)(inputsC)

	for idx, input := range inputs {
		inputsCIndexable[idx] = C.CString(input)
	}

	if err := C.concat(C.CString(output), C.size_t(len(inputs)), (**C.char)(inputsC), C.int(o.audioOnly)); err != 0 {
		if err == C.AVERROR_EOF {
			return nil
		}
		buf := make([]byte, C.AV_ERROR_MAX_STRING_SIZE)
		C.av_make_error_string((*C.char)(unsafe.Pointer(&buf[0])), C.AV_ERROR_MAX_STRING_SIZE, err)

		return errors.New(string(buf))
	}
	return nil
}

func filterFiles(names []string, base string, path string, o *Options) ([]string, error) {
	selectedMap := make(map[string]string)
	for _, name := range names {
		// Ignore files with "combined"
		if strings.Contains(name, ".combined.") {
			continue
		}

		ext := filepath.Ext(name)
		var uniqueID string
		if o.numbered {
			uniqueID = strings.TrimSuffix(name, ext)
		} else {
			uniqueID = name
		}

		if strings.HasPrefix(name, base) {
			if selectedMap[uniqueID] == "" {
				selectedMap[uniqueID] = filepath.Join(path, name)
				continue
			}

			// Conflicts
			if getFormatPriority(
				strings.ToLower(ext),
			) > getFormatPriority(
				strings.ToLower(filepath.Ext(selectedMap[uniqueID])),
			) {
				selectedMap[uniqueID] = filepath.Join(path, name)
			}
		}
	}

	selected := make([]string, 0, len(selectedMap))
	for _, v := range selectedMap {
		selected = append(selected, v)
	}

	sort.Slice(selected, func(i, j int) bool {
		a := selected[i]
		b := selected[j]
		numA := extractNumericSuffix(a)
		numB := extractNumericSuffix(b)

		baseA := strings.TrimSuffix(a, filepath.Ext(a))
		baseA = strings.TrimSuffix(baseA, "."+numA)
		baseB := strings.TrimSuffix(b, filepath.Ext(b))
		baseB = strings.TrimSuffix(baseB, "."+numB)

		if baseA == baseB {
			return numA < numB
		}

		return a < b
	})

	return selected, nil
}

// WithPrefix Concat multiple videos with a prefix.
//
// Prefix can be a path.
func WithPrefix(remuxFormat string, prefix string, opts ...Option) error {
	o := applyOptions(opts)
	path := filepath.Dir(prefix)
	base := filepath.Base(prefix)
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, de := range entries {
		names = append(names, de.Name())
	}

	selected, err := filterFiles(names, base, path, o)
	if err != nil {
		return err
	}

	return Do(prefix+".combined."+remuxFormat, selected, opts...)
}

func extractNumericSuffix(filename string) string {
	// Extracts the numeric suffix from the filename, if present
	ext := filepath.Ext(filename)
	filename = strings.TrimSuffix(filename, ext)

	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		_, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			return "0"
		}
		return parts[len(parts)-1]
	}
	return "0"
}
