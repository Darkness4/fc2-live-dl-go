package remux

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/Darkness4/fc2-live-dl-go/remux"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var (
	extractAudio bool
)

var Command = &cli.Command{
	Name:      "remux",
	Usage:     "Remux a mpegts to mp4 or m4a.",
	ArgsUsage: "file",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:        "extract-audio",
			Value:       false,
			Usage:       "Generate an audio-only copy of the stream.",
			Aliases:     []string{"x"},
			Destination: &extractAudio,
		},
	},
	Action: func(cCtx *cli.Context) error {
		file := cCtx.Args().Get(0)
		if file == "" {
			logger.I.Error("arg[0] is empty?! Use --help for remux usage.")
			return errors.New("missing file path")
		}

		if _, err := os.Stat(file); err != nil {
			return err
		}

		fnameMuxed := prepareFile(file, "mp4")
		fnameAudio := prepareFile(file, "m4a")

		logger.I.Info(
			"remuxing stream...",
			zap.String("output", fnameMuxed),
			zap.String("input", file),
		)
		if err := remux.Do(file, fnameMuxed, false); err != nil {
			logger.I.Error("ffmpeg remux finished with error", zap.Error(err))
		}
		if extractAudio {
			logger.I.Info(
				"extrating audio...",
				zap.String("output", fnameAudio),
				zap.String("input", file),
			)
			if err := remux.Do(file, fnameAudio, true); err != nil {
				logger.I.Error("ffmpeg audio extract finished with error", zap.Error(err))
			}
		}
		return nil
	},
}

func removeExtension(filename string) string {
	return filename[:len(filename)-len(filepath.Ext(filename))]
}

func prepareFile(filename, newExt string) (fName string) {
	n := 0
	// Find unique name
	filename = removeExtension(filename)
	for {
		var extn string
		if n == 0 {
			extn = newExt
		} else {
			extn = fmt.Sprintf("%d.%s", n, newExt)
		}
		fName = fmt.Sprintf("%s.%s", filename, extn)
		if _, err := os.Stat(fName); errors.Is(err, os.ErrNotExist) {
			break
		}
		n++
	}

	// Mkdir parents dirs
	if err := os.MkdirAll(filepath.Dir(fName), 0o755); err != nil {
		logger.I.Panic("couldn't create mkdir", zap.Error(err))
	}
	return fName
}
