// Package concat provides a way to concatenate video files.
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
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"github.com/Darkness4/fc2-live-dl-go/video/probe"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("video/concat")

var formatPriorities = map[string]int{
	".ts":  100, // mpegts
	".mkv": 50,  // matroska
	".mp4": 20,  // mpeg4
	".avi": 10,  // avi
	".aac": 5,   // aac, which includes adts equivalent to mpegts
	".m4a": 1,   // mpeg4 audio
	".mp3": 0,   // mpeg audio
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
	ctx, span := tracer.Start(context.Background(), "concat.Do")
	defer span.End()

	o := applyOptions(opts)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Info().Str("output", output).Strs("inputs", inputs).Any("options", o).Msg("concat")

	if o.ignoreSingle && len(inputs) <= 1 {
		return nil
	}

	// If mixed formats (adts vs asc), we should remux the others first using intermediates or FIFO
	if areFormatMixed(inputs) {
		i, useFIFO, err := remuxMixedTS(ctx, inputs, opts...)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		inputs = i

		if !useFIFO {
			// Delete intermediates
			defer func() {
				for _, input := range i {
					if err := os.Remove(input); err != nil {
						log.Error().
							Err(err).
							Str("file", input).
							Msg("failed to remove intermediate file")
					}
				}
			}()
		}
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

		err := errors.New(string(buf))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return err
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

func areFormatMixed(files []string) bool {
	if len(files) <= 1 {
		return false
	}

	// Check if there are mixed formats
	ts := 0
	for _, file := range files {
		is, err := probe.IsMPEGTSOrAAC(file)
		if err != nil {
			log.Err(err).Msg("failed to probe file to determine format, will use extension")
			ext := strings.ToLower(filepath.Ext(file))
			is = ext == ".ts" || ext == ".aac"
		}
		if is {
			ts++
		}
	}
	return ts > 0 && ts < len(files)
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
