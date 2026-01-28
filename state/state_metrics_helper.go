package state

import (
	"context"

	"github.com/Darkness4/fc2-live-dl-go/telemetry/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// setStateMetrics demuxes the state to the metrics.
func setStateMetrics(
	ctx context.Context,
	channelID string,
	state DownloadState,
	labels map[string]string,
) {
	attrs := make([]attribute.KeyValue, 0, len(labels)+1)
	attrs = append(attrs, attribute.String("channel_id", channelID))
	for k, v := range labels {
		attrs = append(attrs, attribute.String(k, v))
	}
	m := metrics.Watcher.State
	m.Record(
		ctx,
		1,
		metric.WithAttributes(append(attrs, attribute.String("state", state.String()))...),
	)
	// Remove the rest of the states from the metrics.
	for i := DownloadStateUnspecified; i <= DownloadStateCanceled; i++ {
		if i != state {
			m.Record(
				ctx,
				0,
				metric.WithAttributes(append(attrs, attribute.String("state", i.String()))...),
			)
		}
	}
}
