// Package metrics provides a way to record metrics.
package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "github.com/darkness4/fc2-live-dl-go"

var (
	// Downloads metrics
	Downloads struct {
		// InitTime is the time taken to initiate a download.
		InitTime metric.Float64Histogram
		// CompletionTime is the time taken to complete a download.
		CompletionTime metric.Float64Histogram
		// Errors is the number of errors during downloads.
		Errors metric.Int64Counter
		// Runs is the number of downloads.
		Runs metric.Int64Counter
	}

	// Concat metrics
	Concat struct {
		// CompletionTime is the time taken to complete a concat.
		CompletionTime metric.Float64Histogram
		// Errors is the accumulated failed runs of concats.
		Errors metric.Int64Counter
		// Runs is the number of concats.
		Runs metric.Int64Counter
	}

	// PostProcessing metrics
	PostProcessing struct {
		// CompletionTime is the time taken to complete a post process.
		CompletionTime metric.Float64Histogram
		// Errors is the accumulated failed runs of post processes.
		Errors metric.Int64Counter
		// Runs is the number of post processes.
		Runs metric.Int64Counter
	}

	// Watcher metrics
	Watcher struct {
		// State is the current state of the watcher.
		State metric.Int64Gauge
	}

	// Cleaner metrics
	Cleaner struct {
		// FilesRemoved is the number of files removed.
		FilesRemoved metric.Int64Counter
		// Errors is the number of errors during cleaning.
		Errors metric.Int64Counter
		// Runs is the number of cleaning runs.
		Runs metric.Int64Counter
		// CleanTime is the time taken to clean.
		CleanTime metric.Float64Histogram
	}
)

func init() {
	// Downloads
	meter := otel.GetMeterProvider().Meter(meterName)

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
	Downloads.Errors.Add(context.Background(), 0)
	Downloads.Runs, err = meter.Int64Counter(
		"downloads.runs",
		metric.WithDescription("Number of downloads"),
	)
	if err != nil {
		panic(err)
	}
	Downloads.Runs.Add(context.Background(), 0)

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
	Concat.Errors.Add(context.Background(), 0)
	Concat.Runs, err = meter.Int64Counter(
		"concat.runs",
		metric.WithDescription("Number of concats"),
	)
	if err != nil {
		panic(err)
	}
	Concat.Runs.Add(context.Background(), 0)

	// PostProcessing
	PostProcessing.CompletionTime, err = meter.Float64Histogram(
		"post_processing.completion.time",
		metric.WithDescription("Time taken to complete a post process"),
		metric.WithUnit("s"),
	)
	if err != nil {
		panic(err)
	}
	PostProcessing.Errors, err = meter.Int64Counter(
		"post_processing.errors",
		metric.WithDescription("Accumulated failed runs of post processes"),
	)
	if err != nil {
		panic(err)
	}
	PostProcessing.Errors.Add(context.Background(), 0)
	PostProcessing.Runs, err = meter.Int64Counter(
		"post_processing.runs",
		metric.WithDescription("Number of post processes"),
	)
	if err != nil {
		panic(err)
	}
	PostProcessing.Runs.Add(context.Background(), 0)

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
	Cleaner.FilesRemoved.Add(context.Background(), 0)
	Cleaner.Errors, err = meter.Int64Counter(
		"cleaner.errors",
		metric.WithDescription("Number of errors during cleaning"),
	)
	if err != nil {
		panic(err)
	}
	Cleaner.Errors.Add(context.Background(), 0)
	Cleaner.Runs, err = meter.Int64Counter(
		"cleaner.runs",
		metric.WithDescription("Number of cleaning runs"),
	)
	if err != nil {
		panic(err)
	}
	Cleaner.Runs.Add(context.Background(), 0)
	Cleaner.CleanTime, err = meter.Float64Histogram(
		"cleaner.clean.time",
		metric.WithDescription("Time taken to clean"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(err)
	}
}
