package fc2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/Darkness4/fc2-live-dl-lite/ffmpeg"
	"github.com/Darkness4/fc2-live-dl-lite/hls"
	"github.com/Darkness4/fc2-live-dl-lite/logger"
	"github.com/Darkness4/fc2-live-dl-lite/utils"
	"github.com/Darkness4/fc2-live-dl-lite/utils/try"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

type FC2Params struct {
	Quality                Quality           `yaml:"quality,default=3Mbps"`
	Latency                Latency           `yaml:"latency,default=mid"`
	ErrorMax               int               `yaml:"errorMax,default=200"`
	OutFormat              string            `yaml:"outFormat,default={{ .Date }} {{ .Title }} ({{ .ChannelName }}).{{ .Ext }}"`
	WriteChat              bool              `yaml:"writeChat"`
	WriteInfoJSON          bool              `yaml:"writeInfoJson"`
	WriteThumbnail         bool              `yaml:"writeThumbnail"`
	WaitForLive            bool              `yaml:"waitForLive"`
	WaitForQualityMaxTries int               `yaml:"waitForQualityMaxTries,default=15"`
	WaitPollInterval       time.Duration     `yaml:"waitPollInterval,default=5s"`
	CookiesFile            string            `yaml:"cookiesFile"`
	Remux                  bool              `yaml:"remux,default=true"`
	KeepIntermediates      bool              `yaml:"keepIntermediates"`
	ExtractAudio           bool              `yaml:"extractAudio"`
	Labels                 map[string]string `yaml:"labels"`
}

type FC2 struct {
	*http.Client
	params FC2Params
}

func New(client *http.Client, params FC2Params) *FC2 {
	return &FC2{
		Client: client,
		params: params,
	}
}

func (f *FC2) Download(ctx context.Context, channelID string) error {
	logger.I.Info("downloading", zap.String("channelID", channelID))

	ls := NewLiveStream(f.Client, channelID)

	if !ls.IsOnline(ctx) {
		if !f.params.WaitForLive {
			return ErrLiveStreamNotOnline
		}
		ls.WaitForOnline(ctx, f.params.WaitPollInterval)
	}

	meta, err := ls.GetMeta(ctx, GetMetaOptions{Refetch: false})
	if err != nil {
		return err
	}

	fnameInfo, err := f.prepareFile(meta, "info.json")
	if err != nil {
		return err
	}
	fnameThumb, err := f.prepareFile(meta, "png")
	if err != nil {
		return err
	}
	fnameStream, err := f.prepareFile(meta, "ts")
	if err != nil {
		return err
	}
	fnameChat, err := f.prepareFile(meta, "fc2chat.json")
	if err != nil {
		return err
	}
	var fnameMuxedExt string
	if f.params.Quality == QualitySound {
		fnameMuxedExt = "m4a"
	} else {
		fnameMuxedExt = "mp4"
	}
	fnameMuxed, err := f.prepareFile(meta, fnameMuxedExt)
	if err != nil {
		return err
	}
	fnameAudio, err := f.prepareFile(meta, "m4a")
	if err != nil {
		return err
	}

	if f.params.WriteInfoJSON {
		logger.I.Info("writing info json", zap.String("fnameInfo", fnameInfo))
		func() {
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(meta); err != nil {
				logger.I.Error("failed to encode meta in info json", zap.Error(err))
				return
			}
			if err := os.WriteFile(fnameInfo, buf.Bytes(), 0o755); err != nil {
				logger.I.Error("failed to write meta in info json", zap.Error(err))
				return
			}
		}()
	}

	if f.params.WriteThumbnail {
		logger.I.Info("writing thunnail", zap.String("fnameThumb", fnameThumb))
		func() {
			url := meta.ChannelData.Image
			resp, err := f.Get(url)
			if err != nil {
				logger.I.Error("failed to fetch thumbnail", zap.Error(err))
				return
			}
			defer resp.Body.Close()
			out, err := os.Create(fnameThumb)
			if err != nil {
				logger.I.Error("failed to open thumbnail file", zap.Error(err))
				return
			}
			defer out.Close()
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				logger.I.Error("failed to download thumbnail file", zap.Error(err))
				return
			}
		}()
	}

	wsURL, err := ls.GetWebSocketURL(ctx)
	if err != nil {
		return err
	}

	if err := f.HandleWS(ctx, wsURL, fnameStream, fnameChat); err != nil {
		logger.I.Error("fc2 finished with error", zap.Error(err))
	}

	logger.I.Info("post-processing...")

	_, err = os.Stat(fnameStream)
	if f.params.Remux && !os.IsNotExist(err) {
		logger.I.Info("remuxing stream...", zap.String("output", fnameMuxed), zap.String("input", fnameStream))
		if err := ffmpeg.RemuxStream(ctx, fnameStream, fnameMuxed); err != nil {
			logger.I.Error("ffmpeg remux finished with error", zap.Error(err))
		}
	}
	if f.params.ExtractAudio {
		logger.I.Info("extrating audio...", zap.String("output", fnameAudio), zap.String("input", fnameStream))
		if err := ffmpeg.RemuxStream(ctx, fnameStream, fnameAudio, "-vn"); err != nil {
			logger.I.Error("ffmpeg audio extract finished with error", zap.Error(err))
		}
	}
	_, err = os.Stat(fnameMuxed)
	if !f.params.KeepIntermediates && !os.IsNotExist(err) {
		logger.I.Info("delete intermediate files", zap.String("file", fnameStream))
		if err := os.Remove(fnameStream); err != nil {
			logger.I.Error("couldn't delete intermediate file", zap.Error(err))
		}
	}

	logger.I.Info("done")

	return nil
}

func (f *FC2) HandleWS(
	ctx context.Context,
	wsURL string,
	fnameStream string,
	fnameChat string,
) error {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	msgChan := make(chan *WSResponse, 100)
	errChan := make(chan error, 4)
	var commentChan chan *Comment
	if f.params.WriteChat {
		commentChan = make(chan *Comment, 100)
	}
	defer func() {
		cancel()
		wg.Wait()
		close(msgChan)
		close(errChan)
		if f.params.WriteChat {
			close(commentChan)
		}
	}()
	ws := NewWebSocket(f.Client, wsURL, 30*time.Second)
	conn, err := ws.Dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "ended connection")

	wg.Add(1)
	go func() {
		if err := ws.HealthCheckLoop(ctx, conn); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.I.Info("healthcheck canceled")
			} else {
				logger.I.Error("healthcheck failed", zap.Error(err))
				errChan <- err
			}
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		if err := ws.Listen(ctx, conn, msgChan, commentChan); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.I.Info("ws listen canceled")
			} else {
				logger.I.Error("ws listen failed", zap.Error(err))
				errChan <- err
			}
		}
		wg.Done()
	}()

	playlist, err := f.FetchPlaylist(ctx, ws, conn, msgChan)
	if err != nil {
		return err
	}

	logger.I.Info("received HLS info", zap.Any("playlist", playlist))

	wg.Add(1)
	go func() {
		if err := f.downloadStream(ctx, playlist.URL, fnameStream); err != nil {
			if errors.Is(err, context.Canceled) {
				logger.I.Info("download stream canceled")
			} else {
				logger.I.Error("download stream failed", zap.Error(err))
				errChan <- err
			}
		}
		wg.Done()
	}()

	if f.params.WriteChat {
		wg.Add(1)
		go func() {
			if err := f.downloadChat(ctx, commentChan, fnameChat); err != nil {
				if errors.Is(err, context.Canceled) {
					logger.I.Info("download chat canceled")
				} else {
					logger.I.Error("download chat failed", zap.Error(err))
					errChan <- err
				}
			}
			wg.Done()
		}()
	}

	// Stop at the first error
	return <-errChan
}

func (f *FC2) downloadStream(ctx context.Context, url, fName string) error {
	out := make(chan []byte)
	defer close(out)
	downloader := hls.NewDownloader(f.Client, f.params.ErrorMax, url)

	file, err := os.Create(fName)
	if err != nil {
		return err
	}

	// Download
	go func(out chan<- []byte) {
		if err := downloader.Read(ctx, out); err != nil {
			if err != io.EOF {
				logger.I.Info("downloader finished reading")
				return
			}
			logger.I.Error("downloader failed with error", zap.Error(err))
		}
	}(out)

	// Write to file
	for {
		select {
		case data, ok := <-out:
			if !ok {
				logger.I.Panic("writing stream failed, channel was closed")
			}
			_, err := file.Write(data)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func removeDuplicatesComment(input <-chan *Comment) <-chan *Comment {
	output := make(chan *Comment)
	var last *Comment

	go func() {
		defer close(output)
		for new := range input {
			if !reflect.DeepEqual(new, last) {
				output <- new
			}
			last = new
		}
	}()

	return output
}

func (f *FC2) downloadChat(ctx context.Context, commentChan <-chan *Comment, fName string) error {
	file, err := os.Create(fName)
	if err != nil {
		return err
	}

	filteredCommentChannel := removeDuplicatesComment(commentChan)

	// Write to file
	for {
		select {
		case data, ok := <-filteredCommentChannel:
			if !ok {
				logger.I.Error("writing chat failed, channel was closed")
				return io.EOF
			}

			jsonData, err := json.Marshal(data)
			if err != nil {
				return err
			}
			_, err = file.Write(jsonData)
			if err != nil {
				return err
			}
			_, err = file.Write([]byte("\n"))
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (f *FC2) FetchPlaylist(
	ctx context.Context,
	ws *WebSocket,
	conn *websocket.Conn,
	msgChan chan *WSResponse,
) (*Playlist, error) {
	expectedMode := int(f.params.Quality) + int(f.params.Latency) - 1
	maxTries := f.params.WaitForQualityMaxTries
	return try.DoWithContextTimeoutWithResult(ctx, maxTries, time.Second, 15*time.Second,
		func(try int) (*Playlist, error) {
			hlsInfo, err := ws.GetHLSInformation(ctx, conn, msgChan)
			if err != nil {
				return nil, err
			}

			playlist, err := GetPlaylistOrBest(SortPlaylists(ExtractAndMergePlaylists(hlsInfo)), expectedMode)
			if err != nil {
				return nil, err
			}
			if expectedMode != playlist.Mode {
				if try == maxTries-1 {
					logger.I.Warn(
						"requested quality is not available, will do...",
						zap.String("expected_quality", QualityFromMode(expectedMode).String()),
						zap.String("expected_latency", LatencyFromMode(expectedMode).String()),
						zap.String("got_quality", QualityFromMode(playlist.Mode).String()),
						zap.String("got_latency", LatencyFromMode(playlist.Mode).String()),
					)
					return playlist, nil
				}
				return nil, errors.New("requested quality is not available")
			}

			return playlist, nil
		},
	)
}

func (f *FC2) prepareFile(meta *GetMetaData, ext string) (fName string, err error) {
	n := 0
	// Find unique name
	for {
		var extn string
		if n == 0 {
			extn = ext
		} else {
			extn = fmt.Sprintf("%d.%s", n, ext)
		}
		fName, err = f.formatOutput(meta, extn)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(fName); errors.Is(err, os.ErrNotExist) {
			break
		}
		n++
	}

	// Mkdir parents dirs
	parent := filepath.Dir(fName)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		logger.I.Panic("couldn't create mkdir", zap.Error(err))
	}
	return fName, nil
}

func (f *FC2) formatOutput(meta *GetMetaData, ext string) (string, error) {
	timeNow := time.Now()
	formatInfo := struct {
		ChannelID   string
		ChannelName string
		Date        string
		Time        string
		Title       string
		Ext         string
	}{
		Date: timeNow.Format("2006-01-02"),
		Time: timeNow.Format("150405"),
		Ext:  ext,
	}

	tmpl, err := template.New("gotpl").Parse(f.params.OutFormat)
	if err != nil {
		return "", err
	}

	if meta != nil {
		formatInfo.ChannelID = utils.SanitizeFilename(meta.ChannelData.ChannelID)
		formatInfo.ChannelName = utils.SanitizeFilename(meta.ProfileData.Name)
		formatInfo.Title = utils.SanitizeFilename(meta.ChannelData.Title)
	}

	var formatted bytes.Buffer
	if err = tmpl.Execute(&formatted, formatInfo); err != nil {
		return "", err
	}

	return formatted.String(), nil
}
