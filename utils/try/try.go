package try

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

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
		log.Warn().Err(err).Int("try", try).Int("maxTries", tries).Msg("try failed")
		time.Sleep(delay)
	}
	log.Warn().Err(err).Msg("failed all tries")
	return err
}

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

		log.Warn().Err(err).Int("try", try).Int("maxTries", tries).Msg("try failed")
		time.Sleep(delay)
	}
	log.Warn().Err(err).Msg("failed all tries")
	return err
}

func DoWithResult[T interface{}](
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
		log.Warn().Int("try", try).Err(err).Msg("try failed")
		time.Sleep(delay)
	}
	log.Warn().Err(err).Msg("failed all tries")
	return result, err
}

func DoWithContextTimeoutWithResult[T interface{}](
	parent context.Context,
	tries int,
	delay time.Duration,
	timeout time.Duration,
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
			log.Warn().Msg("canceled all tries")
			return result, err
		}
		log.Warn().Int("try", try).Int("maxTries", tries).Err(err).Msg("try failed")
		time.Sleep(delay)
	}
	log.Warn().Err(err).Msg("failed all tries")
	return result, err
}

func DoExponentialBackoffWithResult[T interface{}](
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

func DoExponentialBackoffWithContextAndResult[T interface{}](
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
