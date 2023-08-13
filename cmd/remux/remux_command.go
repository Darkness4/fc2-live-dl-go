package remux

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Darkness4/fc2-live-dl-go/remux"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
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
			log.Error().Msg("arg[0] is empty")
			return errors.New("missing file path")
		}

		if _, err := os.Stat(file); err != nil {
			return err
		}

		fnameMuxed := prepareFile(file, "mp4")
		fnameAudio := prepareFile(file, "m4a")

		log.Info().Str("output", fnameMuxed).Str("input", file).Msg("remuxing stream...")
		if err := remux.Do(file, fnameMuxed, false); err != nil {
			log.Error().Str("output", fnameMuxed).Str("input", file).Err(err).Msg("ffmpeg remux finished with error")
		}
		if extractAudio {
			log.Error().Str("output", fnameAudio).Str("input", file).Msg("extrating audio...")
			if err := remux.Do(file, fnameAudio, true); err != nil {
				log.Error().Str("output", fnameAudio).Str("input", file).Err(err).Msg("ffmpeg audio extract finished with error")
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
		log.Panic().Err(err).Msg("couldn't create mkdir")
	}
	return fName
}
