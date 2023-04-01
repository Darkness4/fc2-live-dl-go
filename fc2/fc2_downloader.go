package fc2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"text/template"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/hls"
	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/Darkness4/fc2-live-dl-go/remux"
	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/Darkness4/fc2-live-dl-go/utils/try"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
)

type FC2 struct {
	*http.Client
	params *Params
}

func NewDownloader(client *http.Client, params *Params) *FC2 {
	if client == nil {
		logger.I.Panic("client is nil")
	}
	return &FC2{
		Client: client,
		params: params,
	}
}

func (f *FC2) Download(ctx context.Context, channelID string) error {
	logger.I.Info("downloading", zap.String("channelID", channelID))

	ls := NewLiveStream(f.Client, channelID)

	if online, err := ls.IsOnline(ctx); err != nil {
		return err
	} else if !online {
		if !f.params.WaitForLive {
			return ErrLiveStreamNotOnline
		}
		if err := ls.WaitForOnline(ctx, f.params.WaitPollInterval); err != nil {
			return err
		}
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

	errWs := f.HandleWS(ctx, wsURL, fnameStream, fnameChat)
	if errWs != nil {
		logger.I.Error("fc2 finished with error", zap.Error(errWs))
	}

	logger.I.Info("post-processing...")

	_, err = os.Stat(fnameStream)
	if f.params.Remux && !os.IsNotExist(err) {
		logger.I.Info("remuxing stream...", zap.String("output", fnameMuxed), zap.String("input", fnameStream))
		if err := remux.Do(fnameStream, fnameMuxed, false); err != nil {
			logger.I.Error("ffmpeg remux finished with error", zap.Error(err))
		}
	}
	if f.params.ExtractAudio {
		logger.I.Info("extrating audio...", zap.String("output", fnameAudio), zap.String("input", fnameStream))
		if err := remux.Do(fnameStream, fnameMuxed, true); err != nil {
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

	return errWs
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
	errChan := make(chan error, 10)
	var commentChan chan *Comment
	if f.params.WriteChat {
		commentChan = make(chan *Comment, 100)
	}
	defer func() {
		cancel()
		wg.Wait()
		close(msgChan)
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
		if err := ws.HeartbeatLoop(ctx, conn); err != nil {
			if errors.Is(err, context.Canceled) || err == io.EOF {
				logger.I.Info("healthcheck finished")
			} else {
				logger.I.Error("healthcheck failed", zap.Error(err))
				errChan <- err
			}
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := ws.Listen(ctx, conn, msgChan, commentChan)

		if err == nil {
			logger.I.Panic("undefined behavior, ws listen finished with nil, the ws listen MUST finish with io.EOF")
		}
		if err == io.EOF {
			logger.I.Info("ws listen finished")
			errChan <- err
		} else if errors.Is(err, context.Canceled) {
			logger.I.Info("ws listen canceled")
		} else {
			logger.I.Error("ws listen failed", zap.Error(err))
			errChan <- err
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		playlist, err := f.FetchPlaylist(ctx, ws, conn, msgChan)
		if err != nil {
			errChan <- err
			return
		}

		logger.I.Info("received HLS info", zap.Any("playlist", playlist))

		err = f.downloadStream(ctx, playlist.URL, fnameStream)
		if err == nil {
			logger.I.Panic("undefined behavior, downloader finished with nil, the download MUST finish with io.EOF")
		}
		if err == io.EOF {
			logger.I.Info("download stream finished")
			errChan <- err
		} else if errors.Is(err, context.Canceled) {
			logger.I.Info("download stream canceled")
		} else {
			logger.I.Error("download stream failed", zap.Error(err))
			errChan <- err
		}
	}()

	if f.params.WriteChat {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := f.downloadChat(ctx, commentChan, fnameChat)
			if err == nil {
				logger.I.Panic("undefined behavior, chat downloader finished with nil, the chat downloader MUST finish with io.EOF")
			}

			if err == io.EOF {
				logger.I.Info("download chat finished")
				errChan <- err
			} else if errors.Is(err, context.Canceled) {
				logger.I.Info("download chat canceled")
			} else {
				logger.I.Error("download chat failed", zap.Error(err))
				errChan <- err
			}
		}()
	}

	// Stop at the first error
	select {
	case err := <-errChan:
		if err == io.EOF {
			return nil
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *FC2) downloadStream(ctx context.Context, url, fName string) error {
	out := make(chan []byte)
	downloader := hls.NewDownloader(f.Client, f.params.PacketLossMax, url)

	file, err := os.Create(fName)
	if err != nil {
		return err
	}

	// Download
	go func(out chan<- []byte) {
		defer close(out)
		err := downloader.Read(ctx, out)

		if err == nil {
			logger.I.Panic("undefined behavior, downloader finished with nil, the download MUST finish with io.EOF")
		}
		if err == io.EOF {
			logger.I.Info("downloader finished reading")
			return
		}
		logger.I.Error("downloader failed with error", zap.Error(err))
	}(out)

	// Write to file
	for {
		select {
		case data, ok := <-out:
			if !ok {
				logger.I.Info("downloader finished writing")
				return io.EOF
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
		func(ctx context.Context, try int) (*Playlist, error) {
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
	if err := os.MkdirAll(filepath.Dir(fName), 0o755); err != nil {
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
		Labels      map[string]string
	}{
		Date:   timeNow.Format("2006-01-02"),
		Time:   timeNow.Format("150405"),
		Ext:    ext,
		Labels: f.params.Labels,
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
