// Package hls provides functions to download HLS streams.
package hls

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const tracerName = "hls"

var (
	timeZero = time.Unix(0, 0)
	// ErrHLSForbidden is returned when the HLS download is stopped with a forbidden error.
	ErrHLSForbidden = errors.New("hls download stopped with forbidden error")
)

// Downloader is used to download HLS streams.
type Downloader struct {
	*http.Client
	packetLossMax int
	log           *zerolog.Logger
	url           string

	// ready is used to notify that the downloader is running.
	// This is to avoid stressing the users with warning logs.
	ready bool
}

// NewDownloader creates a new HLS downloader.
func NewDownloader(
	client *http.Client,
	log *zerolog.Logger,
	packetLossMax int,
	url string,
) *Downloader {

	return &Downloader{
		Client:        client,
		packetLossMax: packetLossMax,
		url:           url,
		log:           log,
	}
}

// GetFragmentURLs fetches the fragment URLs from the HLS manifest.
func (hls *Downloader) GetFragmentURLs(ctx context.Context) ([]string, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "hls.GetFragmentURLs")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", hls.url, nil)
	if err != nil {
		return []string{}, err
	}
	resp, err := hls.Client.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		url, _ := url.Parse(hls.url)

		switch resp.StatusCode {
		case 403:
			hls.log.Error().
				Str("url", url.String()).
				Int("response.status", resp.StatusCode).
				Str("response.body", string(body)).
				Str("method", "GET").
				Any("cookies", hls.Client.Jar.Cookies(url)).
				Msg("http error")
			return []string{}, ErrHLSForbidden
		case 404:
			hls.log.Warn().
				Str("url", url.String()).
				Int("response.status", resp.StatusCode).
				Str("response.body", string(body)).
				Str("method", "GET").
				Any("cookies", hls.Client.Jar.Cookies(url)).
				Msg("stream not ready")
			return []string{}, nil
		default:
			hls.log.Error().
				Str("url", url.String()).
				Int("response.status", resp.StatusCode).
				Str("response.body", string(body)).
				Str("method", "GET").
				Any("cookies", hls.Client.Jar.Cookies(url)).
				Msg("http error")
			return []string{}, errors.New("http error")
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	urls := make([]string, 0, 10)
	exists := make(map[string]bool) // Avoid duplicates

	// URLs are supposedly sorted.
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 0 && line[0] != '#' && !exists[line] {
			_, err := url.Parse(line)
			if err != nil {
				hls.log.Warn().
					Err(err).
					Msg("m3u8 returned a bad url, skipping that line")
				continue
			}
			urls = append(urls, line)
			exists[line] = true
		}
	}

	if !hls.ready {
		hls.ready = true
		hls.log.Info().Msg("downloading")
	}
	return urls, nil
}

// Checkpoint is used to resume the download from the last fragment.
type Checkpoint struct {
	LastFragmentName    string
	LastFragmentTime    time.Time
	UseTimeBasedSorting bool
}

// DefaultCheckpoint returns a default checkpoint.
func DefaultCheckpoint() Checkpoint {
	return Checkpoint{
		LastFragmentName:    "",
		LastFragmentTime:    timeZero,
		UseTimeBasedSorting: true,
	}
}

// fillQueue continuously fetches fragments url until stream end
func (hls *Downloader) fillQueue(
	ctx context.Context,
	urlChan chan<- string,
	checkpoint Checkpoint,
) (newCheckpoint Checkpoint, err error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "hls.fillQueue")
	defer span.End()

	// Used for termination
	lastFragmentReceivedTimestamp := time.Now()

	// Fields used to find the last fragment URL in the m3u8 manifest
	// TODO: warn if the checkpoint is loaded
	lastFragmentName := checkpoint.LastFragmentName
	lastFragmentTime := checkpoint.LastFragmentTime
	useTimeBasedSorting := checkpoint.UseTimeBasedSorting

	// Create a new ticker to log every 10 second
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	errorCount := 0

	for {
		select {
		case <-ticker.C:
			hls.log.Debug().Msg("still downloading")
		default:
			// Do nothing if the ticker hasn't ticked yet
		}

		urls, err := hls.GetFragmentURLs(ctx)
		if err != nil {
			span.RecordError(err)
			// Failed to fetch playlist in time
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, syscall.ECONNRESET) {
				errorCount++
				hls.log.Error().
					Int("error.count", errorCount).
					Int("error.max", hls.packetLossMax).
					Err(err).
					Msg("a playlist failed to be downloaded, retrying")

				// Ignore the error if tolerated
				if errorCount <= hls.packetLossMax {
					time.Sleep(time.Second)
					continue
				}
			}
			// fillQueue will exits here because of a stream ended with a HLSErrorForbidden
			// It can also exit here on context cancelled
			return Checkpoint{
				LastFragmentName:    lastFragmentName,
				LastFragmentTime:    lastFragmentTime,
				UseTimeBasedSorting: useTimeBasedSorting,
			}, err
		}

		newIdx := 0
		// Find the last fragment url to resume download
		if lastFragmentName != "" &&
			((useTimeBasedSorting && !lastFragmentTime.Equal(timeZero)) || !useTimeBasedSorting) {
			for i, u := range urls {
				parsed, err := url.Parse(u)
				if err != nil {
					hls.log.Err(err).
						Str("url", u).
						Msg("failed to parse fragment URL when checking for last fragment, skipping")
					continue
				}
				fragmentName := filepath.Base(parsed.Path)
				var fragmentTime time.Time
				if useTimeBasedSorting {
					tsI, err := strconv.ParseInt(parsed.Query().Get("time"), 10, 64)
					if err != nil {
						hls.log.Err(err).
							Str("url", u).
							Msg("failed to parse fragment URL, time is invalid, fragment will now be sorted by name")
						useTimeBasedSorting = false
					} else {
						fragmentTime = time.Unix(tsI, 0)
					}
				}

				if lastFragmentName >= fragmentName &&
					((useTimeBasedSorting && lastFragmentTime.Compare(fragmentTime) >= 0) || !useTimeBasedSorting) {
					newIdx = i + 1
				}
			}
		}

		nNew := len(urls) - newIdx
		if nNew > 0 {
			lastFragmentReceivedTimestamp = time.Now()
			hls.log.Debug().Strs("urls", urls[newIdx:]).Msg("found new fragments")
		}

		for _, u := range urls[newIdx:] {
			parsed, err := url.Parse(u)
			if err != nil {
				hls.log.Err(err).
					Str("url", u).
					Msg("failed to parse fragment URL, skipping")
				continue
			}
			lastFragmentName = filepath.Base(parsed.Path)
			if useTimeBasedSorting {
				tsI, err := strconv.ParseInt(parsed.Query().Get("time"), 10, 64)
				if err != nil {
					hls.log.Err(err).
						Str("url", u).
						Msg("failed to parse fragment URL, time is invalid, fragment will now be sorted by name")
					useTimeBasedSorting = false
				} else {
					lastFragmentTime = time.Unix(tsI, 0)
				}
			}
			urlChan <- u
		}

		// fillQueue will also exit here if the stream has ended (and do not send any fragment)
		if time.Since(lastFragmentReceivedTimestamp) > 30*time.Second {
			hls.log.Warn().
				Time("lastTime", lastFragmentReceivedTimestamp).
				Msg("timeout receiving new fragments, abort")
			return Checkpoint{
				LastFragmentName:    lastFragmentName,
				LastFragmentTime:    lastFragmentTime,
				UseTimeBasedSorting: useTimeBasedSorting,
			}, io.EOF
		}

		time.Sleep(time.Second)
	}
}

func (hls *Downloader) download(ctx context.Context, url string) ([]byte, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "hls.download")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return []byte{}, err
	}
	resp, err := hls.Client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return []byte{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		hls.log.Error().
			Int("response.status", resp.StatusCode).
			Str("response.body", string(body)).
			Str("url", url).
			Str("method", "GET").
			Msg("http error")

		if resp.StatusCode == 403 {
			span.RecordError(ErrHLSForbidden)
			span.SetStatus(codes.Error, ErrHLSForbidden.Error())
			return []byte{}, ErrHLSForbidden
		}

		err = errors.New("http error")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return []byte{}, err
	}

	return io.ReadAll(resp.Body)
}

// Read reads the HLS stream and sends the data to the output channel.
//
// The function will return when the context is canceled or when the stream ends.
func (hls *Downloader) Read(
	ctx context.Context,
	out chan<- []byte,
	checkpoint Checkpoint,
) (newCheckpoint Checkpoint, err error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "hls.Read")
	defer span.End()

	errChan := make(chan error, 1)
	checkpointChan := make(chan Checkpoint, 1)
	urlsChan := make(chan string, 10)

	go func() {
		defer close(errChan)
		defer close(urlsChan)
		defer close(checkpointChan)

		newCheckpoint, err := hls.fillQueue(ctx, urlsChan, checkpoint)
		checkpointChan <- newCheckpoint
		errChan <- err
	}()

	errorCount := 0

loop:
	for {
		select {
		case url, ok := <-urlsChan:
			if !ok {
				break loop
			}
			data, err := hls.download(ctx, url)
			if err != nil {
				span.RecordError(err)
				if err == ErrHLSForbidden {
					hls.log.Error().Err(err).Msg("stream was interrupted")
					return DefaultCheckpoint(), err
				}
				errorCount++
				hls.log.Error().
					Int("error.count", errorCount).
					Int("error.max", hls.packetLossMax).
					Err(err).
					Msg("a packet failed to be downloaded, skipping")
				if errorCount <= hls.packetLossMax {
					continue
				}
				return DefaultCheckpoint(), err
			} else {
				out <- data
			}
		case <-ctx.Done():
			hls.log.Info().Msg("canceled hls read")

			select {
			case cp := <-checkpointChan:
				return cp, ctx.Err()
			case <-time.After(10 * time.Second):
				hls.log.Error().Msg("timeout waiting for checkpoint")
				return DefaultCheckpoint(), ctx.Err()
			}
		case err := <-errChan:
			if err == io.EOF {
				hls.log.Info().Msg("downloaded exited with success")
			}

			select {
			case cp := <-checkpointChan:
				return cp, ctx.Err()
			case <-time.After(10 * time.Second):
				hls.log.Error().Msg("timeout waiting for checkpoint")
				return DefaultCheckpoint(), ctx.Err()
			}
		}
	}

	return DefaultCheckpoint(), io.EOF
}

// Probe checks if the stream is ready to be downloaded.
func (hls *Downloader) Probe(ctx context.Context) (bool, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "hls.Probe")
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", hls.url, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	resp, err := hls.Client.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		url, _ := url.Parse(hls.url)

		switch resp.StatusCode {
		case 404:
			hls.log.Warn().
				Str("url", hls.url).
				Int("response.status", resp.StatusCode).
				Str("response.body", string(body)).
				Str("method", "GET").
				Any("cookies", hls.Client.Jar.Cookies(url)).
				Msg("stream not ready")
			return false, nil
		default:
			hls.log.Error().
				Str("url", hls.url).
				Int("response.status", resp.StatusCode).
				Str("response.body", string(body)).
				Str("method", "GET").
				Any("cookies", hls.Client.Jar.Cookies(url)).
				Msg("http error")
			err = errors.New("http error")
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return false, err
		}
	}

	return true, nil
}
