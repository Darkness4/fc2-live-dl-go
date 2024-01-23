package cleaner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/video/probe"
	"github.com/rs/zerolog/log"
)

// cleanerMutex is used to avoid multiple clean in parallel.
//
// Less stress for CPU, and avoid risks of race condition.
var cleanerMutex sync.Mutex

type Option func(*Options)

type Options struct {
	dryRun      bool
	probe       bool
	eligibleAge time.Duration
}

func WithDryRun() Option {
	return func(o *Options) {
		o.dryRun = true
	}
}

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

func Scan(
	scanDirectory string,
	opts ...Option,
) (queueForDeletion []string, queueForRenaming []string, err error) {
	o := applyOptions(opts)

	set := make(map[string]bool)
	queueForRenaming = make([]string, 0)

	if err := filepath.WalkDir(scanDirectory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			if strings.HasSuffix(name, ".combined") {
				prefix := strings.TrimSuffix(name, ".combined")
				dir := filepath.Dir(path)

				finfo, err := d.Info()
				if err != nil {
					return err
				}

				// Check if file is eligible for cleaning.
				if time.Since(finfo.ModTime()) <= o.eligibleAge {
					return nil
				}

				// Check if file is a video
				if o.probe {
					if isVideo, err := probe.ContainsVideoOrAudio(path); err != nil {
						log.Err(err).Str("path", path).Msg("deletion skipped due to error")
						return nil
					} else if !isVideo {
						return nil
					}
				}

				// Look for .TS files with the same prefix.
				entries, err := os.ReadDir(dir)
				if err != nil {
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
		return []string{}, []string{}, nil
	}

	queue := make([]string, 0, len(set))
	for k := range set {
		queue = append(queue, k)
	}
	return queue, queueForRenaming, nil
}

func Clean(scanDirectory string, opts ...Option) error {
	cleanerMutex.Lock()
	defer cleanerMutex.Unlock()

	o := applyOptions(opts)

	queueForDeletion, queueForRenaming, err := Scan(scanDirectory, opts...)
	if err != nil {
		return err
	}

	for _, path := range queueForDeletion {
		log.Info().Str("path", path).Msg("deleting old .ts file")
		if !o.dryRun {
			if err := os.Remove(path); err != nil {
				log.Err(err).Str("path", path).Msg("failed to delete old .ts file, skipping...")
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
