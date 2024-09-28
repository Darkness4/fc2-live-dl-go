package utils

import (
	"context"
	"errors"
	"io"
)

// GetFirstValuableErrorOrFirst returns the first error that is not nil, not EOF, and not context.Canceled. If not found, returns the first error.
func GetFirstValuableErrorOrFirst(errs []error) error {
	var nonNilErr error
	var firstEOF error
	for _, err := range errs {
		if err != nil {
			if nonNilErr == nil {
				nonNilErr = err
			}
			if firstEOF == nil && errors.Is(err, io.EOF) {
				firstEOF = err
			}
			if !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
				return err
			}
		}
	}
	if firstEOF != nil {
		return firstEOF
	}
	return nonNilErr
}
