package remux

/*
#cgo pkg-config: libavformat libavcodec libavutil
#include "remux.h"

#include <libavutil/common.h>
*/
import "C"
import (
	"errors"
	"unsafe"
)

type Option func(*Options)

type Options struct {
	audioOnly int
}

func WithAudioOnly() Option {
	return func(o *Options) {
		o.audioOnly = 1
	}
}

func applyOptions(opts []Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func Do(input string, output string, opts ...Option) error {
	o := applyOptions(opts)
	if err := C.remux(C.CString(input), C.CString(output), C.int(o.audioOnly)); err != 0 {
		if err == C.AVERROR_EOF {
			return nil
		}
		buf := make([]byte, C.AV_ERROR_MAX_STRING_SIZE)
		C.av_make_error_string((*C.char)(unsafe.Pointer(&buf[0])), C.AV_ERROR_MAX_STRING_SIZE, err)

		return errors.New(string(buf))
	}
	return nil
}
