// Package watch provides the watch command for watching multiple live FC2 streams.
package watch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	// Import the pprof package to enable profiling via HTTP.
	"net/http/cookiejar"
	_ "net/http/pprof"

	// Import the godeltaprof package to enable continuous profiling via Pyroscope.
	_ "github.com/grafana/pyroscope-go/godeltaprof/http/pprof"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/Darkness4/fc2-live-dl-go/cookie"
	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/Darkness4/fc2-live-dl-go/fc2/cleaner"
	"github.com/Darkness4/fc2-live-dl-go/notify"
	"github.com/Darkness4/fc2-live-dl-go/notify/notifier"
	"github.com/Darkness4/fc2-live-dl-go/state"
	"github.com/Darkness4/fc2-live-dl-go/telemetry"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// Hardcoded URL to check for new versions.
const versionCheckURL = "https://api.github.com/repos/Darkness4/fc2-live-dl-go/releases/latest"

var (
	configPath             string
	pprofListenAddress     string
	enableTracesExporting  bool
	enableMetricsExporting bool
	cookieEncryptionSecret string
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
			Usage:       "The address to listen on for pprof.",
			EnvVars:     []string{"PPROF_LISTEN_ADDRESS"},
		},
		&cli.StringFlag{
			Name:        "cookie.encryption-secret",
			Value:       "FC2_LIVE_DL_GO_COOKIE_ENCRYPTION_SECRET",
			Destination: &cookieEncryptionSecret,
			Usage:       "A encryption secret to encrypt the cookies.",
			EnvVars:     []string{"COOKIE_ENCRYPTION_SECRET"},
		},
		&cli.BoolFlag{
			Name:        "traces.export",
			Usage:       "Enable traces push. (To configure the exporter, set the OTEL_EXPORTER_OTLP_ENDPOINT environment variable, see https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/)",
			Value:       false,
			Destination: &enableTracesExporting,
			EnvVars:     []string{"OTEL_EXPORTER_OTLP_TRACES_ENABLED"},
		},
		&cli.BoolFlag{
			Name:        "metrics.export",
			Usage:       "Enable metrics push. (To configure the exporter, set the OTEL_EXPORTER_OTLP_ENDPOINT environment variable, see https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/). Note that a Prometheus path is already exposed at /metrics.",
			Value:       false,
			Destination: &enableMetricsExporting,
			EnvVars:     []string{"OTEL_EXPORTER_OTLP_METRICS_ENABLED"},
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

		// Setup telemetry
		prom, err := prometheus.New()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create prometheus exporter")
		}

		telOpts := []telemetry.Option{
			telemetry.WithMetricReader(prom),
		}

		if enableMetricsExporting {
			metricExporter, err := otlpmetricgrpc.New(ctx)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to create OTEL metric exporter")
			}
			telOpts = append(telOpts, telemetry.WithMetricExporter(metricExporter))
		}

		if enableTracesExporting {
			traceExporter, err := otlptracegrpc.New(ctx)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to create OTEL trace exporter")
			}
			telOpts = append(telOpts, telemetry.WithTraceExporter(traceExporter))
		}

		shutdown, err := telemetry.SetupOTELSDK(ctx,
			telOpts...,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to setup OTEL SDK")
		}
		defer func() {
			if err := shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Error().Err(err).Msg("failed to shutdown OTEL SDK")
			}
		}()

		configChan := make(chan *Config)
		go ObserveConfig(ctx, configPath, configChan)

		go func() {
			http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				s := state.DefaultState.ReadState()
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				if err := enc.Encode(s); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			})
			http.Handle("/metrics", promhttp.Handler())
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

type PersistentCookieJar interface {
	http.CookieJar

	Exists() bool
	Save() error
	Delete()
}

type noPersistCookieJar struct {
	http.CookieJar
}

func (j *noPersistCookieJar) Exists() bool {
	return false
}

func (j *noPersistCookieJar) Save() error {
	return nil
}

func (j *noPersistCookieJar) Delete() {}

func handleConfig(ctx context.Context, version string, config *Config) {
	var jar PersistentCookieJar
	var err error
	if config.CookiesFile != "" {
		jar, err = cookie.NewJar(config.CookiesFile, &cookie.JarOptions{
			EncryptionSecret: cookieEncryptionSecret,
		})
		if err != nil {
			log.Fatal().Err(err).Msg("failed to initialize cookie jar")
		}
	} else {
		ijar, err := cookiejar.New(&cookiejar.Options{})
		if err != nil {
			// Panic here since it's unexpected
			log.Panic().Err(err).Msg("failed to initialize cookie jar")
		}
		jar = &noPersistCookieJar{ijar}
	}

	params := fc2.DefaultParams.Clone()
	config.DefaultParams.Override(&params)

	// Handle deprecated parameters
	if config.DefaultParams.CookiesFile != nil && *config.DefaultParams.CookiesFile != "" {
		config.CookiesImportFile = params.CookiesFile

		log.Warn().
			Msg("defaultParams.cookiesFile is deprecated, please use top-level cookiesImportFile instead")
	}
	if config.DefaultParams.CookiesRefreshDuration != nil &&
		*config.DefaultParams.CookiesRefreshDuration != 0 {
		config.CookiesRefreshDuration = params.CookiesRefreshDuration

		log.Warn().
			Msg("defaultParams.cookiesRefreshDuration is deprecated, please use top-level cookiesRefreshDuration instead")
	}

	if !jar.Exists() {
		if config.CookiesImportFile != "" && !jar.Exists() {
			if err := cookie.ParseFromFile(jar, config.CookiesImportFile); err != nil {
				log.Error().Err(err).Msg("failed to load cookies, using unauthenticated")
			} else {
				log.Info().Str("file", config.CookiesImportFile).Msg("loaded cookies")
			}
		}
	} else {
		log.Info().Str("file", config.CookiesFile).Msg("loaded persisted cookies")
	}

	hclient := &http.Client{
		Jar:     jar,
		Timeout: time.Minute,
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithTracerProvider(noop.NewTracerProvider()),
		),
	}
	client := api.NewClient(hclient)
	if config.CookiesRefreshDuration != 0 && config.CookiesImportFile != "" {
		log.Info().Dur("duration", config.CookiesRefreshDuration).Msg("will refresh cookies")
		if err := client.Login(ctx); err != nil {
			log.Err(err).
				Msg("failed to login to id.fc2.com, we will try again, but you should extract new cookies")
			jar.Delete()
		}
		if err := jar.Save(); err != nil {
			log.Err(err).Msg("failed to save cookies")
		}
		go LoginLoop(ctx, client, jar, config.CookiesRefreshDuration)
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
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			if err := notifier.NotifyPanicked(ctx, err); err != nil {
				log.Err(err).Msg("notify failed")
			}
			os.Exit(1)
		}
	}()

	// Check new version
	go checkVersion(ctx, hclient, version)

	var wg sync.WaitGroup
	wg.Add(len(config.Channels))
	for channel, overrideParams := range config.Channels {
		channelParams := params.Clone()
		overrideParams.Override(&channelParams)

		// Scan for intermediates .ts used for concatenation
		if !channelParams.KeepIntermediates && channelParams.Concat &&
			channelParams.ScanDirectory != "" {
			wg.Add(1)
			go func(params fc2.Params) {
				defer wg.Done()
				cleaner.CleanPeriodically(
					ctx,
					params.ScanDirectory,
					time.Hour,
					cleaner.WithEligibleAge(params.EligibleForCleaningAge),
				)
			}(channelParams)
		}

		go func(channelID string, params fc2.Params) {
			defer wg.Done()
			err := fc2.New(client, params, channelID).Watch(ctx)
			if err != nil && err != io.EOF {
				log.Err(err).Str("channelID", channelID).Msg("failed to download")
			}

			select {
			case <-ctx.Done():
				return
			default:
				log.Panic().
					Err(err).
					Str("channelID", channelID).
					Msg("stopped watching channel without parent context being canceled")
			}
		}(channel, channelParams)

		// Spread out the channel start time to avoid hammering the server.
		time.Sleep(200 * time.Millisecond)
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

// LoginLoop will try to login to FC2 every duration.
func LoginLoop(
	ctx context.Context,
	c *api.Client,
	jar PersistentCookieJar,
	duration time.Duration,
) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.Login(ctx); err != nil {
				if err := notifier.NotifyLoginFailed(ctx, err); err != nil {
					log.Err(err).Msg("notify failed")
				}
				log.Err(err).
					Msg("failed to login to id.fc2.com, we will try again, but you should extract new cookies")
				jar.Delete()
			} else {
				if err := jar.Save(); err != nil {
					log.Err(err).Msg("failed to save cookies")
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
