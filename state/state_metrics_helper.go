package state

import (
	"context"

	"github.com/Darkness4/fc2-live-dl-go/telemetry/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// setStateMetrics demuxes the state to the metrics.
func setStateMetrics(ctx context.Context, channelID string, state DownloadState) {
	m := metrics.Watcher.State
	m.Record(ctx, 1, metric.WithAttributes(
		attribute.String("channel_id", channelID),
		attribute.String("state", state.String()),
	))
	for i := DownloadStateUnspecified; i <= DownloadStateCanceled; i++ {
		if i != state {
			m.Record(ctx, 0, metric.WithAttributes(
				attribute.String("channel_id", channelID),
				attribute.String("state", i.String()),
			))
		}
	}
}
