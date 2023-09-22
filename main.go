package main

import (
	"os"

	"github.com/Darkness4/fc2-live-dl-go/cmd/download"
	"github.com/Darkness4/fc2-live-dl-go/cmd/remux"
	"github.com/Darkness4/fc2-live-dl-go/cmd/watch"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var version = "dev"

var app = &cli.App{
	Name:    "fc2-live-dl-go",
	Usage:   "FC2 Live download.",
	Version: version,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:       "debug",
			EnvVars:    []string{"DEBUG"},
			Value:      false,
			HasBeenSet: true,
			Action: func(ctx *cli.Context, s bool) error {
				if s {
					log.Logger = log.Logger.Level(zerolog.DebugLevel)
					zerolog.SetGlobalLevel(zerolog.DebugLevel)
				} else {
					log.Logger = log.Logger.Level(zerolog.InfoLevel)
					zerolog.SetGlobalLevel(zerolog.InfoLevel)
				}
				return nil
			},
		},
	},
	Commands: []*cli.Command{
		download.Command,
		watch.Command,
		remux.Command,
	},
}

func main() {
	log.Logger = log.With().Caller().Logger()
	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("application finished")
	}
}
