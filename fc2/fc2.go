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

const (
	msgBufMax     = 100
	errBufMax     = 10
	commentBufMax = 100
)

type FC2 struct {
	*http.Client
	params    *Params
	channelID string
	log       *zap.Logger
}

func New(client *http.Client, params *Params, channelID string) *FC2 {
	if client == nil {
		logger.I.Panic("client is nil")
	}
	return &FC2{
		Client:    client,
		params:    params,
		channelID: channelID,
		log:       logger.I.With(zap.String("channelID", channelID)),
	}
}

func (f *FC2) Watch(ctx context.Context) error {
	f.log.Info("watching channel", zap.Any("params", f.params))

	ls := NewLiveStream(f.Client, f.channelID)

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
		f.log.Info("writing info json", zap.String("fnameInfo", fnameInfo))
		func() {
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(meta); err != nil {
				f.log.Error("failed to encode meta in info json", zap.Error(err))
				return
			}
			if err := os.WriteFile(fnameInfo, buf.Bytes(), 0o755); err != nil {
				f.log.Error("failed to write meta in info json", zap.Error(err))
				return
			}
		}()
	}

	if f.params.WriteThumbnail {
		f.log.Info("writing thunnail", zap.String("fnameThumb", fnameThumb))
		func() {
			url := meta.ChannelData.Image
			resp, err := f.Get(url)
			if err != nil {
				f.log.Error("failed to fetch thumbnail", zap.Error(err))
				return
			}
			defer resp.Body.Close()
			out, err := os.Create(fnameThumb)
			if err != nil {
				f.log.Error("failed to open thumbnail file", zap.Error(err))
				return
			}
			defer out.Close()
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				f.log.Error("failed to download thumbnail file", zap.Error(err))
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
		f.log.Error("fc2 finished with error", zap.Error(errWs))
	}

	f.log.Info("post-processing...")

	var remuxErr error
	_, err = os.Stat(fnameStream)
	if f.params.Remux && !os.IsNotExist(err) {
		f.log.Info(
			"remuxing stream...",
			zap.String("output", fnameMuxed),
			zap.String("input", fnameStream),
		)
		remuxErr = remux.Do(fnameStream, fnameMuxed, false)
		if remuxErr != nil {
			f.log.Error("ffmpeg remux finished with error", zap.Error(remuxErr))
		}
	}
	var extractAudioErr error
	if f.params.ExtractAudio && !os.IsNotExist(err) {
		f.log.Info(
			"extrating audio...",
			zap.String("output", fnameAudio),
			zap.String("input", fnameStream),
		)
		extractAudioErr = remux.Do(fnameStream, fnameAudio, true)
		if extractAudioErr != nil {
			f.log.Error("ffmpeg audio extract finished with error", zap.Error(extractAudioErr))
		}
	}
	_, err = os.Stat(fnameMuxed)
	if !f.params.KeepIntermediates && !os.IsNotExist(err) && remuxErr == nil &&
		extractAudioErr == nil {
		f.log.Info("delete intermediate files", zap.String("file", fnameStream))
		if err := os.Remove(fnameStream); err != nil {
			f.log.Error("couldn't delete intermediate file", zap.Error(err))
		}
	}

	f.log.Info("done")

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
	msgChan := make(chan *WSResponse, msgBufMax)
	errChan := make(chan error, errBufMax)
	var commentChan chan *Comment
	if f.params.WriteChat {
		commentChan = make(chan *Comment, commentBufMax)
	}
	ws := NewWebSocket(f.Client, wsURL, 30*time.Second)
	conn, err := ws.Dial(ctx)
	if err != nil {
		cancel()
		return err
	}
	defer func() {
		f.log.Info("cancelling...")
		conn.Close(websocket.StatusNormalClosure, "ended connection")
		cancel()
		wg.Wait()
	}()

	wg.Add(1)
	go func() {
		if err := ws.HeartbeatLoop(ctx, conn, msgChan); err != nil {
			if err == io.EOF {
				f.log.Info("healthcheck finished")
				errChan <- err
			} else if errors.Is(err, context.Canceled) {
				f.log.Info("healthcheck canceled")
			} else {
				f.log.Error("healthcheck failed", zap.Error(err))
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
			f.log.Panic(
				"undefined behavior, ws listen finished with nil, the ws listen MUST finish with io.EOF",
			)
		}
		if err == io.EOF || err == ErrWebSocketStreamEnded {
			f.log.Info("ws listen finished")
			errChan <- io.EOF
		} else if errors.Is(err, context.Canceled) {
			f.log.Info("ws listen canceled")
		} else {
			f.log.Error("ws listen failed", zap.Error(err))
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

		f.log.Info("received HLS info", zap.Any("playlist", playlist))

		err = f.downloadStream(ctx, playlist.URL, fnameStream)
		if err == nil {
			f.log.Panic(
				"undefined behavior, downloader finished with nil, the download MUST finish with io.EOF",
			)
		}
		if err == io.EOF {
			f.log.Info("download stream finished")
			errChan <- err
		} else if errors.Is(err, context.Canceled) {
			f.log.Info("download stream canceled")
		} else {
			f.log.Error("download stream failed", zap.Error(err))
			errChan <- err
		}
	}()

	if f.params.WriteChat {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := f.downloadChat(ctx, commentChan, fnameChat)
			if err == nil {
				f.log.Panic(
					"undefined behavior, chat downloader finished with nil, the chat downloader MUST finish with io.EOF",
				)
			}

			if err == io.EOF {
				f.log.Info("download chat finished")
				errChan <- err
			} else if errors.Is(err, context.Canceled) {
				f.log.Info("download chat canceled")
			} else {
				f.log.Error("download chat failed", zap.Error(err))
				errChan <- err
			}
		}()
	}

	ticker := time.NewTicker(5 * time.Second) // print channel length every 5 seconds
	defer ticker.Stop()

	for {
		select {
		// Check for overflow
		case <-ticker.C:
			if lenMsgChan := len(msgChan); lenMsgChan == msgBufMax {
				f.log.Error("msgChan overflow, flushing...")
				utils.Flush(msgChan)
			}
			if lenErrChan := len(errChan); lenErrChan == errBufMax {
				f.log.Error("errChan overflow, flushing...")
				utils.Flush(errChan)
			}
			if f.params.WriteChat {
				if lenCommentChan := len(commentChan); lenCommentChan == commentBufMax {
					f.log.Error("commentChan overflow, flushing...")
					utils.Flush(commentChan)
				}
			}

		// Stop at the first error
		case err := <-errChan:
			if err == io.EOF {
				return nil
			}
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (f *FC2) downloadStream(ctx context.Context, url, fName string) error {
	out := make(chan []byte)
	downloader := hls.NewDownloader(f.Client, f.log, f.params.PacketLossMax, url)

	file, err := os.Create(fName)
	if err != nil {
		return err
	}

	// Download
	go func(out chan<- []byte) {
		defer close(out)
		err := downloader.Read(ctx, out)

		if err == nil {
			f.log.Panic(
				"undefined behavior, downloader finished with nil, the download MUST finish with io.EOF",
			)
		}
		if err == io.EOF {
			f.log.Info("downloader finished reading")
			return
		}
		f.log.Error("downloader failed with error", zap.Error(err))
	}(out)

	// Write to file
	for {
		select {
		case data, ok := <-out:
			if !ok {
				f.log.Info("downloader finished writing")
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
				f.log.Error("writing chat failed, channel was closed")
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

			playlist, err := GetPlaylistOrBest(
				SortPlaylists(ExtractAndMergePlaylists(hlsInfo)),
				expectedMode,
			)
			if err != nil {
				return nil, err
			}
			if expectedMode != playlist.Mode {
				if try == maxTries-1 {
					f.log.Warn(
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
		f.log.Panic("couldn't create mkdir", zap.Error(err))
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
