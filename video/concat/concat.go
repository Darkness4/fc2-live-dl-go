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

	"github.com/rs/zerolog/log"
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

// Option is a function that configures the concatenation.
type Option func(*Options)

// Options are the concatenation options.
type Options struct {
	audioOnly    int
	numbered     bool
	ignoreSingle bool
}

// WithAudioOnly forces the concatenation on audio only.
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

	log.Info().Str("output", output).Strs("inputs", inputs).Any("options", o).Msg("concat")

	if o.ignoreSingle && len(inputs) <= 1 {
		return nil
	}

	inputsC := C.malloc(C.size_t(len(inputs)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	defer C.free(inputsC)
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

func filterFiles(
	names []string,
	base string,
	path string,
	o *Options,
) ([]string, error) {
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
		orderA := extractOrderPart(base, a)
		orderB := extractOrderPart(base, b)

		// Check numeric ordering
		valueA, errA := strconv.Atoi(orderA)
		valueB, errB := strconv.Atoi(orderB)
		if errA == nil && errB == nil {
			return valueA < valueB
		}

		// Check lexico-ordering
		return orderA < orderB
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

func extractOrderPart(prefix string, filename string) string {
	// Extracts the numeric suffix from the filename, if present
	ext := filepath.Ext(filename)
	filename = strings.TrimSuffix(filename, ext)
	filename = strings.TrimPrefix(filename, prefix)
	filename = strings.Trim(filename, ".")

	if filename == "" {
		return "0"
	}

	return filename
}
