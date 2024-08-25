//go:build windows

package concat

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// remuxMixedTS remuxes mixed TS/AAC files into intermediate format.
func remuxMixedTS(
	ctx context.Context, // ctx is not used as operations are IO bound and has finality.
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

	var wg sync.WaitGroup

	// Check if there are mixed formats
	log.Warn().Msg("mixed formats detected, intermediate files will be created")

	// Remux all the files into intermediate format
	for _, path := range filePaths {
		wg.Add(1)
		randName := utils.GenerateRandomString(8)
		fileName := filepath.Base(path)
		dirName := filepath.Dir(path)
		intermediateName := filepath.Join(dirName, "."+fileName+"."+randName+".ts")
		intermediates = append(intermediates, intermediateName)

		// Make mpegts intermediates
		go func() {
			defer wg.Done()
			// Will IO block due to the FIFO
			if err := Do(ctx, intermediateName, []string{path}, opts...); err != nil {
				log.Err(err).
					Str("file", path).
					Msg("failed to remux to intermediate file")
			}
		}()
	}

	// Wait for all the remuxing to finish
	wg.Wait()

	return intermediates, useFIFO, nil
}
