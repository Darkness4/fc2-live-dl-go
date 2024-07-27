package metrics

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
)

var (
	startTimes = make(map[string]time.Time)
	mu         sync.Mutex
)

// TimeStartRecording starts a timer and returns a function that records the
// elapsed time to the given histogram metric when called.
func TimeStartRecording(
	ctx context.Context,
	m metric.Float64Histogram,
	unit time.Duration,
	opts ...metric.RecordOption,
) func() {
	start := time.Now()
	return func() {
		switch unit {
		case time.Nanosecond:
			m.Record(ctx, float64(time.Since(start).Nanoseconds()), opts...)
		case time.Microsecond:
			m.Record(ctx, float64(time.Since(start).Microseconds()), opts...)
		case time.Millisecond:
			m.Record(ctx, float64(time.Since(start).Milliseconds()), opts...)
		default:
			m.Record(ctx, time.Since(start).Seconds(), opts...)
		}
	}
}

// TimeStartRecordingDeferred starts a timer and stores the start time in a
// map. The caller is responsible for calling TimeEndRecording with the same
// id to record the elapsed time.
func TimeStartRecordingDeferred(id string) {
	mu.Lock()
	defer mu.Unlock()
	start := time.Now()
	startTimes[id] = start
}

// TimeEndRecording records the elapsed time since the corresponding call to
// TimeStartRecordingDeferred.
func TimeEndRecording(
	ctx context.Context,
	m metric.Float64Histogram,
	id string,
	opts ...metric.RecordOption,
) {
	mu.Lock()
	defer mu.Unlock()
	start, ok := startTimes[id]
	if !ok {
		return
	}
	delete(startTimes, id)
	m.Record(ctx, time.Since(start).Seconds(), opts...)
}
