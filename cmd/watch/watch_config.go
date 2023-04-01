package watch

import (
	"context"
	"os"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/logger"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultParams fc2.OptionalParams            `yaml:"defaultParams"`
	Channels      map[string]fc2.OptionalParams `yaml:"channels"`
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}
	return config, err
}

func observeConfig(ctx context.Context, filename string, configChan chan<- *Config) {
	var lastModTime time.Time
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ctx.Done():
			// The parent context was canceled, exit the loop
			return
		case <-ticker.C:
			fileinfo, err := os.Stat(filename)
			if err != nil {
				logger.I.Error("failed to stat file", zap.Error(err), zap.String("file", filename))
				continue
			}

			modTime := fileinfo.ModTime()
			if !lastModTime.IsZero() && modTime.After(lastModTime) {
				logger.I.Info("new config detected")

				config, err := loadConfig(filename)
				if err != nil {
					logger.I.Error("failed to load config", zap.Error(err), zap.String("file", filename))
					continue
				}

				select {
				case configChan <- config:
					// Config sent successfully
				case <-ctx.Done():
					// The parent context was canceled, exit the loop
					return
				}
			}

			lastModTime = modTime
		}
	}
}
