package download

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/cookie"
	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/Darkness4/fc2-live-dl-go/utils/try"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	downloadParams = fc2.Params{}
	maxTries       int
	loop           bool
)

var Command = &cli.Command{
	Name:      "download",
	Usage:     "Download a Live FC2 stream.",
	ArgsUsage: "channelID",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:       "quality",
			Value:      "1.2Mbps",
			HasBeenSet: true,
			Usage: `Quality of the stream to download.
Available latency options: 150Kbps, 400Kbps, 1.2Mbps, 2Mbps, 3Mbps, sound.`,
			Action: func(ctx *cli.Context, s string) error {
				downloadParams.Quality = fc2.QualityParseString(s)
				if downloadParams.Quality == fc2.QualityUnknown {
					logger.I.Error("Unknown input quality", zap.String("quality", s))
					return errors.New("unknown quality")
				}
				return nil
			},
		},
		&cli.StringFlag{
			Name:       "latency",
			Value:      "mid",
			HasBeenSet: true,
			Usage: `Stream latency. Select a higher latency if experiencing stability issues.
Available latency options: low, high, mid.`,
			Action: func(ctx *cli.Context, s string) error {
				downloadParams.Latency = fc2.LatencyParseString(s)
				if downloadParams.Latency == fc2.LatencyUnknown {
					logger.I.Error("Unknown input latency", zap.String("latency", s))
					return errors.New("unknown latency")
				}
				return nil
			},
		},
		&cli.StringFlag{
			Name:  "format",
			Value: "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}",
			Usage: `Golang templating format. Available fields: ChannelID, ChannelName, Date, Time, Title, Ext, Labels.Key.
Available format options:
  ChannelID: ID of the broadcast
  ChannelName: broadcaster's profile name
  Date: local date YYYY-MM-DD
  Time: local time HHMMSS
  Ext: file extension
  Title: title of the live broadcast
  Labels.Key: custom labels
`,
			Destination: &downloadParams.OutFormat,
		},
		&cli.IntFlag{
			Name:        "max-packet-loss",
			Value:       200,
			Usage:       "Allow a maximum of packet loss before aborting stream download.",
			Destination: &downloadParams.PacketLossMax,
		},
		&cli.BoolFlag{
			Name:       "no-remux",
			Value:      false,
			HasBeenSet: true,
			Usage:      "Do not remux recordings into mp4/m4a after it is finished.",
			Action: func(ctx *cli.Context, b bool) error {
				downloadParams.Remux = !b
				return nil
			},
		},
		&cli.BoolFlag{
			Name:        "keep-intermediates",
			Value:       false,
			Usage:       "Keep the raw .ts recordings after it has been remuxed.",
			Aliases:     []string{"k"},
			Destination: &downloadParams.KeepIntermediates,
		},
		&cli.BoolFlag{
			Name:        "extract-audio",
			Value:       false,
			Usage:       "Generate an audio-only copy of the stream.",
			Aliases:     []string{"x"},
			Destination: &downloadParams.ExtractAudio,
		},
		&cli.PathFlag{
			Name:        "cookies-file",
			Usage:       "Path to a cookies file. Format is a netscape cookies file.",
			Destination: &downloadParams.CookiesFile,
		},
		&cli.BoolFlag{
			Name:        "write-chat",
			Value:       false,
			Usage:       "Save live chat into a json file.",
			Destination: &downloadParams.WriteChat,
		},
		&cli.BoolFlag{
			Name:        "write-info-json",
			Value:       false,
			Usage:       "Dump output stream information into a json file.",
			Destination: &downloadParams.WriteInfoJSON,
		},
		&cli.BoolFlag{
			Name:        "write-thumbnail",
			Value:       false,
			Usage:       "Download thumbnail into a file.",
			Destination: &downloadParams.WriteThumbnail,
		},
		&cli.BoolFlag{
			Name:       "no-wait",
			Value:      false,
			HasBeenSet: true,
			Usage:      "Don't wait until the broadcast goes live, then start recording.",
			Action: func(ctx *cli.Context, b bool) error {
				downloadParams.WaitForLive = !b
				return nil
			},
		},
		&cli.IntFlag{
			Name:        "wait-for-quality-max-tries",
			Value:       10,
			Usage:       "If the requested quality is not available, keep retrying before falling back to the next best quality.",
			Destination: &downloadParams.WaitForQualityMaxTries,
		},
		&cli.DurationFlag{
			Name:        "poll-interval",
			Value:       5 * time.Second,
			Usage:       "How many seconds between checks to see if broadcast is live.",
			Destination: &downloadParams.WaitPollInterval,
		},
		&cli.IntFlag{
			Name:        "max-tries",
			Value:       10,
			Usage:       "On failure, keep retrying (cancellation and end of stream will still force abort).",
			Destination: &maxTries,
		},
		&cli.BoolFlag{
			Name:        "loop",
			Value:       false,
			Usage:       "Continue to download streams indefinitely.",
			Destination: &loop,
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

		channelID := cCtx.Args().Get(0)
		if channelID == "" {
			logger.I.Error("ChannelID is empty?! Use --help for download usage.")
			return errors.New("missing channel")
		}

		jar, err := cookiejar.New(&cookiejar.Options{})
		if err != nil {
			logger.I.Panic("failed to initialize cookie jar", zap.Error(err))
		}
		if downloadParams.CookiesFile != "" {
			if err := cookie.ParseFromFile(jar, downloadParams.CookiesFile); err != nil {
				logger.I.Error("failed to load cookies", zap.Error(err))
			}
		}

		client := &http.Client{Jar: jar, Timeout: time.Minute}

		downloader := fc2.NewDownloader(client, &downloadParams)
		logger.I.Info("running", zap.Any("params", downloadParams))

		if loop {
			for {
				err := downloader.Download(ctx, channelID)
				if errors.Is(err, context.Canceled) {
					logger.I.Info("abort watching channel", zap.String("channelID", channelID))
					break
				}
				if err != nil {
					logger.I.Error("failed to download", zap.Error(err))
				}
				time.Sleep(time.Second)
			}
			return nil
		} else {
			return try.DoExponentialBackoff(maxTries, time.Second, 2, time.Minute, func() error {
				err := downloader.Download(ctx, channelID)
				if err == io.EOF || errors.Is(err, context.Canceled) {
					return nil
				}
				return err
			})
		}
	},
}
