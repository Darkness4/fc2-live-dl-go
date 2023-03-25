package hls

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/logger"
	"go.uber.org/zap"
)

var (
	ErrHLSForbidden = errors.New("hls download stopped with forbidden error")
)

type Downloader struct {
	*http.Client
	packetLossMax int
	log           *zap.Logger
	url           string
}

func NewDownloader(
	client *http.Client,
	packetLossMax int,
	url string,
) *Downloader {

	return &Downloader{
		Client:        client,
		packetLossMax: packetLossMax,
		url:           url,
		log:           logger.I.With(zap.String("url", url)),
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
		hls.log.Error(
			"http error",
			zap.Int("response.status", resp.StatusCode),
			zap.String("response.body", string(body)),
			zap.String("method", "GET"),
			zap.Any("cookies", hls.Client.Jar.Cookies(url)),
		)

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

	for {
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
			hls.log.Info("found new fragments", zap.Int("n", nNew))
		}

		for _, url := range urls[newIdx:] {
			lastFragmentURL = url
			urlChan <- url
		}

		if time.Since(lastFragmentTimestamp) > 30*time.Second {
			hls.log.Warn("timeout receiving new fragments, abort")
			return nil
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
		hls.log.Error(
			"http error",
			zap.Int("response.status", resp.StatusCode),
			zap.String("response.body", string(body)),
			zap.String("url", url),
			zap.String("method", "GET"),
		)

		return []byte{}, errors.New("http error")
	}

	return io.ReadAll(resp.Body)
}

func (hls *Downloader) Read(ctx context.Context, out chan<- []byte) error {
	errChan := make(chan error)
	defer close(errChan)
	urlsChan := make(chan string, 10)
	defer close(urlsChan)

	go func() {
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
				errorCount++
				logger.I.Error("a packet failed to be downloaded, skipping", zap.Int("error.count", errorCount), zap.Int("error.max", hls.packetLossMax))
				if errorCount <= hls.packetLossMax {
					continue
				}
				return err
			}
			out <- data
		case <-ctx.Done():
			hls.log.Info("canceled hls read")
			break loop
		}
	}

	err := <-errChan
	if err == io.EOF {
		logger.I.Info("downloaded exited with success")
		return nil
	}

	return err
}
