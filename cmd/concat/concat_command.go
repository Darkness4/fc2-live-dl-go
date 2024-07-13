// Package concat implements the concatenation of streams.
package concat

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Darkness4/fc2-live-dl-go/video/concat"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

var (
	extractAudio bool
	outputFormat string
)

// Command is the command for concating multiple files to another container.
var Command = &cli.Command{
	Name:      "concat",
	Usage:     "Concat multiple file to another container. Order is important.",
	ArgsUsage: "...files",
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
		files := cCtx.Args().Slice()
		if len(files) == 0 {
			log.Error().Msg("arg[0] is empty")
			return errors.New("missing file path")
		}

		for _, file := range files {
			if _, err := os.Stat(file); err != nil {
				return err
			}
		}

		fnameMuxed := prepareFile(files[0], strings.ToLower(outputFormat))
		fnameAudio := prepareFile(files[0], "m4a")

		log.Info().
			Str("output", fnameMuxed).
			Strs("input", files).
			Msg("concat and remuxing streams...")
		if err := concat.Do(ctx, fnameMuxed, files); err != nil {
			log.Error().
				Str("output", fnameMuxed).
				Strs("input", files).
				Err(err).
				Msg("ffmpeg concat finished with error")
		}
		if extractAudio {
			log.Error().Str("output", fnameAudio).Strs("input", files).Msg("extrating audio...")
			if err := concat.Do(ctx, fnameAudio, files, concat.WithAudioOnly()); err != nil {
				log.Error().
					Str("output", fnameAudio).
					Strs("input", files).
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
