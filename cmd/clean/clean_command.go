package clean

import (
	"errors"

	"github.com/Darkness4/fc2-live-dl-go/fc2/cleaner"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var (
	dryRun bool
)

var Command = &cli.Command{
	Name:      "clean",
	Usage:     "Clean a directory.",
	ArgsUsage: "path",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:        "dry-run",
			Value:       false,
			Usage:       "Dry run.",
			Destination: &dryRun,
		},
	},
	Action: func(cCtx *cli.Context) error {
		path := cCtx.Args().First()
		if path == "" {
			log.Error().Msg("arg[0] is empty")
			return errors.New("missing file path")
		}

		opts := []cleaner.Option{}

		if dryRun {
			opts = append(opts, cleaner.WithDryRun())
		}

		return cleaner.Clean(path, opts...)
	},
}
