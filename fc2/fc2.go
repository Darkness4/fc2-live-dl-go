// Package fc2 provides a way to watch a FC2 channel.
package fc2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/hls"
	"github.com/Darkness4/fc2-live-dl-go/notify/notifier"
	"github.com/Darkness4/fc2-live-dl-go/state"
	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/Darkness4/fc2-live-dl-go/utils/try"
	"github.com/Darkness4/fc2-live-dl-go/video/concat"
	"github.com/Darkness4/fc2-live-dl-go/video/probe"
	"github.com/Darkness4/fc2-live-dl-go/video/remux"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"nhooyr.io/websocket"
)

const (
	msgBufMax     = 100
	errBufMax     = 10
	commentBufMax = 100
)

// FC2 is responsible to watch a FC2 channel.
type FC2 struct {
	*http.Client
	params    *Params
	channelID string
	log       *zerolog.Logger
}

// New creates a new FC2.
func New(client *http.Client, params *Params, channelID string) *FC2 {
	if client == nil {
		log.Panic().Msg("client is nil")
	}
	logger := log.With().Str("channelID", channelID).Logger()
	return &FC2{
		Client:    client,
		params:    params,
		channelID: channelID,
		log:       &logger,
	}
}

// Watch watches the channel for any new live stream.
func (f *FC2) Watch(ctx context.Context) (*GetMetaData, error) {
	f.log.Info().Any("params", f.params).Msg("watching channel")

	ls := NewLiveStream(f.Client, f.channelID)

	if online, err := ls.IsOnline(ctx); err != nil {
		return nil, err
	} else if !online {
		if !f.params.WaitForLive {
			return nil, ErrLiveStreamNotOnline
		}
		if err := ls.WaitForOnline(ctx, f.params.WaitPollInterval); err != nil {
			return nil, err
		}
	}

	meta, err := ls.GetMeta(ctx, WithRefetch())
	if err != nil {
		return nil, err
	}
	state.DefaultState.SetChannelState(f.channelID, state.DownloadStatePreparingFiles, nil)
	if err := notifier.NotifyPreparingFiles(ctx, f.channelID, f.params.Labels, meta); err != nil {
		log.Err(err).Msg("notify failed")
	}

	fnameInfo, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "info.json")
	if err != nil {
		return meta, err
	}
	fnameThumb, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "png")
	if err != nil {
		return meta, err
	}
	fnameStream, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "ts")
	if err != nil {
		return meta, err
	}
	fnameChat, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "fc2chat.json")
	if err != nil {
		return meta, err
	}
	fnameMuxedExt := strings.ToLower(f.params.RemuxFormat)
	fnameMuxed, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, fnameMuxedExt)
	if err != nil {
		return meta, err
	}
	fnameAudio, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "m4a")
	if err != nil {
		return meta, err
	}
	nameConcatenated, err := FormatOutput(
		f.params.OutFormat,
		meta,
		f.params.Labels,
		"combined."+fnameMuxedExt,
	)
	if err != nil {
		return meta, err
	}
	nameConcatenatedPrefix := strings.TrimSuffix(
		nameConcatenated,
		".combined."+fnameMuxedExt,
	)
	nameAudioConcatenated, err := FormatOutput(
		f.params.OutFormat,
		meta,
		f.params.Labels,
		"combined.m4a",
	)
	if err != nil {
		return meta, err
	}
	nameAudioConcatenatedPrefix := strings.TrimSuffix(
		nameAudioConcatenated,
		".combined.m4a",
	)

	if f.params.WriteInfoJSON {
		f.log.Info().Str("fnameInfo", fnameInfo).Msg("writing info json")
		func() {
			var buf bytes.Buffer
			if err := json.NewEncoder(&buf).Encode(meta); err != nil {
				f.log.Error().Err(err).Msg("failed to encode meta in info json")
				return
			}
			if err := os.WriteFile(fnameInfo, buf.Bytes(), 0o755); err != nil {
				f.log.Error().Err(err).Msg("failed to write meta in info json")
				return
			}
		}()
	}

	if f.params.WriteThumbnail {
		f.log.Info().Str("fnameThumb", fnameThumb).Msg("writing thunnail")
		func() {
			url := meta.ChannelData.Image
			resp, err := f.Get(url)
			if err != nil {
				f.log.Error().Err(err).Msg("failed to fetch thumbnail")
				return
			}
			defer resp.Body.Close()
			out, err := os.Create(fnameThumb)
			if err != nil {
				f.log.Error().Err(err).Msg("failed to open thumbnail file")
				return
			}
			defer out.Close()
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				f.log.Error().Err(err).Msg("failed to download thumbnail file")
				return
			}
		}()
	}

	state.DefaultState.SetChannelState(
		f.channelID,
		state.DownloadStateDownloading,
		map[string]interface{}{
			"metadata": meta,
		},
	)
	if err := notifier.NotifyDownloading(
		ctx,
		f.channelID,
		f.params.Labels,
		meta,
	); err != nil {
		log.Err(err).Msg("notify failed")
	}

	wsURL, err := ls.GetWebSocketURL(ctx)
	if err != nil {
		return meta, err
	}

	errWs := f.HandleWS(ctx, wsURL, fnameStream, fnameChat)
	if errWs != nil {
		f.log.Error().Err(errWs).Msg("fc2 finished with error")
	}

	state.DefaultState.SetChannelState(
		f.channelID,
		state.DownloadStatePostProcessing,
		map[string]interface{}{
			"metadata": meta,
		},
	)
	if err := notifier.NotifyPostProcessing(
		ctx,
		f.channelID,
		f.params.Labels,
		meta,
	); err != nil {
		log.Err(err).Msg("notify failed")
	}
	f.log.Info().Msg("post-processing...")

	var remuxErr error

	probeErr := probe.Do([]string{fnameStream}, probe.WithQuiet())
	if probeErr != nil {
		f.log.Error().Err(probeErr).Msg("ts is unreadable by ffmpeg")
		if f.params.DeleteCorrupted {
			if err := os.Remove(fnameStream); err != nil {
				f.log.Error().
					Str("path", fnameStream).
					Err(err).
					Msg("failed to remove corrupted file")
			}
		}
	}
	if f.params.Remux && probeErr == nil {
		f.log.Info().Str("output", fnameMuxed).Str("input", fnameStream).Msg(
			"remuxing stream...",
		)
		remuxErr = remux.Do(fnameMuxed, fnameStream)
		if remuxErr != nil {
			f.log.Error().Err(remuxErr).Msg("ffmpeg remux finished with error")
		}
	}
	var extractAudioErr error
	// Extract audio if remux on, or when concat is off.
	if f.params.ExtractAudio && (!f.params.Concat || f.params.Remux) && probeErr == nil {
		f.log.Info().Str("output", fnameAudio).Str("input", fnameStream).Msg(
			"extrating audio...",
		)
		extractAudioErr = remux.Do(fnameAudio, fnameStream, remux.WithAudioOnly())
		if extractAudioErr != nil {
			f.log.Error().Err(extractAudioErr).Msg("ffmpeg audio extract finished with error")
		}
	}

	// Concat
	if f.params.Concat {
		f.log.Info().Str("output", nameConcatenated).Str("prefix", nameConcatenatedPrefix).Msg(
			"concatenating stream...",
		)
		concatOpts := []concat.Option{
			concat.IgnoreExtension(),
		}
		if f.params.Remux {
			concatOpts = append(concatOpts, concat.IgnoreSingle())
		}
		if concatErr := concat.WithPrefix(f.params.RemuxFormat, nameConcatenatedPrefix, concatOpts...); concatErr != nil {
			f.log.Error().Err(concatErr).Msg("ffmpeg concat finished with error")
		}

		if f.params.ExtractAudio {
			f.log.Info().
				Str("output", nameAudioConcatenated).
				Str("prefix", nameAudioConcatenatedPrefix).
				Msg(
					"concatenating audio stream...",
				)
			concatOpts = append(concatOpts, concat.WithAudioOnly())
			if concatErr := concat.WithPrefix("m4a", nameAudioConcatenatedPrefix, concatOpts...); concatErr != nil {
				f.log.Error().Err(concatErr).Msg("ffmpeg concat finished with error")
			}
		}
	}

	// Delete intermediates
	if !f.params.KeepIntermediates && f.params.Remux &&
		probeErr == nil &&
		remuxErr == nil &&
		extractAudioErr == nil {
		f.log.Info().Str("file", fnameStream).Msg("delete intermediate files")
		if err := os.Remove(fnameStream); err != nil {
			f.log.Error().Err(err).Msg("couldn't delete intermediate file")
		}
	}

	f.log.Info().Msg("done")

	return meta, errWs
}

// HandleWS handles the websocket connection.
func (f *FC2) HandleWS(
	ctx context.Context,
	wsURL string,
	fnameStream string,
	fnameChat string,
) error {
	msgChan := make(chan *WSResponse, msgBufMax)
	var commentChan chan *Comment
	if f.params.WriteChat {
		commentChan = make(chan *Comment, commentBufMax)
	}
	ws := NewWebSocket(f.Client, wsURL, 30*time.Second)
	conn, err := ws.Dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "ended connection")

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		err := ws.HeartbeatLoop(ctx, conn, msgChan)
		if err == nil {
			f.log.Panic().Msg(
				"undefined behavior, heartbeat finished with nil, the download MUST finish with io.EOF or Canceled",
			)
		}
		if err == io.EOF {
			f.log.Info().Msg("healthcheck finished")
		} else if errors.Is(err, context.Canceled) {
			f.log.Info().Msg("healthcheck canceled")
		} else {
			f.log.Error().Err(err).Msg("healthcheck failed")
		}
		return err
	})

	g.Go(func() error {
		err := ws.Listen(ctx, conn, msgChan, commentChan)

		if err == nil {
			f.log.Panic().Msg(
				"undefined behavior, ws listen finished with nil, the ws listen MUST finish with io.EOF",
			)
		}
		if err == io.EOF || err == ErrWebSocketStreamEnded {
			f.log.Info().Msg("ws listen finished")
			return io.EOF
		} else if errors.Is(err, context.Canceled) {
			f.log.Info().Msg("ws listen canceled")
		} else {
			f.log.Error().Err(err).Msg("ws listen failed")
		}
		return err
	})

	g.Go(func() error {
		playlist, err := f.FetchPlaylist(ctx, ws, conn, msgChan)
		if err != nil {
			return err
		}

		f.log.Info().Any("playlist", playlist).Msg("received HLS info")

		err = f.downloadStream(ctx, playlist.URL, fnameStream)
		if err == nil {
			f.log.Panic().Msg(
				"undefined behavior, downloader finished with nil, the download MUST finish with io.EOF",
			)
		}
		if err == io.EOF {
			f.log.Info().Msg("download stream finished")
		} else if errors.Is(err, context.Canceled) {
			f.log.Info().Msg("download stream canceled")
		} else {
			f.log.Error().Err(err).Msg("download stream failed")
		}
		return err
	})

	if f.params.WriteChat {
		g.Go(func() error {
			err := f.downloadChat(ctx, commentChan, fnameChat)
			if err == nil {
				f.log.Panic().Msg(
					"undefined behavior, chat downloader finished with nil, the chat downloader MUST finish with io.EOF",
				)
			}

			if err == io.EOF {
				f.log.Info().Msg("download chat finished")
			} else if errors.Is(err, context.Canceled) {
				f.log.Info().Msg("download chat canceled")
			} else {
				f.log.Error().Err(err).Msg("download chat failed")
			}
			return err
		})
	}

	ticker := time.NewTicker(5 * time.Second) // print channel length every 5 seconds
	defer ticker.Stop()

	for {
		select {
		// Check for overflow
		case <-ticker.C:
			if len(msgChan) == msgBufMax {
				f.log.Error().Msg("msgChan overflow, flushing...")
				utils.Flush(msgChan)
			}
			if f.params.WriteChat {
				if lenCommentChan := len(commentChan); lenCommentChan == commentBufMax {
					f.log.Error().Msg("commentChan overflow, flushing...")
					utils.Flush(commentChan)
				}
			}

		// Stop at the first error
		case <-ctx.Done():
			f.log.Info().Msg("cancelling...")
			err = g.Wait()
			f.log.Info().Msg("cancelled.")
			if err == io.EOF {
				return nil
			}
			return err
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
			f.log.Panic().Msg(
				"undefined behavior, downloader finished with nil, the download MUST finish with io.EOF",
			)
		}
		if err == io.EOF {
			f.log.Info().Msg("downloader finished reading")
			return
		}
		f.log.Error().Err(err).Msg("downloader failed with error")
	}(out)

	// Write to file
	for {
		select {
		case data, ok := <-out:
			if !ok {
				f.log.Info().Msg("downloader finished writing")
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
				f.log.Error().Msg("writing chat failed, channel was closed")
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

func playlistsSummary(pp []Playlist) []struct {
	Quality Quality
	Latency Latency
} {
	summary := make([]struct {
		Quality Quality
		Latency Latency
	}, len(pp))
	for i, p := range pp {
		summary[i] = struct {
			Quality Quality
			Latency Latency
		}{
			Quality: QualityFromMode(p.Mode),
			Latency: LatencyFromMode(p.Mode),
		}
	}
	return summary
}

// FetchPlaylist fetches the playlist.
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

			playlists := SortPlaylists(ExtractAndMergePlaylists(hlsInfo))

			playlist, err := GetPlaylistOrBest(
				playlists,
				expectedMode,
			)
			if err != nil {
				return nil, err
			}
			if expectedMode != playlist.Mode {
				if try == maxTries-1 {
					f.log.Warn().
						Stringer("expected_quality", QualityFromMode(expectedMode)).
						Stringer("expected_latency", LatencyFromMode(expectedMode)).
						Stringer("got_quality", QualityFromMode(playlist.Mode)).
						Stringer("got_latency", LatencyFromMode(playlist.Mode)).
						Any("available_playlists", playlistsSummary(playlists)).
						Msg("requested quality is not available, will do...")
					return playlist, nil
				}
				return nil, errors.New("requested quality is not available")
			}

			return playlist, nil
		},
	)
}
