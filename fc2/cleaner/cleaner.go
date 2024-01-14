package cleaner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type Option func(*Options)

type Options struct {
	dryRun bool
}

func WithDryRun() Option {
	return func(o *Options) {
		o.dryRun = true
	}
}

func applyOptions(opts []Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func Scan(scanDirectory string) ([]string, error) {
	set := make(map[string]bool)

	if err := filepath.WalkDir(scanDirectory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			if strings.HasSuffix(name, ".combined") {
				prefix := strings.TrimSuffix(name, ".combined")
				dir := filepath.Dir(path)

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
						finfo, err := entry.Info()
						if err != nil {
							return err
						}

						if time.Since(finfo.ModTime()) > 48*time.Hour {
							set[filepath.Join(dir, entry.Name())] = true
						}
					}
				}
			}
		}

		return nil
	}); err != nil {
		return []string{}, nil
	}

	queue := make([]string, 0, len(set))
	for k := range set {
		queue = append(queue, k)
	}
	return queue, nil
}

func Clean(scanDirectory string, opts ...Option) error {
	o := applyOptions(opts)

	queue, err := Scan(scanDirectory)
	if err != nil {
		return err
	}

	for _, path := range queue {
		log.Info().Str("path", path).Msg("deleting old .ts file")
		if !o.dryRun {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}

	return nil
}
