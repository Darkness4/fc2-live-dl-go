package watch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/cookie"
	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	configPath string
)

var Command = &cli.Command{
	Name:  "watch",
	Usage: "Automatically download multiple Live FC2 streams.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Aliases:     []string{"c"},
			Required:    true,
			Usage:       `Config file path. (required)`,
			Destination: &configPath,
		},
	},
	Action: func(cCtx *cli.Context) error {
		ctx, cancel := context.WithCancel(cCtx.Context)

		// Trap cleanup
		cleanChan := make(chan os.Signal, 1)
		signal.Notify(cleanChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-cleanChan
			cancel()
		}()

		configChan := make(chan *Config)
		go WatchConfig(ctx, configPath, configChan)

		return ConfigReloader(ctx, configChan, handleConfig)
	},
}

func ConfigReloader(ctx context.Context, configChan <-chan *Config, handleConfig func(ctx context.Context, config *Config)) error {
	var configContext context.Context
	var configCancel context.CancelFunc
	// Mutex to assure only one handleConfig can be launched
	var mu sync.Mutex

	for {
		select {
		case newConfig := <-configChan:
			if configContext != nil && configCancel != nil {
				configCancel()
			}
			configContext, configCancel = context.WithCancel(ctx)
			mu.Lock()
			go func() {
				logger.I.Info("config reloaded")
				handleConfig(configContext, newConfig)
				mu.Unlock()
			}()
		case <-ctx.Done():
			if configContext != nil && configCancel != nil {
				configCancel()
				configContext = nil
				configCancel = nil
			}
			// This assure that the `handleConfig` ends gracefully
			mu.Lock()
			mu.Unlock()
			// The context was canceled, exit the loop
			return ctx.Err()
		}
	}
}

func handleConfig(ctx context.Context, config *Config) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		logger.I.Panic("failed to initialize cookie jar", zap.Error(err))
	}

	params := fc2.DefaultParams.Clone()
	config.DefaultParams.Override(params)
	if params.CookiesFile != "" {
		if err := cookie.ParseFromFile(jar, params.CookiesFile); err != nil {
			logger.I.Error("failed to load cookies", zap.Error(err))
		}
	}

	client := &http.Client{Jar: jar, Timeout: time.Minute}

	var wg sync.WaitGroup
	wg.Add(len(config.Channels))
	for channel, overrideParams := range config.Channels {
		channelParams := params.Clone()
		overrideParams.Override(channelParams)

		go func(channel string, params *fc2.Params) {
			defer wg.Done()
			for {
				err := handleChannel(ctx, client, channel, params)
				if errors.Is(err, context.Canceled) {
					logger.I.Info("abort watching channel", zap.String("channelID", channel))
					return
				}
				if err != nil {
					logger.I.Error("failed to download", zap.Error(err))
				}
				time.Sleep(time.Second)
			}
		}(channel, channelParams)
	}

	wg.Wait()
}

func handleChannel(ctx context.Context, client *http.Client, channelID string, params *fc2.Params) error {
	downloader := fc2.NewDownloader(client, params)
	logger.I.Info("running", zap.Any("params", params))

	err := downloader.Download(ctx, channelID)
	if err == io.EOF {
		return nil
	}
	return err
}
