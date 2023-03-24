package cmd

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/Darkness4/fc2-live-dl-lite/fc2"
	"github.com/Darkness4/fc2-live-dl-lite/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var params = fc2.Params{}

var Download = &cli.Command{
	Name:      "download",
	Usage:     "Download a Live FC2 stream.",
	ArgsUsage: "channelID",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:       "quality",
			Value:      "3Mbps",
			HasBeenSet: true,
			Usage: `Quality of the stream to download.
Available latency options: 150Kbps, 400Kbps, 1.2Mbps, 2Mbps, 3Mbps, sound.`,
			Action: func(ctx *cli.Context, s string) error {
				params.Quality = fc2.QualityParseString(s)
				if params.Quality == fc2.QualityUnknown {
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
				params.Latency = fc2.LatencyParseString(s)
				if params.Latency == fc2.LatencyUnknown {
					logger.I.Error("Unknown input latency", zap.String("latency", s))
					return errors.New("unknown latency")
				}
				return nil
			},
		},
		&cli.StringFlag{
			Name:  "format",
			Value: "{{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}",
			Usage: `Golang templating format. Available fields: ChannelID, ChannelName, Date, Time, Title, Ext, Labels[key].
Available format options:
  ChannelID: ID of the broadcast
  ChannelName: broadcaster's profile name
  Date: local date YYYY-MM-DD
  Time: local time HHMMSS
  Ext: file extension
  Title: title of the live broadcast
  Labels[key]: custom labels
`,
			Destination: &params.OutFormat,
		},
		&cli.BoolFlag{
			Name:       "no-remux",
			Value:      false,
			HasBeenSet: true,
			Usage:      "Do not remux recordings into mp4/m4a after it is finished.",
			Action: func(ctx *cli.Context, b bool) error {
				params.Remux = !b
				return nil
			},
		},
		&cli.BoolFlag{
			Name:        "keep-intermediates",
			Value:       false,
			Usage:       "Keep the raw .ts recordings after it has been remuxed.",
			Aliases:     []string{"k"},
			Destination: &params.KeepIntermediates,
		},
		&cli.BoolFlag{
			Name:        "extract-audio",
			Value:       false,
			Usage:       "Generate an audio-only copy of the stream.",
			Aliases:     []string{"x"},
			Destination: &params.ExtractAudio,
		},
		&cli.PathFlag{
			Name:  "cookies",
			Usage: "Path to a cookies file.",
		},
		&cli.BoolFlag{
			Name:        "write-chat",
			Value:       false,
			Usage:       "Save live chat into a json file.",
			Destination: &params.WriteChat,
		},
		&cli.BoolFlag{
			Name:        "write-info-json",
			Value:       false,
			Usage:       "Dump output stream information into a json file.",
			Destination: &params.WriteInfoJSON,
		},
		&cli.BoolFlag{
			Name:        "write-thumbnail",
			Value:       false,
			Usage:       "Download thumbnail into a file.",
			Destination: &params.WriteThumbnail,
		},
		&cli.BoolFlag{
			Name:        "wait",
			Value:       false,
			Usage:       "Wait until the broadcast goes live, then start recording.",
			Destination: &params.WaitForLive,
		},
		&cli.IntFlag{
			Name:        "wait-for-quality-max-tries",
			Value:       10,
			Usage:       "If the requested quality is not available, keep retrying up to this many time before falling back to the next best quality.",
			Destination: &params.WaitForQualityMaxTries,
		},
		&cli.DurationFlag{
			Name:        "poll-interval",
			Value:       5 * time.Second,
			Usage:       "How many seconds between checks to see if broadcast is live.",
			Destination: &params.WaitPollInterval,
		},
	},
	Action: func(cCtx *cli.Context) error {
		ctx := cCtx.Context

		channelID := cCtx.Args().Get(0)
		if channelID == "" {
			logger.I.Error("ChannelID is empty?! Use --help for download usage.")
			return errors.New("missing channel")
		}

		jar, err := cookiejar.New(&cookiejar.Options{})
		if err != nil {
			logger.I.Panic("failed to initialize cookie jar", zap.Error(err))
		}
		client := &http.Client{Jar: jar, Timeout: time.Minute}

		downloader := fc2.NewDownloader(client, &params)
		logger.I.Info("running", zap.Any("params", params))
		return downloader.Download(ctx, channelID)
	},
}
