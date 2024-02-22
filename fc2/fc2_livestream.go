package fc2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/utils/try"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	fc2MemberAPIURL        = "https://live.fc2.com/api/memberApi.php"
	fc2ControlServerAPIURL = "https://live.fc2.com/api/getControlServer.php"
)

var (
	// ErrLiveStreamNotOnline is returned when the live stream is not online.
	ErrLiveStreamNotOnline = errors.New("live stream is not online")
	// ErrRateLimit is returned when the API is rate limited.
	ErrRateLimit = errors.New("API rate limited")
)

// LiveStream encapsulates the FC2 live stream.
type LiveStream struct {
	*http.Client
	ChannelID string
	log       *zerolog.Logger
	meta      *GetMetaData
}

// NewLiveStream creates a new LiveStream.
func NewLiveStream(client *http.Client, channelID string) *LiveStream {
	if client.Jar == nil {
		log.Panic().Msg("jar is nil")
	}
	logger := log.With().Str("channelID", channelID).Logger()
	return &LiveStream{
		Client:    client,
		ChannelID: channelID,
		log:       &logger,
	}
}

// WaitForOnline waits for the live stream to be online.
func (ls *LiveStream) WaitForOnline(ctx context.Context, interval time.Duration) error {
	ls.log.Info().Msg("waiting for stream")
	for {
		online, err := ls.IsOnline(ctx)
		if err != nil {
			return err
		}
		if online {
			break
		}
		time.Sleep(interval)
	}
	return nil
}

// IsOnline checks if the live stream is online.
func (ls *LiveStream) IsOnline(ctx context.Context, options ...GetMetaOption) (bool, error) {
	return try.DoExponentialBackoffWithContextAndResult(
		ctx,
		5,
		30*time.Second,
		2,
		5*time.Minute,
		func(ctx context.Context) (bool, error) {
			meta, err := ls.GetMeta(ctx, options...)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return false, err
				} else if err == ErrRateLimit {
					ls.log.Error().Err(err).Msg("failed to get meta, rate limited, backoff")
					return false, err
				}
				ls.log.Error().Err(err).Msg("failed to get meta, considering channel as not online")
				return false, nil
			}

			return meta.ChannelData.IsPublish > 0, nil
		},
	)
}

// GetMetaOption is a function that sets options for GetMeta.
type GetMetaOption func(*GetMetaOptions)

// WithRefetch forces a refetch of the meta.
func WithRefetch() GetMetaOption {
	return func(opts *GetMetaOptions) {
		opts.refetch = true
	}
}

// GetMetaOptions contains options for GetMeta.
type GetMetaOptions struct {
	refetch bool
}

func applyGetMetaOptions(opts []GetMetaOption) *GetMetaOptions {
	o := &GetMetaOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// GetMeta gets the metadata of the live stream.
func (ls *LiveStream) GetMeta(
	ctx context.Context,
	options ...GetMetaOption,
) (*GetMetaData, error) {
	opts := applyGetMetaOptions(options)
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if len(options) > 0 {
		if !opts.refetch && ls.meta != nil {
			return ls.meta, nil
		}
	}

	v := url.Values{
		"channel":  []string{"1"},
		"profile":  []string{"1"},
		"user":     []string{"1"},
		"streamid": []string{ls.ChannelID},
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fc2MemberAPIURL,
		strings.NewReader(v.Encode()),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := ls.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		ls.log.Error().
			Int("response.status", resp.StatusCode).
			Str("response.body", string(body)).
			Str("url", fc2MemberAPIURL).
			Str("method", "POST").
			Any("values", v).
			Msg("http error")

		if resp.StatusCode == 503 {
			return nil, ErrRateLimit
		}

		return nil, errors.New("http error")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	metaResp := GetMetaResponse{}
	if err := json.Unmarshal(body, &metaResp); err != nil {
		ls.log.Error().Str("body", string(body)).Msg("failed to decode body")
		return nil, err
	}
	metaResp.Data.ChannelData.Title = html.UnescapeString(metaResp.Data.ChannelData.Title)

	ls.meta = &metaResp.Data

	return &metaResp.Data, nil
}

// GetWebSocketURL gets the WebSocket URL for the live stream.
func (ls *LiveStream) GetWebSocketURL(ctx context.Context) (string, error) {
	meta, err := ls.GetMeta(ctx)
	if err != nil {
		return "", err
	}
	if online, err := ls.IsOnline(ctx, WithRefetch()); err != nil {
		return "", err
	} else if !online {
		return "", ErrLiveStreamNotOnline
	}

	u, err := url.Parse(fc2ControlServerAPIURL)
	if err != nil {
		return "", err
	}

	orz := ""
	cookies := ls.Client.Jar.Cookies(u)
	for _, cookie := range cookies {
		if cookie.Name == "l_ortkn" {
			orz = cookie.Value
			break
		}
	}

	v := url.Values{
		"channel_id":      []string{ls.ChannelID},
		"mode":            []string{"play"},
		"orz":             []string{orz},
		"channel_version": []string{meta.ChannelData.Version},
		"client_version":  []string{"2.2.1  [1]"},
		"client_type":     []string{"pc"},
		"client_app":      []string{"browser_hls"},
		"ipv6":            []string{""},
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fc2ControlServerAPIURL,
		strings.NewReader(v.Encode()),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := ls.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		ls.log.Error().
			Int("response.status", resp.StatusCode).
			Str("response.body", string(body)).
			Str("url", fc2ControlServerAPIURL).
			Str("method", "POST").
			Any("values", v).
			Msg("http error")

		return "", errors.New("http error")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	info := GetControlServerResponse{}
	if err := json.Unmarshal(body, &info); err != nil {
		ls.log.Error().Str("body", string(body)).Msg("failed to decode body")
		return "", err
	}

	controlToken := &ControlToken{}
	_, _, err = jwt.NewParser().ParseUnverified(info.ControlToken, controlToken)
	if err != nil {
		ls.log.Error().Str("token", info.ControlToken).Msg("failed to decode jwt")
		return "", err
	}

	switch fc2ID := controlToken.Fc2ID.(type) {
	case int:
		if fc2ID > 0 {
			ls.log.Info().Int("fc2ID", fc2ID).Msg("logged with ID")
		} else {
			ls.log.Info().Msg("Using anonymous account")
		}
	case string:
		if fc2ID != "" && fc2ID != "0" {
			ls.log.Info().Str("fc2ID", fc2ID).Msg("logged with ID")
		} else {
			ls.log.Info().Msg("Using anonymous account")
		}
	}

	return fmt.Sprintf(
		"%s?%s",
		info.URL,
		url.Values{"control_token": []string{info.ControlToken}}.Encode(),
	), nil
}
