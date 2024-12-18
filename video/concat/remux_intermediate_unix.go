//go:build !windows

package concat

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// remuxMixedTS remuxes mixed TS/AAC files into intermediate format.
func remuxMixedTS(
	ctx context.Context,
	filePaths []string,
	opts ...Option,
) (intermediates []string, useFIFO bool, err error) {
	attrs := make([]attribute.KeyValue, 0, len(filePaths))
	for idx, file := range filePaths {
		attrs = append(attrs, attribute.String(fmt.Sprintf("input%d", idx), file))
	}
	ctx, span := otel.Tracer(tracerName).
		Start(ctx, "concat.remuxMixedTS", trace.WithAttributes(attrs...))
	defer span.End()

	intermediates = make([]string, 0, len(filePaths))

	useFIFO = true

	var wg sync.WaitGroup

	// Check if we can use FIFO
	for _, path := range filePaths {
		randName := utils.GenerateRandomString(8)
		fileName := filepath.Base(path)
		dirName := filepath.Dir(path)
		intermediateName := filepath.Join(dirName, "."+fileName+"."+randName+".ts")
		intermediates = append(intermediates, intermediateName)

		if useFIFO {
			if err := syscall.Mkfifo(intermediateName, 0600); err != nil {
				// If fails to create the FIFO, ignore it and use an intermediate file
				log.Err(err).Msg("failed to create FIFO, FIFO will not be used")
				useFIFO = false
			}
		}
	}

	if !useFIFO {
		// Delete eventual existing FIFOs
		for _, intermediateName := range intermediates {
			_ = os.Remove(intermediateName)
		}
	}

	// Remux all the files into intermediate format
	wg.Add(len(intermediates))
	for i, intermediateName := range intermediates {
		doneCh := make(chan struct{}, 1)

		// Make mpegts intermediates
		go func(intermediateName string) {
			defer func() {
				doneCh <- struct{}{}
			}()
			// Will IO block due to the FIFO
			if err := Do(ctx, intermediateName, []string{filePaths[i]}, opts...); err != nil {
				log.Error().
					Err(err).
					Str("file", filePaths[i]).
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

			log.Warn().
				Str("intermediateName", intermediateName).
				Msg("context cancelled, flushing FIFOs")

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
		log.Err(err).Str("file", file).Msg("failed to open FIFO")
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
			log.Err(err).Str("file", file).Msg("failed to read from FIFO")
			return err
		}
		if n == 0 {
			break
		}
	}

	return nil
}
