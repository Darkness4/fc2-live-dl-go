package try

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/logger"
	"go.uber.org/zap"
)

func Do(
	tries int,
	delay time.Duration,
	fn func() error,
) (err error) {
	if tries <= 0 {
		logger.I.Panic("tries is 0 or negative", zap.Int("tries", tries))
	}
	for try := 0; try < tries; try++ {
		err = fn()
		if err == nil {
			return nil
		}
		logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try), zap.Int("maxTries", tries))
		time.Sleep(delay)
	}
	logger.I.Warn("failed all tries", zap.Error(err))
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
		logger.I.Panic("tries is 0 or negative", zap.Int("tries", tries))
	}
	for try := 0; try < tries; try++ {
		err = fn()
		if err == nil {
			return nil
		}
		logger.I.Warn(
			"try failed",
			zap.Error(err),
			zap.Int("try", try),
			zap.Int("maxTries", tries),
			zap.String("backoff", delay.String()),
		)
		time.Sleep(delay)
		delay = delay * multiplier
		if delay > maxBackoff {
			delay = maxBackoff
		}
	}
	logger.I.Warn("failed all tries", zap.Error(err))
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
		logger.I.Panic("tries is 0 or negative", zap.Int("tries", tries))
	}

	for try := 0; try < tries; try++ {
		err = func() error {
			ctx, cancel := context.WithTimeout(parent, timeout)
			defer cancel()

			errChan := make(chan error)
			go func() {
				defer close(errChan)
				errChan <- fn(ctx, try)
			}()

			select {
			case err = <-errChan:
				if err == nil {
					return nil
				}
			case <-ctx.Done():
				err = ctx.Err()
			}
			return err
		}()
		// Finish early on context canceled
		if errors.Is(err, context.Canceled) {
			logger.I.Warn("canceled all tries", zap.Error(err))
			return err
		}

		logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try), zap.Int("maxTries", tries))
		time.Sleep(delay)
	}
	logger.I.Warn("failed all tries", zap.Error(err))
	return err
}

func DoWithResult[T interface{}](
	tries int,
	delay time.Duration,
	fn func() (T, error),
) (result T, err error) {
	if tries <= 0 {
		logger.I.Panic("tries is 0 or negative", zap.Int("tries", tries))
	}
	for try := 0; try < tries; try++ {
		result, err = fn()
		if err == nil {
			return result, nil
		}
		logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try))
		time.Sleep(delay)
	}
	logger.I.Warn("failed all tries", zap.Error(err))
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
		logger.I.Panic("tries is 0 or negative", zap.Int("tries", tries))
	}
	var mu sync.Mutex
	errChan := make(chan error)
	resultChan := make(chan T)
	defer func() {
		mu.Lock()
		close(errChan)
		close(resultChan)
		mu.Unlock()
	}()

	for try := 0; try < tries; try++ {
		ctx, cancel := context.WithTimeout(parent, timeout)
		defer cancel()

		go func(resultChan chan T, errChan chan error) {
			mu.Lock()
			result, err = fn(ctx, try)
			if err != nil {
				errChan <- err
			} else {
				resultChan <- result
			}
			mu.Unlock()
		}(resultChan, errChan)

		select {
		case err = <-errChan:

		case result := <-resultChan:
			return result, nil
		case <-ctx.Done():
			err = ctx.Err()
		}
		// Finish early on context canceled
		if errors.Is(err, context.Canceled) {
			logger.I.Warn("canceled all tries")
			return result, err
		}
		logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try), zap.Int("maxTries", tries))
		time.Sleep(delay)
	}
	logger.I.Warn("failed all tries", zap.Error(err))
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
		logger.I.Panic("tries is 0 or negative", zap.Int("tries", tries))
	}
	for try := 0; try < tries; try++ {
		result, err = fn()
		if err == nil {
			return result, nil
		}
		logger.I.Warn(
			"try failed",
			zap.Error(err),
			zap.Int("try", try),
			zap.Int("maxTries", tries),
			zap.String("backoff", delay.String()),
		)
		time.Sleep(delay)
		delay = delay * time.Duration(multiplier)
		if delay > maxBackoff {
			delay = maxBackoff
		}
	}
	logger.I.Warn("failed all tries", zap.Error(err))
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
		logger.I.Panic("tries is 0 or negative", zap.Int("tries", tries))
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
		logger.I.Warn(
			"try failed",
			zap.Error(err),
			zap.Int("try", try),
			zap.Int("maxTries", tries),
			zap.String("backoff", delay.String()),
		)
		time.Sleep(delay)
		delay = delay * time.Duration(multiplier)
		if delay > maxBackoff {
			delay = maxBackoff
		}
	}
	logger.I.Warn("failed all tries", zap.Error(err))
	return result, err
}
