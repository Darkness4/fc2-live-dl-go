package hls

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Darkness4/fc2-live-dl-lite/logger"
	"github.com/Darkness4/fc2-live-dl-lite/utils/blockingheap"
	"github.com/Darkness4/fc2-live-dl-lite/utils/queue"
	"github.com/Darkness4/fc2-live-dl-lite/utils/try"
	"go.uber.org/zap"
)

var (
	ErrHLSForbidden = errors.New("hls download stopped with forbidden error")
)

type Downloader struct {
	*http.Client
	threads  int
	fragURLs *blockingheap.BlockingHeap[*queue.Item[string]]
	fragData *blockingheap.BlockingHeap[*queue.Item[[]byte]]
	log      *zap.Logger
	url      string
}

func NewDownloader(
	client *http.Client,
	threads int,
	url string,
) *Downloader {
	fragURLs := queue.NewPriorityQueue[string](100)
	fragData := queue.NewPriorityQueue[[]byte](100)

	return &Downloader{
		Client:   client,
		threads:  threads,
		url:      url,
		fragURLs: blockingheap.New[*queue.Item[string]](fragURLs),
		fragData: blockingheap.New[*queue.Item[[]byte]](fragData),
		log:      logger.I.With(zap.String("url", url)).With(zap.Int("threads", threads)),
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
func (hls *Downloader) fillQueue(ctx context.Context) error {
	lastFragmentTimestamp := time.Now()
	lastFragmentURL := ""
	fragIdx := 0

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
			hls.log.Info("found new fragments", zap.Int("n", nNew))
		}

		for _, url := range urls[newIdx:] {
			lastFragmentURL = url
			if err := hls.fragURLs.Push(&queue.Item[string]{
				Value:    url,
				Priority: 0,
			}); err != nil {
				if err == io.EOF {
					hls.log.Warn("fillQueue received EOF when pushing new URLs")
				}
				return err
			}
			fragIdx++
		}

		if time.Since(lastFragmentTimestamp) > 30*time.Second {
			hls.log.Warn("timeout receiving new fragments, abort")
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

// downloadWorker continuously consume a queue of urls and push the data in an another queue
func (hls *Downloader) downloadWorker(ctx context.Context, workerID int) {
	parentLog := hls.log.With(zap.Int("workerID", workerID))

	for {
		item, err := hls.fragURLs.Pop()
		if err != nil {
			// Worker will exits here because the main thread will close the queue
			if err == io.EOF {
				parentLog.Info("worker received EOF, exiting...", zap.Error(err))
				return
			}
			parentLog.Error("worker received error, exiting...", zap.Error(err))
			return
		}

		log := parentLog.With(zap.Any("fragment", item))
		log.Debug("downloading fragment")
		if err := try.Do(5, time.Second, func() error {
			req, err := http.NewRequestWithContext(ctx, "GET", item.Value, nil)
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
				hls.log.Error(
					"http error",
					zap.Int("response.status", resp.StatusCode),
					zap.String("response.body", string(body)),
					zap.String("url", item.Value),
					zap.String("method", "GET"),
				)

				return errors.New("http error")
			}

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if err := hls.fragData.Push(&queue.Item[[]byte]{
				Value:    data,
				Priority: item.Priority,
			}); err != nil {
				if err == io.EOF {
					log.Warn("worker received EOF, abort download...", zap.Error(err))
					return nil
				}
				return err
			}
			return nil
		}); err != nil {
			log.Error("failed to download fragment", zap.Error(err))
			if err := hls.fragData.Push(&queue.Item[[]byte]{
				Value:    []byte{},
				Priority: item.Priority,
			}); err != nil {
				if err == io.EOF {
					log.Warn("worker received EOF, abort download...", zap.Error(err))
				}
			}
		}
	}
}

func (hls *Downloader) download(ctx context.Context) error {
	hls.log.Info("downloading")
	defer func() {
		hls.fragURLs.Close()
		hls.fragData.Close()
	}()

	// Use a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	wg.Add(hls.threads)

	for i := 0; i < hls.threads; i++ {
		go hls.downloadWorker(ctx, i)
	}

	// Fill queue is blocking until the stream end
	return hls.fillQueue(ctx)
}

func (hls *Downloader) Read(ctx context.Context, out chan<- []byte) error {
	errChan := make(chan error)

	go func(out chan<- []byte, errChan chan error) {
		index := 0
		for {
			item, err := hls.fragData.Pop()
			if err != nil {
				if err == io.EOF {
					hls.log.Info("received EOF, reader exiting safely...")
					errChan <- err
					return
				}
				errChan <- err
				return
			}
			if item.Priority == index {
				out <- item.Value
				index++
			}
			if err := hls.fragData.Push(item); err != nil {
				if err == io.EOF {
					hls.log.Info("received EOF, reader exiting safely...")
					errChan <- err
					return
				}
				errChan <- err
				return
			}
		}
	}(out, errChan)

	if err := hls.download(ctx); err != nil {
		if err == ErrHLSForbidden {
			hls.log.Info("download workers stopped, stream ended")
			return nil
		}
		return err
	}
	// Get reader error
	err := <-errChan
	if err == io.EOF {
		hls.log.Info("download workers stopped, stream ended")
	} else {
		hls.log.Error("download workers failed", zap.Error(err))
		return err
	}
	return nil
}
