// fc2-live-dl-go is a tool to download FC2 Live.
package main

import (
	"context"
	"errors"
	"os"

	"github.com/Darkness4/fc2-live-dl-go/cmd/clean"
	"github.com/Darkness4/fc2-live-dl-go/cmd/concat"
	"github.com/Darkness4/fc2-live-dl-go/cmd/download"
	"github.com/Darkness4/fc2-live-dl-go/cmd/remux"
	"github.com/Darkness4/fc2-live-dl-go/cmd/watch"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

var version = "dev"

func init() {
	log.Logger = log.Logger.Level(zerolog.InfoLevel).
		With().
		Caller().
		Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

var cmd = &cli.Command{
	Name:    "fc2-live-dl-go",
	Usage:   "FC2 Live download.",
	Version: version,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Sources: cli.EnvVars("DEBUG"),
			Value:   false,
			Action: func(context.Context, *cli.Command, bool) error {
				log.Logger = log.Logger.Level(zerolog.DebugLevel)
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Debug().Msg("debug logging enabled")
				return nil
			},
		},
		&cli.BoolFlag{
			Name:    "trace",
			Sources: cli.EnvVars("TRACE"),
			Value:   false,
			Action: func(context.Context, *cli.Command, bool) error {
				log.Logger = log.Logger.Level(zerolog.TraceLevel)
				zerolog.SetGlobalLevel(zerolog.TraceLevel)
				log.Trace().Msg("trace logging enabled")
				return nil
			},
		},
	},
	Commands: []*cli.Command{
		download.Command,
		watch.Command,
		remux.Command,
		concat.Command,
		clean.Command,
	},
}

func main() {
	if err := cmd.Run(context.Background(), os.Args); err != nil &&
		!errors.Is(err, context.Canceled) {
		log.Fatal().Err(err).Msg("application finished")
	}
}
