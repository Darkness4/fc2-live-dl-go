// Package watch provides the watch command for watching multiple live FC2 streams.
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
	"strings"
	"sync"
	"syscall"
	"time"

	// Import the pprof package to enable profiling via HTTP.
	_ "net/http/pprof"
	// Import the godeltaprof package to enable continuous profiling via Pyroscope.
	_ "github.com/grafana/pyroscope-go/godeltaprof/http/pprof"

	"github.com/Darkness4/fc2-live-dl-go/cookie"
	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/fc2/cleaner"
	"github.com/Darkness4/fc2-live-dl-go/notify"
	"github.com/Darkness4/fc2-live-dl-go/notify/notifier"
	"github.com/Darkness4/fc2-live-dl-go/state"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// Hardcoded URL to check for new versions.
const versionCheckURL = "https://api.github.com/repos/Darkness4/fc2-live-dl-go/releases/latest"

var (
	configPath         string
	pprofListenAddress string
)

// Command is the command for watching multiple live FC2 streams.
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
		go ObserveConfig(ctx, configPath, configChan)

		go func() {
			http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
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

		return ConfigReloader(ctx, configChan, func(ctx context.Context, config *Config) {
			handleConfig(ctx, cCtx.App.Version, config)
		})
	},
}

func handleConfig(ctx context.Context, version string, config *Config) {
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
	if params.CookiesRefreshDuration != 0 && params.CookiesFile != "" {
		log.Info().Dur("duration", params.CookiesRefreshDuration).Msg("will refresh cookies")
		if err := fc2.Login(ctx, fc2.WithHTTPClient(client)); err != nil {
			if err := notifier.NotifyLoginFailed(ctx, err); err != nil {
				log.Err(err).Msg("notify failed")
			}
			log.Err(err).
				Msg("failed to login to id.fc2.com, we will try again, but you should extract new cookies")
		}
		go fc2.LoginLoop(ctx, params.CookiesRefreshDuration, fc2.WithHTTPClient(client))
	} else {
		log.Info().Msg("cookies refresh duration is zero, will not refresh cookies")
	}

	if config.Notifier.Enabled {
		notifier.Notifier = notify.NewFormatedNotifier(
			notify.NewShoutrrr(
				config.Notifier.URLs,
				notify.IncludeTitleInMessage(config.Notifier.IncludeTitleInMessage),
				notify.NoPriority(config.Notifier.NoPriority),
			),
			config.Notifier.NotificationFormats,
		)
		log.Info().Msg("using shoutrrr")
		if len(config.Notifier.URLs) == 0 {
			log.Warn().Msg("using shoutrrr but there is no URLs")
		}
	} else {
		log.Info().Msg("no notifier configured")
	}

	if err := notifier.NotifyConfigReloaded(ctx); err != nil {
		log.Err(err).Msg("notify failed")
	}

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			if err := notifier.NotifyPanicked(ctx, err); err != nil {
				log.Err(err).Msg("notify failed")
			}
			os.Exit(1)
		}
	}()

	// Check new version
	go checkVersion(ctx, client, version)

	var wg sync.WaitGroup
	wg.Add(len(config.Channels))
	for channel, overrideParams := range config.Channels {
		channelParams := params.Clone()
		overrideParams.Override(channelParams)

		// Scan for intermediates .ts used for concatenation
		if !channelParams.KeepIntermediates && channelParams.Concat &&
			channelParams.ScanDirectory != "" {
			wg.Add(1)
			go func(params *fc2.Params) {
				defer wg.Done()
				cleaner.CleanPeriodically(
					ctx,
					params.ScanDirectory,
					time.Hour,
					cleaner.WithEligibleAge(params.EligibleForCleaningAge),
				)
			}(channelParams)
		}

		go func(channelID string, params *fc2.Params) {
			defer wg.Done()
			log := log.With().Str("channelID", channelID).Logger()
			for {
				state.DefaultState.SetChannelState(channelID, state.DownloadStateIdle, nil)
				if err := notifier.NotifyIdle(ctx, channelID, params.Labels); err != nil {
					log.Err(err).Msg("notify failed")
				}

				meta, err := handleChannel(ctx, client, channelID, params)
				if errors.Is(err, context.Canceled) {
					log.Info().Msg("abort watching channel")
					if state.DefaultState.GetChannelState(
						channelID,
					) != state.DownloadStateIdle {
						state.DefaultState.SetChannelState(
							channelID,
							state.DownloadStateCanceled,
							nil,
						)
						if err := notifier.NotifyCanceled(
							context.Background(),
							channelID,
							params.Labels,
						); err != nil {
							log.Err(err).Msg("notify failed")
						}
					}
					return
				} else if err != nil {
					log.Error().Err(err).Msg("failed to download")
					state.DefaultState.SetChannelError(channelID, err)
					if err := notifier.NotifyError(
						context.Background(),
						channelID,
						params.Labels,
						err,
					); err != nil {
						log.Err(err).Msg("notify failed")
					}
				} else {
					state.DefaultState.SetChannelState(channelID, state.DownloadStateFinished, nil)
					if err := notifier.NotifyFinished(ctx, channelID, params.Labels, meta); err != nil {
						log.Err(err).Msg("notify failed")
					}
				}
				time.Sleep(time.Second)
			}
		}(channel, channelParams)
	}

	wg.Wait()
}

func checkVersion(ctx context.Context, client *http.Client, version string) {
	if strings.Contains(version, "-") { // Version containing a hyphen is a development version.
		log.Warn().Str("version", version).Msg("development version, skipping version check")
		return
	}

	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionCheckURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to create request")
		return
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("failed to check version")
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Str("status", resp.Status).Msg("failed to check version")
		return
	}

	var data struct {
		TagName string `json:"tag_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Error().Err(err).Msg("failed to decode version")
		return
	}

	if data.TagName != version {
		log.Warn().Str("latest", data.TagName).Str("current", version).Msg("new version available")
		if err := notifier.NotifyUpdateAvailable(ctx, data.TagName); err != nil {
			log.Err(err).Msg("notify failed")
		}
	}
}

func handleChannel(
	ctx context.Context,
	client *http.Client,
	channelID string,
	params *fc2.Params,
) (*fc2.GetMetaData, error) {
	downloader := fc2.New(client, params, channelID)

	meta, err := downloader.Watch(ctx)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return meta, nil
}
