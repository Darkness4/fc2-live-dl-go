// Package try provides a set of functions to retry a function with a delay.
package try

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
)

// Do tries a function with a delay.
func Do(
	tries int,
	delay time.Duration,
	fn func() error,
) (err error) {
	if tries <= 0 {
		log.Panic().Int("tries", tries).Msg("tries is 0 or negative")
	}
	for try := 0; try < tries; try++ {
		err = fn()
		if err == nil {
			return nil
		}
		log.Warn().
			Str("parentCaller", getCaller()).
			Err(err).
			Int("try", try).
			Int("maxTries", tries).
			Msg("try failed")
		time.Sleep(delay)
	}
	log.Warn().Err(err).Msg("failed all tries")
	return err
}

// DoExponentialBackoff tries a function with exponential backoff.
func DoExponentialBackoff(
	tries int,
	delay time.Duration,
	multiplier time.Duration,
	maxBackoff time.Duration,
	fn func() error,
) (err error) {
	if tries <= 0 {
		log.Panic().Int("tries", tries).Msg("tries is 0 or negative")
	}
	for try := 0; try < tries; try++ {
		err = fn()
		if err == nil {
			return nil
		}
		log.Warn().
			Str("parentCaller", getCaller()).
			Err(err).
			Int("try", try).
			Int("maxTries", tries).
			Stringer("backoff", delay).
			Msg("try failed")
		time.Sleep(delay)
		delay = delay * multiplier
		if delay > maxBackoff {
			delay = maxBackoff
		}
	}
	log.Warn().Err(err).Msg("failed all tries")
	return err
}

// DoWithContextTimeout tries a function with context and timeout.
func DoWithContextTimeout(
	parent context.Context,
	tries int,
	delay time.Duration,
	timeout time.Duration,
	fn func(ctx context.Context, try int) error,
) (err error) {
	if tries <= 0 {
		log.Panic().Int("tries", tries).Msg("tries is 0 or negative")
	}

	for try := 0; try < tries; try++ {
		err = func() error {
			ctx, cancel := context.WithTimeout(parent, timeout)
			defer cancel()

			return fn(ctx, try)
		}()
		if err == nil {
			return nil
		}
		// Finish early on context canceled
		if errors.Is(err, context.Canceled) {
			log.Warn().Err(err).Msg("canceled all tries")
			return err
		}

		log.Warn().
			Str("parentCaller", getCaller()).
			Err(err).
			Int("try", try).
			Int("maxTries", tries).
			Msg("try failed")
		time.Sleep(delay)
	}
	log.Warn().Err(err).Msg("failed all tries")
	return err
}

// DoWithResult tries a function and returns a result.
//
// nolint: ireturn
func DoWithResult[T any](
	tries int,
	delay time.Duration,
	fn func() (T, error),
) (result T, err error) {
	if tries <= 0 {
		log.Panic().Int("tries", tries).Msg("tries is 0 or negative")
	}
	for try := 0; try < tries; try++ {
		result, err = fn()
		if err == nil {
			return result, nil
		}
		log.Warn().Str("parentCaller", getCaller()).Int("try", try).Err(err).Msg("try failed")
		time.Sleep(delay)
	}
	log.Warn().Err(err).Msg("failed all tries")
	return result, err
}

// DoWithContextTimeoutWithResult performs a function with context
// and returns a result.
//
// nolint: ireturn
func DoWithContextTimeoutWithResult[T any](
	parent context.Context,
	tries int,
	delay time.Duration,
	timeout time.Duration,
	verbose bool,
	fn func(ctx context.Context, try int) (T, error),
) (result T, err error) {
	if tries <= 0 {
		log.Panic().Int("tries", tries).Msg("tries is 0 or negative")
	}

	for try := 0; try < tries; try++ {
		ctx, cancel := context.WithTimeout(parent, timeout)
		defer cancel()

		result, err = fn(ctx, try)
		if err == nil {
			return result, nil
		}
		// Finish early on context canceled
		if errors.Is(err, context.Canceled) {
			if verbose {
				log.Warn().Msg("canceled all tries")
			}
			return result, err
		}
		if verbose {
			log.Warn().
				Str("parentCaller", getCaller()).
				Int("try", try).
				Int("maxTries", tries).
				Err(err).
				Msg("try failed")
		}
		time.Sleep(delay)
	}
	if verbose {
		log.Warn().Err(err).Msg("failed all tries")
	}
	return result, err
}

// DoExponentialBackoffWithResult performs an exponential backoff and return a result.
//
// nolint: ireturn
func DoExponentialBackoffWithResult[T any](
	tries int,
	delay time.Duration,
	multiplier int,
	maxBackoff time.Duration,
	fn func() (T, error),
) (result T, err error) {
	if tries <= 0 {
		log.Panic().Int("tries", tries).Msg("tries is 0 or negative")
	}
	for try := 0; try < tries; try++ {
		result, err = fn()
		if err == nil {
			return result, nil
		}
		log.Warn().
			Str("parentCaller", getCaller()).
			Int("try", try).
			Int("maxTries", tries).
			Stringer("backoff", delay).
			Err(err).Msg(
			"try failed",
		)
		time.Sleep(delay)
		delay = delay * time.Duration(multiplier)
		if delay > maxBackoff {
			delay = maxBackoff
		}
	}
	log.Warn().Err(err).Msg("failed all tries")
	return result, err
}

// DoExponentialBackoffWithContextAndResult performs an exponential backoff
// with context and returns a result
//
// nolint: ireturn
func DoExponentialBackoffWithContextAndResult[T any](
	parent context.Context,
	tries int,
	delay time.Duration,
	multiplier int,
	maxBackoff time.Duration,
	fn func(ctx context.Context) (T, error),
) (result T, err error) {
	if tries <= 0 {
		log.Panic().Int("tries", tries).Msg("tries is 0 or negative")
	}
	for try := 0; try < tries; try++ {
		result, err = fn(parent)
		if err == nil {
			return result, nil
		}
		// Context cancellation means early exit
		if errors.Is(err, context.Canceled) {
			return result, context.Canceled
		}
		log.Warn().
			Str("parentCaller", getCaller()).
			Err(err).
			Int("try", try).
			Int("maxTries", tries).
			Stringer("backoff", delay).
			Msg("try failed")
		time.Sleep(delay)
		delay = delay * time.Duration(multiplier)
		if delay > maxBackoff {
			delay = maxBackoff
		}
	}
	log.Warn().Err(err).Msg("failed all tries")
	return result, err
}

func getCaller() string {
	// Skip 2 frames to get the caller of the function calling this function
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%s:%d", file, line)
}
