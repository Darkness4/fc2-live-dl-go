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

	"github.com/Darkness4/fc2-live-dl-go/logger"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

const (
	fc2MemberAPIURL        = "https://live.fc2.com/api/memberApi.php"
	fc2ControlServerAPIURL = "https://live.fc2.com/api/getControlServer.php"
)

var ErrLiveStreamNotOnline = errors.New("live stream is not online")

type LiveStream struct {
	*http.Client
	ChannelID string
	log       *zap.Logger
	meta      *GetMetaData
}

func NewLiveStream(client *http.Client, channelID string) *LiveStream {
	if client.Jar == nil {
		logger.I.Panic("jar is nil")
	}

	return &LiveStream{
		Client:    client,
		ChannelID: channelID,
		log:       logger.I.With(zap.String("channelID", channelID)),
	}
}

func (ls *LiveStream) WaitForOnline(ctx context.Context, interval time.Duration) error {
	ls.log.Info("waiting for stream")
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

func (ls *LiveStream) IsOnline(ctx context.Context, options ...GetMetaOptions) (bool, error) {
	ls.log.Debug("checking if online")

	meta, err := ls.GetMeta(ctx, options...)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return false, err
		}
		logger.I.Error("failed to get meta, considering channel as not online", zap.Error(err))
		return false, nil
	}
	return meta.ChannelData.IsPublish > 0, nil
}

type GetMetaOptions struct {
	Refetch bool
}

func (ls *LiveStream) GetMeta(ctx context.Context, options ...GetMetaOptions) (*GetMetaData, error) {
	if len(options) > 0 {
		if !options[0].Refetch && ls.meta != nil {
			return ls.meta, nil
		}
	}

	ls.log.Debug("fetching new meta")

	v := url.Values{
		"channel":  []string{"1"},
		"profile":  []string{"1"},
		"user":     []string{"1"},
		"streamid": []string{ls.ChannelID},
	}
	req, err := http.NewRequestWithContext(ctx, "POST", fc2MemberAPIURL, strings.NewReader(v.Encode()))
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
		logger.I.Error(
			"http error",
			zap.Int("response.status", resp.StatusCode),
			zap.String("response.body", string(body)),
			zap.String("url", fc2MemberAPIURL),
			zap.String("method", "POST"),
			zap.Any("values", v),
		)

		return nil, errors.New("http error")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	metaResp := GetMetaResponse{}
	if err := json.Unmarshal(body, &metaResp); err != nil {
		logger.I.Error("failed to decode body", zap.String("body", string(body)))
		fmt.Println(string(body))
		return nil, err
	}
	metaResp.Data.ChannelData.Title = html.UnescapeString(metaResp.Data.ChannelData.Title)

	ls.meta = &metaResp.Data

	return &metaResp.Data, nil
}

func (ls *LiveStream) GetWebSocketURL(ctx context.Context) (string, error) {
	meta, err := ls.GetMeta(ctx)
	if err != nil {
		return "", err
	}
	if online, err := ls.IsOnline(ctx, GetMetaOptions{Refetch: false}); err != nil {
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
	req, err := http.NewRequestWithContext(ctx, "POST", fc2ControlServerAPIURL, strings.NewReader(v.Encode()))
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
		logger.I.Error(
			"http error",
			zap.Int("response.status", resp.StatusCode),
			zap.String("response.body", string(body)),
			zap.String("url", fc2ControlServerAPIURL),
			zap.String("method", "POST"),
			zap.Any("values", v),
		)

		return "", errors.New("http error")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	info := GetControlServerResponse{}
	if err := json.Unmarshal(body, &info); err != nil {
		logger.I.Error("failed to decode body", zap.String("body", string(body)))
		fmt.Println(string(body))
		return "", err
	}

	controlToken := &ControlToken{}
	_, _, err = jwt.NewParser().ParseUnverified(info.ControlToken, controlToken)
	if err != nil {
		return "", err
	}

	if len(controlToken.Fc2ID) > 0 {
		logger.I.Info("logged with ID", zap.Any("fc2ID", controlToken.Fc2ID))
	} else {
		logger.I.Info("Using anonymous account")
	}

	return fmt.Sprintf("%s?%s", info.URL, url.Values{"control_token": []string{info.ControlToken}}.Encode()), nil
}
