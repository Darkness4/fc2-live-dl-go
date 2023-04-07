package watch

import (
	"context"
	"os"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/Darkness4/fc2-live-dl-go/utils/channel"
	"github.com/fsnotify/fsnotify"
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

func WatchConfig(ctx context.Context, filename string, configChan chan<- *Config) {
	var lastModTime time.Time

	// Initial load
	func() {
		stat, err := os.Stat(filename)
		if err != nil {
			logger.I.Error("failed to stat file", zap.Error(err), zap.String("file", filename))
			return
		}
		lastModTime = stat.ModTime()

		logger.I.Info("initial config detected")
		config, err := loadConfig(filename)
		if err != nil {
			logger.I.Error("failed to load config", zap.Error(err), zap.String("file", filename))
			return
		}

		configChan <- config
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.I.Panic("failed to watch config", zap.Error(err))
	}
	defer watcher.Close()

	if err = watcher.Add(filename); err != nil {
		logger.I.Panic("failed to add config to config reloader", zap.Error(err))
	}

	debouncedEvents := channel.Debounce(watcher.Events, time.Second)

	for {
		select {
		case <-ctx.Done():
			// The parent context was canceled, exit the loop
			return
		case _, ok := <-debouncedEvents:
			if !ok {
				return
			}
			stat, err := os.Stat(filename)
			if err != nil {
				logger.I.Error("failed to stat file", zap.Error(err), zap.String("file", filename))
				continue
			}

			if !stat.ModTime().Equal(lastModTime) {
				lastModTime = stat.ModTime()
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

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.I.Error("config reloader thrown an error", zap.Error(err), zap.String("file", filename))
		}
	}
}

func ConfigReloader(ctx context.Context, configChan <-chan *Config, handleConfig func(ctx context.Context, config *Config)) error {
	var configContext context.Context
	var configCancel context.CancelFunc
	// Channel used to assure only one handleConfig can be launched
	doneChan := make(chan struct{})

	for {
		select {
		case newConfig := <-configChan:
			if configContext != nil && configCancel != nil {
				configCancel()
				select {
				case <-doneChan:
					logger.I.Info("loading new config")
				case <-time.After(30 * time.Second):
					logger.I.Fatal("couldn't load a new config because of a deadlock")
				}
			}
			configContext, configCancel = context.WithCancel(ctx)
			go func() {
				logger.I.Info("loaded new config")
				handleConfig(configContext, newConfig)
				doneChan <- struct{}{}
			}()
		case <-ctx.Done():
			if configContext != nil && configCancel != nil {
				configCancel()
				configContext = nil
			}

			// This assure that the `handleConfig` ends gracefully
			select {
			case <-doneChan:
				logger.I.Info("config reloader graceful exit")
			case <-time.After(30 * time.Second):
				logger.I.Fatal("config reloader force fatal exit")
			}

			// The context was canceled, exit the loop
			return ctx.Err()
		}
	}
}
