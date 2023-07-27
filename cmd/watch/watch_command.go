package watch

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/Darkness4/fc2-live-dl-go/cookie"
	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/Darkness4/fc2-live-dl-go/state"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	configPath         string
	pprofListenAddress string
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
		&cli.StringFlag{
			Name:        "pprof.listen-address",
			Value:       ":3000",
			Destination: &pprofListenAddress,
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

		go func() {
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				s := state.ReadState()
				b, err := json.MarshalIndent(s, "", "  ")
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				_, err = w.Write(b)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			})
			logger.I.Info("listening", zap.String("listenAddress", pprofListenAddress))
			if err := http.ListenAndServe(pprofListenAddress, nil); err != nil {
				logger.I.Fatal("fail to serve http", zap.Error(err))
			}
			logger.I.Fatal("http server stopped")
		}()

		return ConfigReloader(ctx, configChan, handleConfig)
	},
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

		go func(channelID string, params *fc2.Params) {
			defer wg.Done()
			log := logger.I.With(zap.String("channelID", channelID))
			for {
				state.SetChannelState(channelID, state.DownloadStateIdle)
				err := handleChannel(ctx, client, channelID, params)
				if errors.Is(err, context.Canceled) {
					log.Info("abort watching channel")
					state.SetChannelError(channelID, nil)
					return
				} else if err == fc2.ErrWebSocketStreamEnded {
					log.Info("stream ended")
					state.SetChannelError(channelID, nil)
				} else if err != nil {
					log.Error("failed to download", zap.Error(err))
					state.SetChannelError(channelID, err)
				}
				time.Sleep(time.Second)
			}
		}(channel, channelParams)
	}

	wg.Wait()
}

func handleChannel(
	ctx context.Context,
	client *http.Client,
	channelID string,
	params *fc2.Params,
) error {
	downloader := fc2.New(client, params, channelID)

	if err := downloader.Watch(ctx); err != nil && err != io.EOF {
		return err
	}
	return nil
}
