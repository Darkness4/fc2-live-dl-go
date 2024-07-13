// Package probe provide a probe for checking video containers.
package probe

/*
#cgo pkg-config: libavformat libavcodec libavutil
#include "probe.h"

#include <stddef.h>
#include <stdlib.h>
#include <libavutil/common.h>
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"unsafe"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "video/probe"

// Option is a function that configures the probe.
type Option func(*Options)

// Options is a probe options.
type Options struct {
	quiet int
}

// WithQuiet sets the quiet option.
func WithQuiet() Option {
	return func(o *Options) {
		o.quiet = 1
	}
}

func applyOptions(opts []Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Do probe multiple video streams.
func Do(inputs []string, opts ...Option) error {
	attrs := make([]attribute.KeyValue, 0, len(inputs))
	for idx, input := range inputs {
		attrs = append(attrs, attribute.String(fmt.Sprintf("input%d", idx), input))
	}
	_, span := otel.Tracer(tracerName).
		Start(context.Background(), "probe.Do", trace.WithAttributes(attrs...))
	defer span.End()

	o := applyOptions(opts)
	inputsC := C.malloc(C.size_t(len(inputs)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	defer C.free(inputsC)

	// convert the C array to a Go Array so we can index it
	inputsCIndexable := (*[1<<30 - 1]*C.char)(inputsC)

	for idx, input := range inputs {
		cInput := C.CString(input)
		defer C.free(unsafe.Pointer(cInput))
		inputsCIndexable[idx] = cInput
	}

	if err := C.probe(C.size_t(len(inputs)), (**C.char)(inputsC), C.int(o.quiet)); err != 0 {
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

// ContainsVideoOrAudio checks if the input contains video or audio.
func ContainsVideoOrAudio(input string) (bool, error) {
	cInput := C.CString(input)
	defer C.free(unsafe.Pointer(cInput))
	s := C.contains_video_or_audio(cInput)
	if s.err != 0 {
		buf := make([]byte, C.AV_ERROR_MAX_STRING_SIZE)
		C.av_make_error_string(
			(*C.char)(unsafe.Pointer(&buf[0])),
			C.AV_ERROR_MAX_STRING_SIZE,
			s.err,
		)

		return false, errors.New(string(buf))
	}
	return s.contains_video_or_audio >= 1, nil
}

// IsMPEGTSOrAAC checks if the input is MPEG-TS or AAC container.
func IsMPEGTSOrAAC(input string) (bool, error) {
	cInput := C.CString(input)
	defer C.free(unsafe.Pointer(cInput))
	s := C.is_mpegts_or_aac(cInput)
	if s.err != 0 {
		buf := make([]byte, C.AV_ERROR_MAX_STRING_SIZE)
		C.av_make_error_string(
			(*C.char)(unsafe.Pointer(&buf[0])),
			C.AV_ERROR_MAX_STRING_SIZE,
			s.err,
		)

		return false, errors.New(string(buf))
	}
	return s.is_mpegts_or_aac >= 1, nil
}
