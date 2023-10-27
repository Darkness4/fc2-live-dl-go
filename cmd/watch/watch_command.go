package watch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/Darkness4/fc2-live-dl-go/notify"
	"github.com/Darkness4/fc2-live-dl-go/notify/notifier"
	"github.com/Darkness4/fc2-live-dl-go/state"
	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
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
				s := state.DefaultState.ReadState()
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
			log.Info().Str("listenAddress", pprofListenAddress).Msg("listening")
			if err := http.ListenAndServe(pprofListenAddress, nil); err != nil {
				log.Fatal().Err(err).Msg("fail to serve http")
			}
			log.Fatal().Msg("http server stopped")
		}()

		return ConfigReloader(ctx, configChan, handleConfig)
	},
}

func handleConfig(ctx context.Context, config *Config) {
	jar, err := cookiejar.New(&cookiejar.Options{})
	if err != nil {
		log.Panic().Err(err).Msg("failed to initialize cookie jar")
	}

	params := fc2.DefaultParams.Clone()
	config.DefaultParams.Override(params)
	if params.CookiesFile != "" {
		if err := cookie.ParseFromFile(jar, params.CookiesFile); err != nil {
			log.Error().Err(err).Msg("failed to load cookies, using unauthenticated")
		}
	}

	client := &http.Client{Jar: jar, Timeout: time.Minute}

	switch {
	case config.Notifier.Gotify.Enabled:
		notifier.Notifier = notify.NewGoNotifier(
			client,
			config.Notifier.Gotify.Endpoint,
			config.Notifier.Gotify.Token,
		)
		log.Info().Msg("using gotify")
	case config.Notifier.Shoutrrr.Enabled:
		notifier.Notifier = notify.NewShoutrrrNotifier(config.Notifier.Shoutrrr.URLs...)
		log.Info().Msg("using shoutrrr")
		if len(config.Notifier.Shoutrrr.URLs) == 0 {
			log.Warn().Msg("using shoutrrr but there is no URLs")
		}
	default:
		notifier.Notifier = notify.NewDummyNotifier()
		log.Info().Msg("no notifier configured")
	}
	if err := notifier.Notify(ctx, "config reloaded", "", 10); err != nil {
		log.Err(err).Msg("notify failed")
	}
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			if err := notifier.Notify(ctx, "panicked", fmt.Sprintf("%v", err), 10); err != nil {
				log.Err(err).Msg("notify failed")
			}
			os.Exit(1)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(config.Channels))
	for channel, overrideParams := range config.Channels {
		channelParams := params.Clone()
		overrideParams.Override(channelParams)

		go func(channelID string, params *fc2.Params) {
			defer wg.Done()
			log := log.With().Str("channelID", channelID).Logger()
			for {
				state.DefaultState.SetChannelState(channelID, state.DownloadStateIdle, nil)
				err := handleChannel(ctx, client, channelID, params)
				if errors.Is(err, context.Canceled) {
					log.Info().Msg("abort watching channel")
					if state.DefaultState.GetChannelState(
						channelID,
					) == state.DownloadStateDownloading {
						if err := notifier.Notify(
							context.Background(),
							fmt.Sprintf("stream download of the channel %s (%v) was canceled", channelID, utils.JSONMustEncode(params.Labels)),
							"",
							7,
						); err != nil {
							log.Err(err).Msg("notify failed")
						}
					}
					return
				} else if err != nil {
					log.Error().Err(err).Msg("failed to download")
					state.DefaultState.SetChannelError(channelID, err)
					if err := notifier.Notify(
						context.Background(),
						fmt.Sprintf("stream download of the channel %s (%v) failed", channelID, utils.JSONMustEncode(params.Labels)),
						err.Error(),
						10,
					); err != nil {
						log.Err(err).Msg("notify failed")
					}
				} else {
					if err := notifier.Notify(
						context.Background(),
						fmt.Sprintf("stream of the channel %s (%v) ended", channelID, utils.JSONMustEncode(params.Labels)),
						"",
						0,
					); err != nil {
						log.Err(err).Msg("notify failed")
					}
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
