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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unsafe"

	"github.com/Darkness4/fc2-live-dl-go/telemetry/metrics"
	"github.com/Darkness4/fc2-live-dl-go/video/probe"
	gopointer "github.com/mattn/go-pointer"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "video/concat"

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
func Do(ctx context.Context, output string, inputs []string, opts ...Option) error {
	o := applyOptions(opts)

	attrs := make([]attribute.KeyValue, 0, len(inputs))
	for idx, input := range inputs {
		attrs = append(attrs, attribute.String(fmt.Sprintf("input%d", idx), input))
	}
	attrs = append(attrs, attribute.String("output", output))
	attrs = append(attrs, attribute.Bool("audio_only", o.audioOnly == 1))
	attrs = append(attrs, attribute.Bool("numbered", o.numbered))
	attrs = append(attrs, attribute.Bool("ignore_single", o.ignoreSingle))

	ctx, span := otel.Tracer(tracerName).
		Start(ctx, "concat.Do", trace.WithAttributes(attrs...))
	defer span.End()

	end := metrics.TimeStartRecording(
		ctx,
		metrics.Concat.CompletionTime,
		metric.WithAttributes(attrs...),
	)
	defer end()
	metrics.Concat.Runs.Add(ctx, 1)

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
		cInput := C.CString(input)
		defer C.free(unsafe.Pointer(cInput))
		inputsCIndexable[idx] = cInput
	}

	ctxp := gopointer.Save(&ctx)
	defer gopointer.Unref(ctxp)

	cOutput := C.CString(output)
	defer C.free(unsafe.Pointer(cOutput))

	if err := C.concat(ctxp, cOutput, C.size_t(len(inputs)), (**C.char)(inputsC), C.int(o.audioOnly)); err != 0 {
		if err == C.AVERROR_EOF {
			return nil
		}
		buf := make([]byte, C.AV_ERROR_MAX_STRING_SIZE)
		C.av_make_error_string((*C.char)(unsafe.Pointer(&buf[0])), C.AV_ERROR_MAX_STRING_SIZE, err)

		err := errors.New(string(buf))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		metrics.Concat.Errors.Add(ctx, 1)

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
func WithPrefix(ctx context.Context, remuxFormat string, prefix string, opts ...Option) error {
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

	return Do(ctx, prefix+".combined."+remuxFormat, selected, opts...)
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

//export goTraceProcessInputStart
func goTraceProcessInputStart(
	ctxp unsafe.Pointer,
	inputIndex C.size_t,
	input *C.char,
) unsafe.Pointer {
	if ctxp == nil {
		return nil
	}
	ctx := gopointer.Restore(ctxp).(*context.Context)
	_, span := otel.Tracer(tracerName).
		Start(*ctx, "concat.ProcessInput",
			trace.WithAttributes(
				attribute.Int64("input_index", int64(inputIndex)),
				attribute.String("input", C.GoString(input)),
			),
		)
	return gopointer.Save(span)
}

//export goTraceProcessInputEnd
func goTraceProcessInputEnd(spanp unsafe.Pointer) {
	if spanp == nil {
		return
	}
	span := gopointer.Restore(spanp).(trace.Span)
	span.End()
	gopointer.Unref(spanp)
}
