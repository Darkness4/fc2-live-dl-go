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
)

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
		hls.log.Error().
			Int("response.status", resp.StatusCode).
			Str("response.body", string(body)).
			Str("method", "GET").
			Any("cookies", hls.Client.Jar.Cookies(url)).
			Msg("http error")

		switch resp.StatusCode {
		case 403:
			return []string{}, ErrHLSForbidden
		case 404:
			return []string{}, nil
		default:
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
	return urls, nil
}

// fillQueue continuously fetches fragments url until stream end
func (hls *Downloader) fillQueue(ctx context.Context, urlChan chan<- string) error {
	// Used for termination
	lastFragmentReceivedTimestamp := time.Now()

	// Fields used to find the last fragment URL in the m3u8 manifest
	lastFragmentName := ""
	lastFragmentTime := timeZero
	useTimeBasedSorting := true

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
			return err
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
			if err != nil {
				hls.log.Err(err).
					Str("url", u).
					Msg("failed to parse fragment URL, skipping")
				continue
			}
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
			return io.EOF
		}

		time.Sleep(time.Second)
	}
}

func (hls *Downloader) download(ctx context.Context, url string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []byte{}, err
	}
	resp, err := hls.Client.Do(req)
	if err != nil {
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
			return []byte{}, ErrHLSForbidden
		}

		return []byte{}, errors.New("http error")
	}

	return io.ReadAll(resp.Body)
}

func (hls *Downloader) Read(ctx context.Context, out chan<- []byte) error {
	errChan := make(chan error, 1)
	urlsChan := make(chan string, 10)

	go func() {
		defer close(errChan)
		defer close(urlsChan)
		errChan <- hls.fillQueue(ctx, urlsChan)
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
				if err == ErrHLSForbidden {
					hls.log.Error().Err(err).Msg("stream was interrupted")
					return err
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
				return err
			} else {
				out <- data
			}
		case <-ctx.Done():
			hls.log.Info().Msg("canceled hls read")
			break loop
		case err := <-errChan:
			if err == io.EOF {
				hls.log.Info().Msg("downloaded exited with success")
			}

			return err
		}
	}

	return io.EOF
}
