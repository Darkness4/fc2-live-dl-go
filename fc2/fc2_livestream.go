package fc2

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/Darkness4/fc2-live-dl-go/hls"
	"github.com/Darkness4/fc2-live-dl-go/telemetry/metrics"
	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/Darkness4/fc2-live-dl-go/utils/try"
	"github.com/coder/websocket"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

// LiveStream encapsulates the FC2 live stream.
type LiveStream struct {
	Meta           api.GetMetaData
	WebsocketURL   string
	OutputFileName string
	ChatFileName   string
	Params         Params
}

// DownloadLiveStream downloads the FC2 live stream.
func DownloadLiveStream(ctx context.Context, client *http.Client, ls LiveStream) error {
	log := log.Ctx(ctx)
	ctx, span := otel.Tracer(tracerName).Start(ctx, "fc2.DownloadLiveStream", trace.WithAttributes(
		attribute.String("channel_id", ls.Meta.ChannelData.ChannelID),
		attribute.String("fname_stream", ls.OutputFileName),
		attribute.String("fname_chat", ls.ChatFileName),
	))
	defer span.End()

	msgChan := make(chan *api.WSResponse, msgBufMax)
	var commentChan chan *api.Comment
	if ls.Params.WriteChat {
		commentChan = make(chan *api.Comment, commentBufMax)
	}

	ws := api.NewWebSocket(client, ls.WebsocketURL, 30*time.Second)
	conn, err := ws.Dial(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "ended connection")

	var errs []error
	var errMu sync.Mutex
	appendErr := func(err error) {
		errMu.Lock()
		errs = append(errs, err)
		errMu.Unlock()
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		err := ws.HeartbeatLoop(ctx, conn, msgChan)
		if err == nil {
			log.Panic().Msg(
				"undefined behavior, heartbeat finished with nil, the download MUST finish with io.EOF or Canceled",
			)
		}
		if err == io.EOF {
			log.Info().Msg("healthcheck finished")
		} else if errors.Is(err, context.Canceled) {
			log.Info().Msg("healthcheck canceled")
		} else {
			log.Error().Err(err).Msg("healthcheck failed")
		}
		appendErr(err)
		return err
	})

	g.Go(func() error {
		err := ws.Listen(ctx, conn, msgChan, commentChan)

		if err == nil {
			log.Panic().Msg(
				"undefined behavior, ws listen finished with nil, the ws listen MUST finish with io.EOF",
			)
		}
		// Producer is dead, close the channel to signal consumers.
		close(msgChan)
		if commentChan != nil {
			close(commentChan)
		}
		if err == io.EOF || err == api.ErrWebSocketStreamEnded {
			log.Info().Msg("ws listen finished")
			return io.EOF
		} else if errors.Is(err, context.Canceled) {
			log.Info().Msg("ws listen canceled")
		} else {
			log.Error().Err(err).Msg("ws listen failed")
		}
		appendErr(err)
		return err
	})

	g.Go(func() error {
		ctx, span := otel.Tracer(tracerName).
			Start(ctx, "fc2.DownloadLiveStream.download", trace.WithAttributes(
				attribute.String("channel_id", ls.Meta.ChannelData.ChannelID),
				attribute.String("ws_url", ls.WebsocketURL),
			))
		defer span.End()

		playlistChan := make(chan api.Playlist)
		go func() {
			<-ctx.Done()
			close(playlistChan)
			log.Info().Msg("cancelling playlist fetching")
		}()

		// Playlist fetching and quality upgrade loop
		//
		// It exits after fetching the first playlist if quality upgrade is not allowed.
		go func() {
			ticker := time.NewTicker(ls.Params.PollQualityUpgradeInterval)
			defer ticker.Stop()

			ctx, span := otel.Tracer(tracerName).
				Start(ctx, "fc2.FetchPlaylistAndQualityUpgrade", trace.WithAttributes(
					attribute.String("channel_id", ls.Meta.ChannelData.ChannelID),
				))
			defer span.End()

			downloading := false

			for {
				playlist, err := fetchPlaylist(ctx, ls, ws, conn, msgChan, !downloading)
				if err == nil {
					// Everything is normal
					playlistChan <- playlist
					return
				}

				if !downloading {
					if errors.Is(err, ErrQualityNotExpected) {
						log.Warn().
							Any("playlist", playlist).
							Msg("quality is not expected, will retry during download")
						// Use the best quality available
						playlistChan <- playlist
						downloading = true
					} else {
						log.Error().Err(err).Msg("failed to fetch playlist")
					}
				}

				if !ls.Params.AllowQualityUpgrade {
					// Exit because we are not allowed to upgrade, therefore, we will not retry.
					return
				}

				select {
				case <-ticker.C:
					continue
				case <-ctx.Done():
					log.Info().Msg("cancelling quality upgrade loop")
					return
				}
			}
		}()

		err = downloadStream(ctx, client, playlistChan, ls)
		if err == nil {
			log.Panic().Msg(
				"undefined behavior, downloader finished with nil, the download MUST finish with io.EOF",
			)
		}
		if err == io.EOF {
			log.Info().Msg("download stream finished")
		} else if errors.Is(err, context.Canceled) {
			log.Info().Msg("download stream canceled")
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.Error().Err(err).Msg("download stream failed")
		}
		appendErr(err)
		return err
	})

	if ls.Params.WriteChat {
		g.Go(func() error {
			err := DownloadChat(ctx, commentChan, ls.ChatFileName)
			if err == nil {
				log.Panic().Msg(
					"undefined behavior, chat downloader finished with nil, the chat downloader MUST finish with io.EOF",
				)
			}

			if err == io.EOF {
				log.Info().Msg("download chat finished")
			} else if errors.Is(err, context.Canceled) {
				log.Info().Msg("download chat canceled")
			} else {
				log.Error().Err(err).Msg("download chat failed")
			}
			appendErr(err)
			return err
		})
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		// Check for overflow
		case <-ticker.C:
			if len(msgChan) == msgBufMax-1 {
				log.Error().Msg("msgChan overflow, flushing...")
				utils.Flush(msgChan)
			}
			if ls.Params.WriteChat {
				if lenCommentChan := len(commentChan); lenCommentChan == commentBufMax-1 {
					log.Error().Msg("commentChan overflow, flushing...")
					utils.Flush(commentChan)
				}
			}

		// Stop at the first error
		case <-ctx.Done():
			log.Info().Msg("cancelling goroutine group...")
			_ = g.Wait()
			log.Info().Msg("cancelled goroutine group.")
			err := utils.GetFirstValuableErrorOrFirst(errs)
			if err == io.EOF {
				return nil
			}
			if !errors.Is(err, context.Canceled) {
				log.Err(err).Msg("download livestream failed")
			}
			span.RecordError(err)
			return err
		}
	}
}

func downloadStream(
	ctx context.Context,
	client *http.Client,
	playlists <-chan api.Playlist,
	ls LiveStream,
) error {
	log := log.Ctx(ctx)
	ctx, span := otel.Tracer(tracerName).Start(ctx, "fc2.downloadStream", trace.WithAttributes(
		attribute.String("channel_id", ls.Meta.ChannelData.ChannelID),
		attribute.String("fname", ls.OutputFileName),
	))
	defer span.End()

	file, err := os.Create(ls.OutputFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	errChan := make(chan error, errBufMax)

	// Variables used to save old downloader and checkpoint in case of quality upgrade.
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
				// Channel closed, we should return.
				if currentCancel != nil {
					currentCancel()
					select {
					case <-doneChan:
					case <-time.After(30 * time.Second):
						log.Fatal().Msg("couldn't cancel downloader because of a deadlock")
					}
					currentCancel = nil
				}
				break playlistLoop
			}
			if playlist.URL == "" {
				panic("empty playlist")
			}

			log.Info().Any("playlist", playlist).Msg("received new HLS info")
			span.AddEvent("playlist received", trace.WithAttributes(
				attribute.String("url", playlist.URL),
				attribute.Int("mode", playlist.Mode),
			))
			metrics.TimeEndRecording(ctx, metrics.Downloads.InitTime, ls.Meta.ChannelData.ChannelID, metric.WithAttributes(
				attribute.String("channel_id", ls.Meta.ChannelData.ChannelID),
			))
			downloader := hls.NewDownloader(
				client,
				log,
				ls.Params.PacketLossMax,
				playlist.URL,
			)

			// Is there a downloader running?
			if currentCancel != nil {
				// There is a downloader running, we need to switch to the new playlist.
				// To avoid a cut off in the recording, we probe the playlist URL before downloading.
				log.Info().Msg("QUALITY UPGRADE! Wait for new stream to be ready...")
				span.AddEvent("quality upgrade")

				for { // Healthcheck the new playlist.
					ok, err := downloader.Probe(ctx)
					if err != nil {
						log.Err(err).Msg("failed to probe playlist, won't redownload")
						continue playlistLoop
					}
					if ok {
						break
					}
					time.Sleep(5 * time.Second)

					// Check if the context for probing has been canceled.
					if ctx.Err() != nil {
						log.Err(ctx.Err()).Msg("playlist probing canceled")
						return ctx.Err()
					}
				}

				// Cancel the old downloader.
				span.AddEvent("stream alive, cancel old downloader")
				log.Info().Msg("cancelling the old downloader...")
				currentCancel()
				select {
				case <-doneChan:
				case <-time.After(30 * time.Second):
					log.Fatal().Msg("couldn't cancel downloader because of a deadlock")
				}
				log.Info().Msg("old downloader cancelled, switching downloader seamlessly...")
			}

			currentCtx, currentCancel = context.WithCancel(ctx)
			doneChan = make(chan struct{}, 1)

			// Download thread.
			go func(ctx context.Context) {
				log.Info().Msg("downloader thread started")
				defer func() {
					close(doneChan)
				}()
				span.AddEvent("downloading")
				end := metrics.TimeStartRecording(ctx, metrics.Downloads.CompletionTime, time.Second, metric.WithAttributes(
					attribute.String("channel_id", ls.Meta.ChannelData.ChannelID),
				),
				)
				defer end()
				metrics.Downloads.Runs.Add(ctx, 1, metric.WithAttributes(
					attribute.String("channel_id", ls.Meta.ChannelData.ChannelID),
				))

				// Actually download. It will block until the download is finished.
				checkpointMu.Lock()
				checkpoint, err = downloader.Read(ctx, file, checkpoint)
				checkpointMu.Unlock()

				if err != nil {
					errChan <- err
				}
				log.Info().Msg("downloader thread finished")
			}(currentCtx)

		case err := <-errChan:
			if err == nil {
				log.Panic().Msg(
					"undefined behavior, downloader finished with nil, the download MUST finish with io.EOF",
				)
			}
			if err == io.EOF {
				log.Info().Msg("downloader finished reading")
			} else if errors.Is(err, context.Canceled) {
				select {
				case <-ctx.Done():
					log.Info().Msg("downloader cancelled by parent context")
					// Parent context was cancelled, we should return.
				default:
					// Parent context was not cancelled, we should continue.
					log.Info().Msg("downloader cancelled")
					continue
				}
			} else {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				log.Err(err).Msg("downloader failed with error")
			}

			if currentCancel != nil {
				currentCancel()
			}
			return err
		}
	}

	if currentCancel != nil {
		currentCancel()
		select {
		case <-doneChan:
		case <-time.After(30 * time.Second):
			log.Fatal().Msg("couldn't cancel downloader because of a deadlock")
		}
	}
	// Context was canceled by parent, we don't need to fetch the error from the downloader.
	return io.EOF
}

func fetchPlaylist(
	ctx context.Context,
	ls LiveStream,
	ws *api.WebSocket,
	conn *websocket.Conn,
	msgChan chan *api.WSResponse,
	verbose bool,
) (api.Playlist, error) {
	log := log.Ctx(ctx)
	ctx, span := otel.Tracer(tracerName).Start(ctx, "fc2.FetchPlaylist", trace.WithAttributes(
		attribute.String("channel_id", ls.Meta.ChannelData.ChannelID),
	))
	defer span.End()

	expectedMode := int(ls.Params.Quality) + int(ls.Params.Latency) - 1
	maxTries := ls.Params.WaitForQualityMaxTries
	res, err := try.DoWithResult(
		maxTries,
		time.Second,
		func(try int) (api.Playlist, error) {
			playlist, availables, err := ws.FetchPlaylist(ctx, conn, msgChan, expectedMode)
			if err != nil {
				if errors.Is(err, api.ErrQualityNotAvailable) {
					if try == maxTries-1 {
						if verbose {
							log.Warn().
								Stringer("expected_quality", api.QualityFromMode(expectedMode)).
								Stringer("expected_latency", api.LatencyFromMode(expectedMode)).
								Stringer("got_quality", api.QualityFromMode(playlist.Mode)).
								Stringer("got_latency", api.LatencyFromMode(playlist.Mode)).
								Any("availables", playlistsSummary(availables)).
								Msg("requested quality is not available, will do...")
						}
						return playlist, ErrQualityNotExpected
					}
					return api.Playlist{}, err
				}

				span.RecordError(err)
				return api.Playlist{}, err
			}

			return playlist, nil
		},
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return res, err
	}
	return res, nil
}

func playlistsSummary(pp []api.Playlist) []struct {
	Quality api.Quality
	Latency api.Latency
} {
	summary := make([]struct {
		Quality api.Quality
		Latency api.Latency
	}, len(pp))
	for i, p := range pp {
		summary[i] = struct {
			Quality api.Quality
			Latency api.Latency
		}{
			Quality: api.QualityFromMode(p.Mode),
			Latency: api.LatencyFromMode(p.Mode),
		}
	}
	return summary
}
