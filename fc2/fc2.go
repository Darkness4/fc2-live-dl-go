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
	"sync"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"
	"nhooyr.io/websocket"
)

const (
	msgBufMax     = 100
	errBufMax     = 10
	commentBufMax = 100
)

var (
	tracer = otel.Tracer("fc2")
	// ErrQualityNotExpected is returned when the quality is not expected.
	ErrQualityNotExpected = errors.New("requested quality is not expected")
	// ErrQualityNotAvailable is returned when the quality is not available.
	ErrQualityNotAvailable = errors.New("requested quality is not available")
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

	ctx, span := tracer.Start(ctx, "fc2.Watch")
	defer span.End()

	span.AddEvent("getting metadata")
	meta, err := ls.GetMeta(ctx, WithRefetch())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.AddEvent("preparing files")
	state.DefaultState.SetChannelState(f.channelID, state.DownloadStatePreparingFiles, nil)
	if err := notifier.NotifyPreparingFiles(ctx, f.channelID, f.params.Labels, meta); err != nil {
		log.Err(err).Msg("notify failed")
	}

	fnameInfo, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "info.json")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return meta, err
	}
	fnameThumb, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "png")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return meta, err
	}
	fnameStream, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "ts")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return meta, err
	}
	fnameChat, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "fc2chat.json")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return meta, err
	}
	fnameMuxedExt := strings.ToLower(f.params.RemuxFormat)
	fnameMuxed, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, fnameMuxedExt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return meta, err
	}
	fnameAudio, err := PrepareFile(f.params.OutFormat, meta, f.params.Labels, "m4a")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return meta, err
	}
	nameConcatenated, err := FormatOutput(
		f.params.OutFormat,
		meta,
		f.params.Labels,
		"combined."+fnameMuxedExt,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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

	span.AddEvent("downloading")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return meta, err
	}

	errWs := f.HandleWS(ctx, wsURL, fnameStream, fnameChat)
	if errWs != nil {
		span.RecordError(errWs)
		span.SetStatus(codes.Error, errWs.Error())
		f.log.Error().Err(errWs).Msg("fc2 finished with error")
	}

	span.AddEvent("post-processing")
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

	span.AddEvent("done")
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
	ctx, span := tracer.Start(ctx, "fc2.HandleWS")
	defer span.End()

	msgChan := make(chan *WSResponse, msgBufMax)
	var commentChan chan *Comment
	if f.params.WriteChat {
		commentChan = make(chan *Comment, commentBufMax)
	}
	ws := NewWebSocket(f.Client, wsURL, 30*time.Second)
	conn, err := ws.Dial(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
		ctx, span := tracer.Start(ctx, "fc2.HandleWS.download")
		defer span.End()

		playlistChan := make(chan *Playlist)
		defer close(playlistChan)
		errChan := make(chan error, errBufMax)
		defer close(errChan)

		// Quality upgrade watcher loop
		go func() {
			ticker := time.NewTicker(f.params.PollQualityUpgradeInterval)
			defer ticker.Stop()

			downloading := false

			for {
				playlist, err := f.FetchPlaylist(ctx, ws, conn, msgChan, !downloading)
				if err == nil {
					// Everything is normal
					playlistChan <- playlist
					return
				}

				if !downloading {
					if errors.Is(err, ErrQualityNotExpected) {
						f.log.Warn().
							Msg("quality is not expected, will retry during download")
						// Use the best quality available
						playlistChan <- playlist
						downloading = true
					} else {
						f.log.Error().Err(err).Msg("failed to fetch playlist")
						errChan <- err
						// error is ignored when we are downloading, we don't want to stop the download
					}
				}

				if !f.params.AllowQualityUpgrade {
					// Exit because we are not allowed to upgrade, therefore, we will not retry.
					return
				}

				select {
				case <-ticker.C:
					continue
				case <-ctx.Done():
					return
				}
			}
		}()

		err = f.downloadStream(ctx, playlistChan, fnameStream)
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
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
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
			span.RecordError(err)
			return err
		}
	}
}

func (f *FC2) downloadStream(ctx context.Context, playlists <-chan *Playlist, fName string) error {
	// TODO: This function requires serious documentation.
	ctx, span := tracer.Start(ctx, "fc2.downloadStream")
	defer span.End()

	file, err := os.Create(fName)
	if err != nil {
		return err
	}

	// Download
	out := make(chan []byte)

	go func() {
		var (
			currentCtx    context.Context
			currentCancel context.CancelFunc
			// Channel used to assure only one downloader can be launched
			doneChan chan struct{}
			// Checkpoint for the downloader when switching playlists
			checkpoint   = hls.DefaultCheckpoint()
			checkpointMu sync.Mutex
		)

	playlistLoop:
		for {
			select {
			// Received a new playlist URL
			case playlist, ok := <-playlists:
				if !ok {
					if currentCancel != nil {
						currentCancel()
					}
					return
				}

				f.log.Info().Any("playlist", playlist).Msg("received new HLS info")
				downloader := hls.NewDownloader(
					f.Client,
					f.log,
					f.params.PacketLossMax,
					playlist.URL,
				)

				if currentCancel != nil {
					// To avoid a cut off in the recording, we probe the playlist URL before downloading.
					f.log.Info().Msg("QUALITY UPGRADE! Wait for new stream to be ready...")
					span.AddEvent("quality upgrade")

					for {
						ok, err := downloader.Probe(ctx)
						if err != nil {
							f.log.Error().Err(err).Msg("failed to probe playlist, won't redownload")
							continue playlistLoop
						}
						if ok {
							break
						}
						time.Sleep(5 * time.Second)
					}

					// Cancel the old downloader
					span.AddEvent("stream alive, cancel old downloader")
					currentCancel()
					f.log.Info().Msg("switching downloader seamlessly...")
					select {
					case <-doneChan:
						log.Info().Msg("downloader switched")
					case <-time.After(30 * time.Second):
						log.Fatal().Msg("couldn't switch downloader because of a deadlock")
					}
				}

				currentCtx, currentCancel = context.WithCancel(ctx)
				doneChan = make(chan struct{}, 1)

				go func() {
					defer func() {
						close(doneChan)
					}()
					span.AddEvent("downloading")
					checkpointMu.Lock()
					checkpoint, err = downloader.Read(currentCtx, out, checkpoint)
					checkpointMu.Unlock()

					if err == nil {
						f.log.Panic().Msg(
							"undefined behavior, downloader finished with nil, the download MUST finish with io.EOF",
						)
					}
					if err == io.EOF {
						f.log.Info().Msg("downloader finished reading")
						return
					} else if errors.Is(err, context.Canceled) {
						f.log.Info().Msg("downloader canceled")
					} else {
						span.RecordError(err)
						span.SetStatus(codes.Error, err.Error())
						f.log.Error().Err(err).Msg("downloader failed with error")
						return
					}
				}()

			case <-ctx.Done():
				if currentCancel != nil {
					currentCancel()
				}
				return
			}
		}
	}()

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
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
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
	verbose bool,
) (*Playlist, error) {
	ctx, span := tracer.Start(ctx, "fc2.FetchPlaylist")
	defer span.End()

	expectedMode := int(f.params.Quality) + int(f.params.Latency) - 1
	maxTries := f.params.WaitForQualityMaxTries
	res, err := try.DoWithContextTimeoutWithResult(
		ctx,
		maxTries,
		time.Second,
		15*time.Second,
		verbose,
		func(ctx context.Context, try int) (*Playlist, error) {
			hlsInfo, err := ws.GetHLSInformation(ctx, conn, msgChan)
			if err != nil {
				span.RecordError(err)
				return nil, err
			}

			playlists := SortPlaylists(ExtractAndMergePlaylists(hlsInfo))

			playlist, err := GetPlaylistOrBest(
				playlists,
				expectedMode,
			)
			if err != nil {
				span.RecordError(err)
				return nil, err
			}
			if expectedMode != playlist.Mode {
				if try == maxTries-1 && verbose {
					if verbose {
						f.log.Warn().
							Stringer("expected_quality", QualityFromMode(expectedMode)).
							Stringer("expected_latency", LatencyFromMode(expectedMode)).
							Stringer("got_quality", QualityFromMode(playlist.Mode)).
							Stringer("got_latency", LatencyFromMode(playlist.Mode)).
							Any("available_playlists", playlistsSummary(playlists)).
							Msg("requested quality is not available, will do...")
					}
					span.RecordError(ErrQualityNotExpected)
					return playlist, ErrQualityNotExpected
				}
				span.RecordError(ErrQualityNotAvailable)
				return nil, ErrQualityNotAvailable
			}

			return playlist, nil
		},
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return res, nil
}
