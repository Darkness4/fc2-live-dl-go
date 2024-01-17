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
	"errors"
	"unsafe"
)

type Option func(*Options)

type Options struct {
	quiet int
}

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
	o := applyOptions(opts)
	inputsC := C.malloc(C.size_t(len(inputs)) * C.size_t(unsafe.Sizeof(uintptr(0))))
	defer C.free(inputsC)

	// convert the C array to a Go Array so we can index it
	inputsCIndexable := (*[1<<30 - 1]*C.char)(inputsC)

	for idx, input := range inputs {
		inputsCIndexable[idx] = C.CString(input)
	}

	if err := C.probe(C.size_t(len(inputs)), (**C.char)(inputsC), C.int(o.quiet)); err != 0 {
		if err == C.AVERROR_EOF {
			return nil
		}
		buf := make([]byte, C.AV_ERROR_MAX_STRING_SIZE)
		C.av_make_error_string((*C.char)(unsafe.Pointer(&buf[0])), C.AV_ERROR_MAX_STRING_SIZE, err)

		return errors.New(string(buf))
	}
	return nil
}

func IsVideo(input string) (bool, error) {
	var ret int
	if err := C.is_video(C.CString(input), (*C.int)(unsafe.Pointer(&ret))); err != 0 {
		buf := make([]byte, C.AV_ERROR_MAX_STRING_SIZE)
		C.av_make_error_string((*C.char)(unsafe.Pointer(&buf[0])), C.AV_ERROR_MAX_STRING_SIZE, err)

		return false, errors.New(string(buf))
	}
	return ret >= 1, nil
}
