// Package remux provides a command for remuxing a mpegts to another container.
package remux

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Darkness4/fc2-live-dl-go/video/remux"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var (
	extractAudio bool
	outputFormat string
)

// Command is the command for remuxing a mpegts to another container.
var Command = &cli.Command{
	Name:      "remux",
	Usage:     "Remux a mpegts to another container.",
	ArgsUsage: "file",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "output-format",
			Value:       "mp4",
			Usage:       "Output format of the container.",
			Aliases:     []string{"format", "f"},
			Destination: &outputFormat,
		},
		&cli.BoolFlag{
			Name:        "extract-audio",
			Value:       false,
			Usage:       "Generate an audio-only copy of the stream.",
			Aliases:     []string{"x"},
			Destination: &extractAudio,
		},
	},
	Action: func(cCtx *cli.Context) error {
		ctx := cCtx.Context
		file := cCtx.Args().Get(0)
		if file == "" {
			log.Error().Msg("arg[0] is empty")
			return errors.New("missing file path")
		}

		if _, err := os.Stat(file); err != nil {
			return err
		}

		fnameMuxed := prepareFile(file, strings.ToLower(outputFormat))
		fnameAudio := prepareFile(file, "m4a")

		log.Info().Str("output", fnameMuxed).Str("input", file).Msg("remuxing stream...")
		if err := remux.Do(ctx, fnameMuxed, file); err != nil {
			log.Error().
				Str("output", fnameMuxed).
				Str("input", file).
				Err(err).
				Msg("ffmpeg remux finished with error")
		}
		if extractAudio {
			log.Error().Str("output", fnameAudio).Str("input", file).Msg("extrating audio...")
			if err := remux.Do(ctx, fnameAudio, file, remux.WithAudioOnly()); err != nil {
				log.Error().
					Str("output", fnameAudio).
					Str("input", file).
					Err(err).
					Msg("ffmpeg audio extract finished with error")
			}
		}
		return nil
	},
}

func prepareFile(filename, newExt string) (fName string) {
	n := 0
	// Find unique name
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
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
		log.Panic().Err(err).Msg("couldn't create mkdir")
	}
	return fName
}
