package main

import (
	"os"

	"github.com/Darkness4/fc2-live-dl-lite/logger"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var app = &cli.App{
	Name:  "fc2-live-dl-lite",
	Usage: "FC2 Live download.",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "quality",
			Value:   ":3000",
			Usage:   "Address to listen on. Is used for receiving job status via the job completion plugin.",
			EnvVars: []string{"LISTEN_ADDRESS"},
		},
	},
	Action: func(cCtx *cli.Context) error {
		return nil
	},
}

func main() {
	if err := app.Run(os.Args); err != nil {
		logger.I.Fatal("app crashed", zap.Error(err))
	}
}
