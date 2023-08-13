package hls

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var (
	ErrHLSForbidden = errors.New("hls download stopped with forbidden error")
)

type Downloader struct {
	*http.Client
	packetLossMax int
	log           zerolog.Logger
	url           string
}

func NewDownloader(
	client *http.Client,
	log zerolog.Logger,
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

func (hls *Downloader) GetFragmentURLs(ctx context.Context) ([]string, error) {
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

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}, err
	}

	lines := strings.Split(string(data), "\n")
	urls := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) > 0 && line[0] != '#' {
			urls = append(urls, strings.TrimSpace(line))
		}
	}
	return urls, nil
}

// fillQueue continuously fetches fragments url until stream end
func (hls *Downloader) fillQueue(ctx context.Context, urlChan chan<- string) error {
	lastFragmentTimestamp := time.Now()
	lastFragmentURL := ""

	// Create a new ticker to log every 10 second
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hls.log.Debug().Msg("still downloading")
		default:
			// Do nothing if the ticker hasn't ticked yet
		}

		urls, err := hls.GetFragmentURLs(ctx)
		if err != nil {
			// fillQueue will exits here because of a stream ended with a HLSErrorForbidden
			// It can also exit here on context cancelled
			return err
		}

		newIdx := 0
		// Find the last fragment url to resume download
		if lastFragmentURL != "" {
			for i, url := range urls {
				if lastFragmentURL == url {
					newIdx = i + 1
					break
				}
			}
		}

		nNew := len(urls) - newIdx
		if nNew > 0 {
			lastFragmentTimestamp = time.Now()
			hls.log.Debug().Strs("urls", urls[newIdx:]).Msg("found new fragments")
		}

		for _, url := range urls[newIdx:] {
			lastFragmentURL = url
			urlChan <- url
		}

		// fillQueue will also exit here if the stream has ended (and do not send any fragment)
		if time.Since(lastFragmentTimestamp) > 30*time.Second {
			hls.log.Warn().Msg("timeout receiving new fragments, abort")
			return io.EOF
		}

		time.Sleep(time.Second)
	}
}

func (hls *Downloader) download(ctx context.Context, url string) ([]byte, error) {
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
			}
			out <- data
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
