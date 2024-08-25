// Package cleaner provides functions to clean old .ts files.
package cleaner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/telemetry/metrics"
	"github.com/Darkness4/fc2-live-dl-go/video/probe"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "fc2/cleaner"

// cleanerMutex is used to avoid multiple clean in parallel.
//
// Less stress for CPU, and avoid risks of race condition.
var cleanerMutex sync.Mutex

// Option is the option for the cleaner.
type Option func(*Options)

// Options are the options for the cleaner.
type Options struct {
	dryRun      bool
	probe       bool
	eligibleAge time.Duration
}

// WithDryRun sets the dryRun option.
func WithDryRun() Option {
	return func(o *Options) {
		o.dryRun = true
	}
}

// WithoutProbe disables the probe.
func WithoutProbe() Option {
	return func(o *Options) {
		o.probe = false
	}
}

// WithEligibleAge sets the minimum time since the modtime of the file to be deleted.
func WithEligibleAge(d time.Duration) Option {
	return func(o *Options) {
		if d < time.Hour {
			log.Warn().
				Dur("eligibleAge", d).
				Msg("You've set an 'eligible to cleaning' age < 1 hour. You should set an age greater than the duration of the streams.")
		}
		if d != 0 {
			o.eligibleAge = d
		}
	}
}

func applyOptions(opts []Option) *Options {
	o := &Options{
		probe:       true,
		eligibleAge: 48 * time.Hour,
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Scan scans the scanDirectory for old .ts files.
func Scan(
	scanDirectory string,
	opts ...Option,
) (queueForDeletion []string, queueForRenaming []string, err error) {
	_, span := otel.Tracer(tracerName).Start(context.Background(), "cleaner.Scan")
	defer span.End()
	metrics.Cleaner.Runs.Add(context.Background(), 1)

	o := applyOptions(opts)

	set := make(map[string]bool)
	queueForRenaming = make([]string, 0)

	if err := filepath.WalkDir(scanDirectory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Err(err).Str("path", path).Msg("failed to walk directory")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		if !d.IsDir() {
			name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			if strings.HasSuffix(name, ".combined") {
				prefix := strings.TrimSuffix(name, ".combined")
				dir := filepath.Dir(path)

				finfo, err := d.Info()
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return err
				}

				// Check if file is eligible for cleaning.
				if time.Since(finfo.ModTime()) <= o.eligibleAge {
					return nil
				}

				// Check if file is a video
				if o.probe {
					if isVideo, err := probe.ContainsVideoOrAudio(path); err != nil {
						if !strings.Contains(err.Error(), "Invalid data found when processing input") {
							span.RecordError(err)
							span.SetStatus(codes.Error, err.Error())
							return nil
						} else {
							// File is corrupted, delete it.
							log.Err(err).Str("path", path).Msg("file is corrupted, deleting...")
							if !o.dryRun {
								if err := os.Remove(path); err != nil {
									log.Err(err).Str("path", path).Msg("failed to delete corrupted file")
								}
							}
							return nil
						}

					} else if !isVideo {
						return nil
					}
				}

				// Look for .TS files with the same prefix.
				entries, err := os.ReadDir(dir)
				if err != nil {
					log.Err(err).Str("dir", dir).Msg("failed to read directory")
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())
					return err
				}

				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), prefix+".") &&
						strings.HasSuffix(entry.Name(), ".ts") &&
						!strings.Contains(entry.Name(), ".combined.") &&
						!entry.IsDir() {

						fpath := filepath.Join(dir, entry.Name())
						set[fpath] = true
					}
				}

				queueForRenaming = append(queueForRenaming, path)
			}
		}

		return nil
	}); err != nil {
		log.Err(err).Str("scan_directory", scanDirectory).Msg("failed to scan directory")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return []string{}, []string{}, err
	}

	queue := make([]string, 0, len(set))
	for k := range set {
		queue = append(queue, k)
	}
	return queue, queueForRenaming, nil
}

// Clean removes old .ts files from the scanDirectory.
func Clean(scanDirectory string, opts ...Option) error {
	cleanerMutex.Lock()
	defer cleanerMutex.Unlock()

	o := applyOptions(opts)

	attrs := []attribute.KeyValue{
		attribute.String("scan_directory", scanDirectory),
		attribute.Bool("dry_run", o.dryRun),
		attribute.Bool("probe", o.probe),
		attribute.Float64("eligible_age", o.eligibleAge.Seconds()),
	}

	_, span := otel.Tracer(tracerName).
		Start(context.Background(), "cleaner.Clean", trace.WithAttributes(attrs...))
	defer span.End()

	end := metrics.TimeStartRecording(
		context.Background(),
		metrics.Cleaner.CleanTime,
		time.Millisecond,
		metric.WithAttributes(attrs...),
	)
	defer end()

	queueForDeletion, queueForRenaming, err := Scan(scanDirectory, opts...)
	if err != nil {
		log.Err(err).Msg("failed to scan directory")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for _, path := range queueForDeletion {
		log.Info().Str("path", path).Msg("deleting old .ts file")
		if !o.dryRun {
			if err := os.Remove(path); err != nil {
				log.Err(err).Str("path", path).Msg("failed to delete old .ts file, skipping...")
			} else {
				metrics.Cleaner.FilesRemoved.Add(context.Background(), 1)
			}
		}
	}

	for _, path := range queueForRenaming {
		ext := filepath.Ext(path)
		prefix := strings.TrimSuffix(path, ext)
		renamedPath := strings.TrimSuffix(prefix, ".combined") + ext

		// Check for conflict
		if _, err := os.Stat(renamedPath); err == nil {
			log.Info().Str("path", path).Str("to", renamedPath).Msg("cannot rename, file exists")
			continue
		}

		log.Info().Str("path", path).Str("to", renamedPath).Msg("renaming combined file")
		if !o.dryRun {
			if err := os.Rename(path, renamedPath); err != nil {
				log.Err(err).
					Str("path", path).
					Str("to", renamedPath).
					Msg("failed old .ts file, skipping...")
			}
		}
	}
	return nil
}

// CleanPeriodically runs the Clean function periodically.
func CleanPeriodically(
	ctx context.Context,
	scanDirectory string,
	interval time.Duration,
	opts ...Option,
) {
	log.Debug().Msg("scanning for old .ts to be deleted")
	if err := Clean(scanDirectory, opts...); err != nil {
		log.Err(err).Msg("failed to cleanup .ts files")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context cancelled, exit the goroutine
			log.Err(ctx.Err()).Msg("context cancelled, stopping cleaner")
			return
		case <-ticker.C:
			// Execute the cleanup routine
			go func() {
				log.Debug().Msg("scanning for old .ts to be deleted")
				if err := Clean(scanDirectory, opts...); err != nil {
					log.Err(err).Msg("failed to cleanup .ts files")
				}
			}()
		}
	}
}
