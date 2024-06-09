//go:build !windows

package concat

import (
	"context"
	"io"
	"os"
	"sync"
	"syscall"

	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/rs/zerolog/log"
)

// remuxMixedTS remuxes mixed TS/AAC files into intermediate format.
func remuxMixedTS(
	ctx context.Context,
	files []string,
	opts ...Option,
) (intermediates []string, useFIFO bool, err error) {
	intermediates = make([]string, 0, len(files))

	useFIFO = false

	var wg sync.WaitGroup

	// Check if we can use FIFO
	for _, file := range files {
		randName := utils.GenerateRandomString(8)
		intermediateName := "." + file + "." + randName + ".ts"
		intermediates = append(intermediates, intermediateName)

		if useFIFO {
			if err := syscall.Mkfifo(intermediateName, 0600); err != nil {
				// If fails to create the FIFO, ignore it and use an intermediate file
				log.Error().Err(err).Msg("failed to create FIFO")
				useFIFO = false
			}
		}
	}

	if !useFIFO {
		// Delete eventual FIFOs
		for _, intermediateName := range intermediates {
			_ = os.Remove(intermediateName)
		}
	}

	// Remux all the files into intermediate format
	for i, intermediateName := range intermediates {
		wg.Add(1)

		doneCh := make(chan struct{}, 1)

		// Make mpegts intermediates
		go func(intermediateName string) {
			defer func() {
				doneCh <- struct{}{}
			}()
			// Will IO block due to the FIFO
			if err := Do(intermediateName, []string{files[i]}, opts...); err != nil {
				log.Error().
					Err(err).
					Str("file", files[i]).
					Str("intermediate", intermediateName).
					Msg("failed to remux to intermediate file")
			}

			// Remove the FIFO
			if useFIFO {
				_ = os.Remove(intermediateName)
			}
		}(intermediateName)

		go func(intermediateName string) {
			defer wg.Done()
			select {
			case <-doneCh:
				return
			case <-ctx.Done():
				// Remove the FIFO
			}

			// Flush fifo
			if useFIFO {
				if err := flushFIFO(intermediateName); err != nil {
					log.Err(err).Msg("failed to flush FIFO")
				}

				_ = os.Remove(intermediateName)
			}
		}(intermediateName)
	}

	if !useFIFO {
		// Wait for all the remuxing to finish
		wg.Wait()
	}

	return intermediates, useFIFO, nil
}

func flushFIFO(file string) error {
	// Open the FIFO for reading
	fifo, err := os.OpenFile(file, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	defer fifo.Close()

	// Read all data from the FIFO
	buffer := make([]byte, 1024)
	for {
		n, err := fifo.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if n == 0 {
			break
		}
	}

	return nil
}
