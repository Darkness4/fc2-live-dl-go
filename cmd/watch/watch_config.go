package watch

import (
	"context"
	"os"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/notify"
	"github.com/Darkness4/fc2-live-dl-go/utils/channel"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Notifier      NotifierConfig                `yaml:"notifier,omitempty"`
	DefaultParams fc2.OptionalParams            `yaml:"defaultParams,omitempty"`
	Channels      map[string]fc2.OptionalParams `yaml:"channels,omitempty"`
}

type NotifierConfig struct {
	Enabled                    bool     `yaml:"enabled,omitempty"`
	IncludeTitleInMessage      bool     `yaml:"includeTitleInMessage,omitempty"`
	NoPriority                 bool     `yaml:"noPriority,omitempty"`
	URLs                       []string `yaml:"urls,omitempty"`
	notify.NotificationFormats `yaml:"notificationFormats,omitempty"`
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
			log.Error().Str("file", filename).Err(err).Msg("failed to stat file")
			return
		}
		lastModTime = stat.ModTime()

		log.Info().Msg("initial config detected")
		config, err := loadConfig(filename)
		if err != nil {
			log.Error().Str("file", filename).Err(err).Msg("failed to load config")
			return
		}

		configChan <- config
	}()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Panic().Err(err).Msg("failed to watch config")
	}
	defer watcher.Close()

	if err = watcher.Add(filename); err != nil {
		log.Panic().Err(err).Msg("failed to add config to config reloader")
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
				log.Error().Str("file", filename).Err(err).Msg("failed to stat file")
				continue
			}

			if !stat.ModTime().Equal(lastModTime) {
				lastModTime = stat.ModTime()
				log.Info().Msg("new config detected")

				config, err := loadConfig(filename)
				if err != nil {
					log.Error().Str("file", filename).Err(err).Msg("failed to load config")
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
			log.Error().Str("file", filename).Err(err).Msg("config reloader thrown an error")
		}
	}
}

func ConfigReloader(
	ctx context.Context,
	configChan <-chan *Config,
	handleConfig func(ctx context.Context, config *Config),
) error {
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
					log.Info().Msg("loading new config")
				case <-time.After(30 * time.Second):
					log.Fatal().Msg("couldn't load a new config because of a deadlock")
				}
			}
			configContext, configCancel = context.WithCancel(ctx)
			go func() {
				log.Info().Msg("loaded new config")
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
				log.Info().Msg("config reloader graceful exit")
			case <-time.After(30 * time.Second):
				log.Fatal().Msg("config reloader force fatal exit")
			}

			// The context was canceled, exit the loop
			return ctx.Err()
		}
	}
}
