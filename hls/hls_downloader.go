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

	"github.com/Darkness4/fc2-live-dl-go/telemetry/metrics"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
			metrics.Downloads.Errors.Add(ctx, 1)
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
			metrics.Downloads.Errors.Add(ctx, 1)
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
	hls.log.Debug().Msg("started to fill queue")
	ctx, span := otel.Tracer(tracerName).Start(ctx, "hls.fillQueue", trace.WithAttributes(
		attribute.String("last_fragment_name", checkpoint.LastFragmentName),
		attribute.String("last_fragment_time", checkpoint.LastFragmentTime.String()),
		attribute.Bool("use_time_based_sorting", checkpoint.UseTimeBasedSorting),
	))
	defer span.End()

	// Used for termination
	lastFragmentReceivedTimestamp := time.Now()

	// Fields used to find the last fragment URL in the m3u8 manifest
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
				metrics.Downloads.Errors.Add(ctx, 1)

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
			hls.log.Trace().Strs("urls", urls[newIdx:]).Msg("found new fragments")
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

func (hls *Downloader) download(
	ctx context.Context,
	w io.Writer,
	url string,
) error {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := hls.Client.Do(req)
	if err != nil {
		return err
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
			metrics.Downloads.Errors.Add(ctx, 1)
			return ErrHLSForbidden
		}

		metrics.Downloads.Errors.Add(ctx, 1)
		return errors.New("http error")
	}

	_, err = io.Copy(w, resp.Body)
	return err
}

// Read reads the HLS stream and sends the data to the writer.
//
// Read runs two threads:
//
//  1. A goroutine will continuously fetch the fragment URLs and send them to the urlsChan.
//  2. The main thread will download the fragments and write them to the writer.
//
// The function will return when the context is canceled or when the stream ends.
func (hls *Downloader) Read(
	ctx context.Context,
	writer io.Writer,
	checkpoint Checkpoint,
) (newCheckpoint Checkpoint, err error) {
	hls.log.Info().Msg("hls downloader started")
	ctx, span := otel.Tracer(tracerName).Start(ctx, "hls.Read", trace.WithAttributes(
		attribute.String("last_fragment_name", checkpoint.LastFragmentName),
		attribute.String("last_fragment_time", checkpoint.LastFragmentTime.String()),
		attribute.Bool("use_time_based_sorting", checkpoint.UseTimeBasedSorting),
	))
	defer span.End()

	ctx, cancel := context.WithCancel(ctx)

	errChan := make(chan error) // Blocking channel is used to wait for fillQueue to finish.
	defer close(errChan)

	checkpointChan := make(chan Checkpoint, 1)
	defer close(checkpointChan)

	urlsChan := make(chan string, 10)
	defer close(urlsChan)

	go func() {
		newCheckpoint, err := hls.fillQueue(ctx, urlsChan, checkpoint)
		errChan <- err
		checkpointChan <- newCheckpoint
	}()

	errorCount := 0

	for {
		select {
		case url := <-urlsChan:
			err := hls.download(ctx, writer, url)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					hls.log.Info().Msg("skip fragment download because of context canceled")
					continue // Continue to wait for fillQueue to finish
				}
				span.RecordError(err)
				if err == ErrHLSForbidden {
					hls.log.Error().Err(err).Msg("stream was interrupted")
					cancel()
					continue // Continue to wait for fillQueue to finish
				}
				errorCount++
				hls.log.Error().
					Int("error.count", errorCount).
					Int("error.max", hls.packetLossMax).
					Err(err).
					Msg("a packet failed to be downloaded, skipping")
				metrics.Downloads.Errors.Add(ctx, 1)
				if errorCount <= hls.packetLossMax {
					continue
				}
				cancel()
				continue // Continue to wait for fillQueue to finish
			}

		// fillQueue will exit here if the stream has ended or context is canceled.
		case err := <-errChan:
			defer cancel()
			if err == nil {
				hls.log.Panic().Msg("didn't expect a nil error")
			}

			if err == io.EOF {
				hls.log.Info().Msg("hls downloader exited with success")
			} else if errors.Is(err, context.Canceled) {
				hls.log.Info().Msg("hls downloader canceled")
			} else {
				hls.log.Error().Err(err).Msg("hls downloader exited with error")
			}

			select {
			case cp := <-checkpointChan:
				return cp, err
			case <-time.After(1 * time.Second):
				hls.log.Error().Msg("no checkpoint")
				return DefaultCheckpoint(), err
			}
		}
	}
}

// Probe checks if the stream is ready to be downloaded.
func (hls *Downloader) Probe(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", hls.url, nil)
	if err != nil {
		return false, err
	}
	resp, err := hls.Client.Do(req)
	if err != nil {
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
			return false, errors.New("http error")
		}
	}

	return true, nil
}
