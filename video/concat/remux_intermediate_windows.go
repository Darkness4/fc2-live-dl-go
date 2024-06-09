//go:build windows

package concat

import (
	"context"
	"sync"

	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/rs/zerolog/log"
)

// remuxMixedTS remuxes mixed TS/AAC files into intermediate format.
func remuxMixedTS(
	_ context.Context, // ctx is not used as operations are IO bound and has finality.
	files []string,
	opts ...Option,
) (intermediates []string, useFIFO bool, err error) {
	intermediates = make([]string, 0, len(files))

	var wg sync.WaitGroup

	// Check if there are mixed formats
	log.Warn().Msg("mixed formats detected, intermediate files will be created")

	// Remux all the files into intermediate format
	for _, file := range files {
		wg.Add(1)
		randName := utils.GenerateRandomString(8)
		intermediateName := "." + file + "." + randName + ".ts"
		intermediates = append(intermediates, intermediateName)

		// Make mpegts intermediates
		go func() {
			defer wg.Done()
			// Will IO block due to the FIFO
			if err := Do(intermediateName, []string{file}, opts...); err != nil {
				log.Error().
					Err(err).
					Str("file", file).
					Msg("failed to remux to intermediate file")
			}
		}()
	}

	// Wait for all the remuxing to finish
	wg.Wait()

	return intermediates, useFIFO, nil
}
