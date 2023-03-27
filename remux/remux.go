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

func cbool(value bool) C.int {
	if value {
		return 1
	} else {
		return 0
	}
}

func Do(input string, output string, audioOnly bool) error {
	if err := C.remux(C.CString(input), C.CString(output), cbool(audioOnly)); err != 0 {
		if err == C.AVERROR_EOF {
			return nil
		}
		buf := make([]byte, C.AV_ERROR_MAX_STRING_SIZE)
		C.av_make_error_string((*C.char)(unsafe.Pointer(&buf[0])), C.AV_ERROR_MAX_STRING_SIZE, err)

		return errors.New(string(buf))
	}
	return nil
}
