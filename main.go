package main

import (
	"os"

	"github.com/Darkness4/fc2-live-dl-go/cmd/download"
	"github.com/Darkness4/fc2-live-dl-go/cmd/watch"
	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var version = "dev"

var app = &cli.App{
	Name:    "fc2-live-dl-go",
	Usage:   "FC2 Live download.",
	Version: version,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			EnvVars: []string{"DEBUG"},
			Value:   false,
			Action: func(ctx *cli.Context, s bool) error {
				if s {
					logger.EnableDebug()
				}
				return nil
			},
		},
	},
	Commands: []*cli.Command{
		download.Command,
		watch.Command,
	},
}

func main() {
	if err := app.Run(os.Args); err != nil {
		logger.I.Fatal("application finished", zap.Error(err))
	}
}
