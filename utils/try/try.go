package try

import (
	"context"
	"time"

	"github.com/Darkness4/fc2-live-dl-lite/logger"
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
		logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try), zap.Int("maxTries", tries))
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
				if err != nil {
					logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try), zap.Int("maxTries", tries))
				}
				if err == nil {
					return nil
				}
			case <-ctx.Done():
				err = ctx.Err()
				logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try), zap.Int("maxTries", tries))
			}
			return err
		}()

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
	fn func(try int) (T, error),
) (result T, err error) {
	if tries <= 0 {
		logger.I.Panic("tries is 0 or negative", zap.Int("tries", tries))
	}
	errChan := make(chan error)
	defer close(errChan)
	resultChan := make(chan T)
	defer close(resultChan)

	for try := 0; try < tries; try++ {
		ctx, cancel := context.WithTimeout(parent, timeout)
		defer cancel()

		go func(resultChan chan T, errChan chan error) {
			result, err = fn(try)
			if err != nil {
				errChan <- err
			} else {
				resultChan <- result
			}
		}(resultChan, errChan)

		select {
		case err = <-errChan:
			if err != nil {
				logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try), zap.Int("maxTries", tries))
			}
		case result := <-resultChan:
			return result, nil
		case <-ctx.Done():
			err = ctx.Err()
			logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try), zap.Int("maxTries", tries))
		}
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
		logger.I.Warn("try failed", zap.Error(err), zap.Int("try", try), zap.Duration("backoff", delay))
		time.Sleep(delay)
		delay = delay * time.Duration(multiplier)
		if delay > maxBackoff {
			delay = maxBackoff
		}
	}
	logger.I.Warn("failed all tries", zap.Error(err))
	return result, err
}
