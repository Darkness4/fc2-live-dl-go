package fc2

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/rs/zerolog/log"
)

// PrepareFileAutoRename prepares a file with a unique name.
func PrepareFileAutoRename(
	outFormat string,
	meta api.GetMetaData,
	labels map[string]string,
	ext string,
) (fName string, err error) {
	n := 0
	// Find unique name
	for {
		var extn string
		if n == 0 {
			extn = ext
		} else {
			extn = fmt.Sprintf("%d.%s", n, ext)
		}
		fName, err = FormatOutput(outFormat, meta, labels, extn)
		if err != nil {
			log.Error().Err(err).Msg("failed to format output")
			return "", err
		}
		if _, err := os.Stat(fName); errors.Is(err, os.ErrNotExist) {
			break
		}
		n++
	}

	// Mkdir parents dirs
	if err := os.MkdirAll(filepath.Dir(fName), 0o755); err != nil {
		panic(err)
	}
	return fName, nil
}

// PrepareFile prepares a file with a formatted name.
func PrepareFile(
	outFormat string,
	meta api.GetMetaData,
	labels map[string]string,
	ext string,
) (fName string, err error) {
	fName, err = FormatOutput(outFormat, meta, labels, ext)
	if err != nil {
		log.Error().Err(err).Msg("failed to format output")
		return "", err
	}

	// Mkdir parents dirs
	if err := os.MkdirAll(filepath.Dir(fName), 0o755); err != nil {
		panic(err)
	}
	return fName, nil
}
