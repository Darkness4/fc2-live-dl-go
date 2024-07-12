// Package telemetry provides a simple way to set up OpenTelemetry SDK.
//
// nolint: ireturn
package telemetry

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Option is a function that configures the OTEL SDK.
type Option func(*options)

type options struct {
	stdout         bool
	traceExporter  trace.SpanExporter
	metricExporter metric.Exporter
	metricReader   metric.Reader
}

// WithStdout sets the exporters to stdout.
func WithStdout() Option {
	return func(o *options) {
		o.stdout = true
	}
}

// WithTraceExporter sets the trace exporter.
func WithTraceExporter(exporter trace.SpanExporter) Option {
	return func(o *options) {
		o.traceExporter = exporter
	}
}

// WithMetricExporter sets the metric exporter.
func WithMetricExporter(exporter metric.Exporter) Option {
	return func(o *options) {
		o.metricExporter = exporter
	}
}

// WithMetricReader sets the metric reader.
func WithMetricReader(reader metric.Reader) Option {
	return func(o *options) {
		o.metricReader = reader
	}
}

func applyOptions(opts []Option) *options {
	opt := &options{}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

// SetupOTELSDK sets up the OpenTelemetry SDK.
func SetupOTELSDK(
	ctx context.Context,
	opts ...Option,
) (
	shutdown func(context.Context) error,
	err error,
) {
	o := applyOptions(opts)
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	propagator := newPropagator()
	otel.SetTextMapPropagator(propagator)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider(o)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider, err := newMeterProvider(o)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider(o *options) (*trace.TracerProvider, error) {
	var opts []trace.TracerProviderOption
	if o.stdout {
		traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		opts = append(opts, trace.WithBatcher(traceExporter))
	}
	if o.traceExporter != nil {
		opts = append(opts, trace.WithBatcher(o.traceExporter))
	}
	traceProvider := trace.NewTracerProvider(opts...)
	return traceProvider, nil
}

func newMeterProvider(o *options) (*metric.MeterProvider, error) {
	var opts []metric.Option
	if o.stdout {
		metricExporter, err := stdoutmetric.New()
		if err != nil {
			return nil, err
		}
		opts = append(opts, metric.WithReader(metric.NewPeriodicReader(metricExporter)))
	}
	if o.metricExporter != nil {
		opts = append(opts, metric.WithReader(metric.NewPeriodicReader(o.metricExporter)))
	}
	if o.metricReader != nil {
		opts = append(opts, metric.WithReader(o.metricReader))
	}
	meterProvider := metric.NewMeterProvider(opts...)
	return meterProvider, nil
}
