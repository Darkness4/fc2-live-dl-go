// Package clean provides a command to clean a directory.
package clean

import (
	"errors"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2/cleaner"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var (
	dryRun                 bool
	eligibleForCleaningAge time.Duration
)

// Command is the command for cleaning a directory.
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
		&cli.DurationFlag{
			Name:        "eligible-for-cleaning-age",
			Value:       48 * time.Hour,
			Usage:       "Minimum age of .combined files to be eligible for cleaning.",
			Aliases:     []string{"cleaning-age"},
			Destination: &eligibleForCleaningAge,
		},
	},
	Action: func(cCtx *cli.Context) error {
		path := cCtx.Args().First()
		if path == "" {
			log.Error().Msg("arg[0] is empty")
			return errors.New("missing file path")
		}

		opts := []cleaner.Option{
			cleaner.WithEligibleAge(eligibleForCleaningAge),
		}

		if dryRun {
			opts = append(opts, cleaner.WithDryRun())
		}

		return cleaner.Clean(path, opts...)
	},
}
