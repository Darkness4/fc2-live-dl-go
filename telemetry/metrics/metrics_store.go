package metrics

import (
	"go.opentelemetry.io/otel/metric"
)

const meterName = "github.com/darkness4/fc2-live-dl-go"

var (
	Downloads struct {
		InitTime       metric.Float64Histogram
		CompletionTime metric.Float64Histogram
		Errors         metric.Int64Counter
		Runs           metric.Int64Counter
		LastRun        metric.Int64Gauge
	}

	Concat struct {
		CompletionTime metric.Float64Histogram
		Errors         metric.Int64Counter
		Runs           metric.Int64Counter
		LastRun        metric.Int64Gauge
	}

	// TODO: Find HTTP metrics

	Watcher struct {
		State metric.Int64Gauge
	}

	Cleaner struct {
		FilesRemoved metric.Int64Counter
		Scans        metric.Int64Counter
		Errors       metric.Int64Counter
		Runs         metric.Int64Counter
		LastRun      metric.Int64Gauge
	}
)

func InitMetrics(provider metric.MeterProvider) {
	// Downloads
	meter := provider.Meter(meterName)

	var err error
	Downloads.InitTime, err = meter.Float64Histogram(
		"downloads.init.time",
		metric.WithDescription("Time taken to initiate a download"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}
	Downloads.CompletionTime, err = meter.Float64Histogram(
		"downloads.time_to_complete",
		metric.WithDescription("Time taken to complete a download"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}
	Downloads.Errors, err = meter.Int64Counter(
		"downloads.errors",
		metric.WithDescription("Number of errors during downloads"),
	)
	if err != nil {
		panic(err)
	}
	Downloads.Runs, err = meter.Int64Counter(
		"downloads.runs",
		metric.WithDescription("Number of downloads"),
	)
	if err != nil {
		panic(err)
	}
	Downloads.LastRun, err = meter.Int64Gauge(
		"downloads.last_run",
		metric.WithDescription("Timestamp of the last download run"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	// Concat
	Concat.CompletionTime, err = meter.Float64Histogram(
		"concat.completion.time",
		metric.WithDescription("Time taken to complete a concat"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}
	Concat.Errors, err = meter.Int64Counter(
		"concat.errors",
		metric.WithDescription("Accumulated failed runs of concats"),
	)
	if err != nil {
		panic(err)
	}
	Concat.Runs, err = meter.Int64Counter(
		"concat.runs",
		metric.WithDescription("Number of concats"),
	)
	if err != nil {
		panic(err)
	}
	Concat.LastRun, err = meter.Int64Gauge(
		"concat.last_run",
		metric.WithDescription("Timestamp of the last concat run"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}

	// States
	Watcher.State, err = meter.Int64Gauge(
		"watcher.state",
		metric.WithDescription("Current state of the watcher"),
	)
	if err != nil {
		panic(err)
	}

	// Cleaner
	Cleaner.FilesRemoved, err = meter.Int64Counter(
		"cleaner.files_removed",
		metric.WithDescription("Number of files removed"),
	)
	if err != nil {
		panic(err)
	}
	Cleaner.Errors, err = meter.Int64Counter(
		"cleaner.errors",
		metric.WithDescription("Number of errors during cleaning"),
	)
	if err != nil {
		panic(err)
	}
	Cleaner.Runs, err = meter.Int64Counter(
		"cleaner.runs",
		metric.WithDescription("Number of cleaning runs"),
	)
	if err != nil {
		panic(err)
	}
	Cleaner.Scans, err = meter.Int64Counter(
		"cleaner.scans",
		metric.WithDescription("Number of scans"),
	)
	if err != nil {
		panic(err)
	}
	Cleaner.LastRun, err = meter.Int64Gauge(
		"cleaner.last_run",
		metric.WithDescription("Timestamp of the last cleaning run"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}
}
