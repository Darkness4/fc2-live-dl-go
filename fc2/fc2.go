// Package fc2 provides a way to watch a FC2 channel.
package fc2

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/Darkness4/fc2-live-dl-go/notify/notifier"
	"github.com/Darkness4/fc2-live-dl-go/state"
	"github.com/Darkness4/fc2-live-dl-go/telemetry/metrics"
	"github.com/Darkness4/fc2-live-dl-go/utils/try"
	"github.com/Darkness4/fc2-live-dl-go/video/concat"
	"github.com/Darkness4/fc2-live-dl-go/video/probe"
	"github.com/Darkness4/fc2-live-dl-go/video/remux"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName    = "fc2"
	msgBufMax     = 100
	errBufMax     = 10
	commentBufMax = 100
)

var (
	// ErrLiveStreamNotOnline is returned when the live stream is not online.
	ErrLiveStreamNotOnline = errors.New("live stream is not online")

	// ErrQualityNotExpected is returned when the quality is not expected.
	ErrQualityNotExpected = errors.New("requested quality is not expected")
)

// FC2 is responsible to watch a FC2 channel.
type FC2 struct {
	*api.Client
	Params    Params
	ChannelID string
}

// New creates a new FC2.
func New(client *api.Client, params Params, channelID string) *FC2 {
	if client == nil {
		log.Panic().Msg("client is nil")
	}
	return &FC2{
		Client:    client,
		Params:    params,
		ChannelID: channelID,
	}
}

// Watch watches the channel for any new live stream.
func (f *FC2) Watch(ctx context.Context) error {
	// NOTE: The only exit conditions are when:
	//
	// - The context is canceled.
	// - The live stream is not online and WaitForLive is false.
	//
	// Besides that, it's undefined behavior.
	// Think of the parent: it is watching multiple channels. If one dies, it is impossible to know if the others should die too.

	log := log.With().Str("channelID", f.ChannelID).Logger()
	log.Info().Any("params", f.Params).Msg("watching channel")
	ctx = log.WithContext(ctx)

	// Generate delays for exponential backoff to avoid "login required/paid program" errors.
	delays := try.GenerateDelays(5, 30*time.Second, 2, 60*time.Minute)
	delayIndex := 0

	for {
		state.DefaultState.SetChannelState(
			f.ChannelID,
			state.DownloadStateIdle,
			state.WithLabels(f.Params.Labels),
		)
		if err := notifier.NotifyIdle(ctx, f.ChannelID, f.Params.Labels); err != nil {
			log.Err(err).Msg("notify failed")
		}

		res, err := f.IsOnline(ctx)
		if err != nil {
			log.Err(err).Msg("failed to check if online")
		}

		if res.Meta.ChannelData.IsPublish == 0 {
			if !f.Params.WaitForLive {
				return ErrLiveStreamNotOnline
			}
			if res, err = f.WaitForOnline(ctx, f.Params.WaitPollInterval); err != nil {
				log.Err(err).Msg("failed to check if online")
				continue
			}
		}

		err = f.Process(ctx, res.Meta, res.WebsocketURL)

		if errors.Is(err, context.Canceled) {
			log.Info().Msg("abort watching channel")
			if state.DefaultState.GetChannelState(
				f.ChannelID,
			) != state.DownloadStateIdle {
				state.DefaultState.SetChannelState(
					f.ChannelID,
					state.DownloadStateCanceled,
					state.WithLabels(f.Params.Labels),
				)
				if err := notifier.NotifyCanceled(
					context.Background(),
					f.ChannelID,
					f.Params.Labels,
				); err != nil {
					log.Err(err).Msg("notify failed")
				}
			}
			return nil
		} else if err != nil {
			log.Err(err).Msg("failed to download")
			state.DefaultState.SetChannelError(f.ChannelID, err)
			if err := notifier.NotifyError(
				context.Background(),
				f.ChannelID,
				f.Params.Labels,
				err,
			); err != nil {
				log.Err(err).Msg("notify failed")
			}
			if errors.Is(err, api.ErrWebSocketLoginRequired) || errors.Is(err, api.ErrWebSocketPaidProgram) {
				log.Warn().Msg("backing off due to login required/paid program")
				time.Sleep(delays[delayIndex])
				if delayIndex < len(delays)-1 {
					delayIndex++
				}
			}
		} else {
			state.DefaultState.SetChannelState(
				f.ChannelID,
				state.DownloadStateFinished,
				state.WithLabels(f.Params.Labels),
			)
			if err := notifier.NotifyFinished(ctx, f.ChannelID, f.Params.Labels, res.Meta); err != nil {
				log.Err(err).Msg("notify failed")
			}
			delayIndex = 0
		}
	}
}

// WaitForOnline waits for the live stream to be online.
func (f *FC2) WaitForOnline(ctx context.Context, interval time.Duration) (IsOnlineResult, error) {
	log := log.Ctx(ctx)
	log.Info().Stringer("wait-poll-interval", interval).Msg("waiting for stream")
	for {
		res, err := f.IsOnline(ctx)
		if err != nil {
			return IsOnlineResult{}, err
		}
		if res.Meta.ChannelData.IsPublish > 0 {
			return res, nil
		}
		time.Sleep(interval)
	}
}

// IsOnlineResult is the result of IsOnline.
type IsOnlineResult struct {
	Meta         api.GetMetaData
	WebsocketURL string
}

// IsOnline checks if the live stream is online.
func (f *FC2) IsOnline(ctx context.Context) (IsOnlineResult, error) {
	log := log.Ctx(ctx)
	return try.DoExponentialBackoffWithResult(
		5,
		30*time.Second,
		2,
		60*time.Minute,
		func() (IsOnlineResult, error) {
			meta, err := f.Client.GetMeta(ctx, f.ChannelID)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return IsOnlineResult{}, err
				} else if err == api.ErrRateLimit {
					log.Error().Err(err).Msg("failed to get meta, rate limited, backoff")
					return IsOnlineResult{}, err
				}
				log.Error().Err(err).Msg("failed to get meta, considering channel as not online")
				return IsOnlineResult{}, nil
			}

			if meta.ChannelData.IsPublish == 0 {
				return IsOnlineResult{}, nil
			}

			wsURL, _, err := f.Client.GetWebSocketURL(ctx, meta)
			if err != nil {
				log.Err(err).Msg("failed to get websocket url")
				return IsOnlineResult{}, err
			}

			return IsOnlineResult{
				Meta:         meta,
				WebsocketURL: wsURL,
			}, nil
		},
	)
}

// Process processes the live stream from the metadata.
func (f *FC2) Process(
	ctx context.Context,
	meta api.GetMetaData,
	wsURL string,
) error {
	log := log.Ctx(ctx)
	ctx, span := otel.Tracer(tracerName).
		Start(ctx, "withny.Process", trace.WithAttributes(attribute.String("channelID", f.ChannelID),
			attribute.Stringer("params", f.Params),
		))
	defer span.End()

	metrics.TimeStartRecordingDeferred(f.ChannelID)

	span.AddEvent("preparing files")
	state.DefaultState.SetChannelState(
		f.ChannelID,
		state.DownloadStatePreparingFiles,
		state.WithLabels(f.Params.Labels),
	)
	if err := notifier.NotifyPreparingFiles(ctx, f.ChannelID, f.Params.Labels, meta); err != nil {
		log.Err(err).Msg("notify failed")
	}

	fnameInfo, err := PrepareFileAutoRename(f.Params.OutFormat, meta, f.Params.Labels, "info.json")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	var fnameThumb string
	if f.Params.Concat {
		fnameThumb, err = PrepareFile(f.Params.OutFormat, meta, f.Params.Labels, "png")
	} else {
		fnameThumb, err = PrepareFileAutoRename(f.Params.OutFormat, meta, f.Params.Labels, "png")
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	fnameStream, err := PrepareFileAutoRename(f.Params.OutFormat, meta, f.Params.Labels, "ts")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	fnameChat, err := PrepareFileAutoRename(
		f.Params.OutFormat,
		meta,
		f.Params.Labels,
		"fc2chat.json",
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	fnameMuxedExt := strings.ToLower(f.Params.RemuxFormat)
	fnameMuxed, err := PrepareFile(f.Params.OutFormat, meta, f.Params.Labels, fnameMuxedExt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	fnameAudio, err := PrepareFile(f.Params.OutFormat, meta, f.Params.Labels, "m4a")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	nameConcatenated, err := FormatOutput(
		f.Params.OutFormat,
		meta,
		f.Params.Labels,
		"combined."+fnameMuxedExt,
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	nameConcatenatedPrefix := strings.TrimSuffix(
		nameConcatenated,
		".combined."+fnameMuxedExt,
	)
	nameAudioConcatenated, err := FormatOutput(
		f.Params.OutFormat,
		meta,
		f.Params.Labels,
		"combined.m4a",
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	nameAudioConcatenatedPrefix := strings.TrimSuffix(
		nameAudioConcatenated,
		".combined.m4a",
	)

	if f.Params.WriteInfoJSON {
		log.Info().Str("fnameInfo", fnameInfo).Msg("writing info json")
		func() {
			f, err := os.OpenFile(fnameInfo, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
			if err != nil {
				log.Error().Err(err).Msg("failed to open info json")
				return
			}
			defer f.Close()
			enc := json.NewEncoder(f)
			enc.SetIndent("", "  ")
			if err := enc.Encode(meta); err != nil {
				log.Error().Err(err).Msg("failed to encode meta in info json")
				return
			}
		}()
	}

	if f.Params.WriteThumbnail {
		log.Info().Str("fnameThumb", fnameThumb).Msg("writing thunnail")
		func() {
			url := meta.ChannelData.Image
			resp, err := f.Get(url)
			if err != nil {
				log.Error().Err(err).Msg("failed to fetch thumbnail")
				return
			}
			defer resp.Body.Close()
			out, err := os.Create(fnameThumb)
			if err != nil {
				log.Error().Err(err).Msg("failed to open thumbnail file")
				return
			}
			defer out.Close()
			_, err = io.Copy(out, resp.Body)
			if err != nil {
				log.Error().Err(err).Msg("failed to download thumbnail file")
				return
			}
		}()
	}

	span.AddEvent("downloading")
	state.DefaultState.SetChannelState(
		f.ChannelID,
		state.DownloadStateDownloading,
		state.WithLabels(f.Params.Labels),
		state.WithExtra(map[string]interface{}{
			"metadata": meta,
		}),
	)
	if err := notifier.NotifyDownloading(
		ctx,
		f.ChannelID,
		f.Params.Labels,
		meta,
	); err != nil {
		log.Err(err).Msg("notify failed")
	}

	errWs := DownloadLiveStream(ctx, f.Client.Client, LiveStream{
		WebsocketURL:   wsURL,
		OutputFileName: fnameStream,
		ChatFileName:   fnameChat,
		Meta:           meta,
		Params:         f.Params,
	})
	if errWs != nil {
		span.RecordError(errWs)
		span.SetStatus(codes.Error, errWs.Error())
		log.Error().Err(errWs).Msg("fc2 finished with error")
	}

	span.AddEvent("post-processing")
	end := metrics.TimeStartRecording(
		ctx,
		metrics.PostProcessing.CompletionTime,
		time.Second,
		metric.WithAttributes(
			attribute.String("channel_id", f.ChannelID),
		),
	)
	defer end()
	metrics.PostProcessing.Runs.Add(ctx, 1, metric.WithAttributes(
		attribute.String("channel_id", f.ChannelID),
	))
	state.DefaultState.SetChannelState(
		f.ChannelID,
		state.DownloadStatePostProcessing,
		state.WithLabels(f.Params.Labels),
		state.WithExtra(map[string]interface{}{
			"metadata": meta,
		}),
	)
	if err := notifier.NotifyPostProcessing(
		ctx,
		f.ChannelID,
		f.Params.Labels,
		meta,
	); err != nil {
		log.Err(err).Msg("notify failed")
	}
	log.Info().Msg("post-processing...")

	var remuxErr error

	probeErr := probe.Do([]string{fnameStream}, probe.WithQuiet())
	if probeErr != nil {
		log.Error().Err(probeErr).Msg("ts is unreadable by ffmpeg")
		if f.Params.DeleteCorrupted {
			if err := os.Remove(fnameStream); err != nil {
				log.Error().
					Str("path", fnameStream).
					Err(err).
					Msg("failed to remove corrupted file")
			}
		}
	}
	if f.Params.Remux && probeErr == nil {
		log.Info().Str("output", fnameMuxed).Str("input", fnameStream).Msg(
			"remuxing stream...",
		)
		remuxErr = remux.Do(ctx, fnameMuxed, fnameStream)
		if remuxErr != nil {
			log.Error().Err(remuxErr).Msg("ffmpeg remux finished with error")
			metrics.PostProcessing.Errors.Add(ctx, 1, metric.WithAttributes(
				attribute.String("channel_id", f.ChannelID),
			))
		}
	}
	var extractAudioErr error
	// Extract audio if remux on, or when concat is off.
	if f.Params.ExtractAudio && (!f.Params.Concat || f.Params.Remux) && probeErr == nil {
		log.Info().Str("output", fnameAudio).Str("input", fnameStream).Msg(
			"extrating audio...",
		)
		extractAudioErr = remux.Do(ctx, fnameAudio, fnameStream, remux.WithAudioOnly())
		if extractAudioErr != nil {
			log.Error().Err(extractAudioErr).Msg("ffmpeg audio extract finished with error")
			metrics.PostProcessing.Errors.Add(ctx, 1, metric.WithAttributes(
				attribute.String("channel_id", f.ChannelID),
			))
		}
	}

	// Concat
	if f.Params.Concat {
		log.Info().Str("output", nameConcatenated).Str("prefix", nameConcatenatedPrefix).Msg(
			"concatenating stream...",
		)
		concatOpts := []concat.Option{
			concat.IgnoreExtension(),
		}
		if concatErr := concat.WithPrefix(ctx, f.Params.RemuxFormat, nameConcatenatedPrefix, concatOpts...); concatErr != nil {
			log.Error().Err(concatErr).Msg("ffmpeg concat finished with error")
			metrics.PostProcessing.Errors.Add(ctx, 1, metric.WithAttributes(
				attribute.String("channel_id", f.ChannelID),
			))
		}

		if f.Params.ExtractAudio {
			log.Info().
				Str("output", nameAudioConcatenated).
				Str("prefix", nameAudioConcatenatedPrefix).
				Msg(
					"concatenating audio stream...",
				)
			concatOpts = append(concatOpts, concat.WithAudioOnly())
			if concatErr := concat.WithPrefix(ctx, "m4a", nameAudioConcatenatedPrefix, concatOpts...); concatErr != nil {
				log.Error().Err(concatErr).Msg("ffmpeg concat finished with error")
				metrics.PostProcessing.Errors.Add(ctx, 1, metric.WithAttributes(
					attribute.String("channel_id", f.ChannelID),
				))
			}
		}
	}

	// Delete intermediates
	if !f.Params.KeepIntermediates && f.Params.Remux &&
		probeErr == nil &&
		remuxErr == nil &&
		extractAudioErr == nil {
		log.Info().Str("file", fnameStream).Msg("delete intermediate files")
		if err := os.Remove(fnameStream); err != nil {
			log.Error().Err(err).Msg("couldn't delete intermediate file")
			metrics.PostProcessing.Errors.Add(ctx, 1, metric.WithAttributes(
				attribute.String("channel_id", f.ChannelID),
			))
		}
	}

	span.AddEvent("done")
	log.Info().Msg("done")

	return errWs
}
