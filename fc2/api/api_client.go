// Package api provides the FC2 API client.
package api

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

const (
	fc2MemberAPIURL        = "https://live.fc2.com/api/memberApi.php"
	fc2ControlServerAPIURL = "https://live.fc2.com/api/getControlServer.php"
	fc2LoginURL            = "https://live.fc2.com/login/"
	fc2ChannelListURL      = "https://live.fc2.com/contents/allchannellist.php"
)

var (
	// ErrRateLimit is returned when the API is rate limited.
	ErrRateLimit = errors.New("API rate limited")
)

// Client is the FC2 API client.
type Client struct {
	*http.Client
}

// NewClient creates a new FC2 API client.
func NewClient(client *http.Client) *Client {
	return &Client{client}
}

// GetMeta gets the metadata of the live stream.
func (c *Client) GetMeta(ctx context.Context, channelID string) (GetMetaData, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	v := url.Values{
		"channel":  []string{"1"},
		"profile":  []string{"1"},
		"user":     []string{"1"},
		"streamid": []string{channelID},
	}
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fc2MemberAPIURL,
		strings.NewReader(v.Encode()),
	)
	if err != nil {
		return GetMetaData{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	log := log.With().
		Str("method", "POST").
		Str("url", fc2MemberAPIURL+"?"+v.Encode()).
		Str("channelID", channelID).
		Logger()

	resp, err := c.Do(req)
	if err != nil {
		return GetMetaData{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Error().
			Int("response.status", resp.StatusCode).
			Str("response.body", string(body)).
			Msg("http error")

		if resp.StatusCode == 503 {
			return GetMetaData{}, ErrRateLimit
		}

		err := errors.New("http error")
		return GetMetaData{}, err
	}

	metaResp := GetMetaResponse{}
	if err := utils.JSONDecodeAndPrintOnError(resp.Body, &metaResp); err != nil {
		return GetMetaData{}, err
	}
	metaResp.Data.ChannelData.Title = html.UnescapeString(metaResp.Data.ChannelData.Title)

	return metaResp.Data, nil
}

// GetWebSocketURL gets the WebSocket URL for the live stream.
func (c *Client) GetWebSocketURL(
	ctx context.Context,
	meta GetMetaData,
) (wsURL string, controlToken ControlToken, err error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "fc2.GetWebSocketURL")
	defer span.End()

	u, err := url.Parse(fc2ControlServerAPIURL)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", ControlToken{}, err
	}

	orz := ""
	cookies := c.Client.Jar.Cookies(u)
	for _, cookie := range cookies {
		if cookie.Name == "l_ortkn" {
			orz = cookie.Value
			break
		}
	}

	v := url.Values{
		"channel_id":      []string{meta.ChannelData.ChannelID},
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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", ControlToken{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	log := log.With().
		Str("method", "POST").
		Str("url", fc2ControlServerAPIURL+"?"+v.Encode()).
		Str("channelID", meta.ChannelData.ChannelID).
		Logger()

	resp, err := c.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", ControlToken{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Error().
			Int("response.status", resp.StatusCode).
			Str("response.body", string(body)).
			Msg("http error")

		err := errors.New("http error")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", ControlToken{}, err
	}

	info := GetControlServerResponse{}
	if err := utils.JSONDecodeAndPrintOnError(resp.Body, &info); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", ControlToken{}, err
	}

	controlToken = ControlToken{}
	_, _, err = jwt.NewParser().ParseUnverified(info.ControlToken, &controlToken)
	if err != nil {
		log.Error().Str("token", info.ControlToken).Msg("failed to decode jwt")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", controlToken, err
	}

	if controlToken.UserName == "" {
		log.Warn().Any("token", controlToken).Msg("downloading with anonymous user")
	} else {
		log.Info().Any("username", controlToken.UserName).Msg("downloading with user")
	}

	return fmt.Sprintf(
		"%s?%s",
		info.URL,
		url.Values{"control_token": []string{info.ControlToken}}.Encode(),
	), controlToken, nil
}

// memberLoginRegex is used to extract the next URL. No need for parsing HTML, just fetch the URL.
var memberLoginRegex = regexp.MustCompile(
	`href="(http://live\.fc2\.com/member_login/\?uid=[^&]+&cc=[^"]+)"`,
)

var usernameRegex = regexp.MustCompile(`<span class="m-hder01_uName">(.*?)</span>`)

// Login to FC2 and fill the CookieJar.
//
// You need to probably need to
func (c *Client) Login(ctx context.Context) error {
	// Phase 1: Redirect to https://id.fc2.com
	var memberLoginURL string
	if err := func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", fc2LoginURL, nil)
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non-ok http code returned: %d", resp.StatusCode)
		}
		if resp.Request.URL.Host != "id.fc2.com" {
			return fmt.Errorf("reached unknown location: %s", resp.Request.Host)
		}
		// Look for next URL
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		matches := memberLoginRegex.FindStringSubmatch(string(body))
		if len(matches) == 0 {
			return errors.New("failed to find next url, cookies are invalid")
		}

		memberLoginURL = matches[1]
		log.Info().Str("memberLoginURL", memberLoginURL).Msg("login phase 1 success")

		return nil
	}(); err != nil {
		return fmt.Errorf("login phase 1 failed: %w", err)
	}

	// Phase 2: Login to https://live.fc2.com/member_login/?uid=<uid>&cc=<cc>
	if err := func() error {
		u, err := url.Parse(memberLoginURL)
		if err != nil {
			return err
		}
		u.Scheme = "https"
		req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("non-ok http code returned: %d", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if !strings.Contains(string(body), "logout") {
			return errors.New("failed to find 'logout', which means the login failed")
		}
		matches := usernameRegex.FindStringSubmatch(string(body))
		if len(matches) == 0 {
			log.Info().Msg("login phase 2 success (but we didn't find your username)")
		} else {
			log.Info().Str("username", matches[1]).Msg("login phase 2 success")
		}

		return nil
	}(); err != nil {
		return fmt.Errorf("login phase 2 failed: %w", err)
	}

	return nil
}

// FindUnrestrictedStream finds the first unrestricted stream.
func (c *Client) FindUnrestrictedStream(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fc2ChannelListURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("non-ok http code returned: %d", resp.StatusCode)
	}

	var channelList GetChannelListResponse
	if err := utils.JSONDecodeAndPrintOnError(resp.Body, &channelList); err != nil {
		return "", err
	}

	if len(channelList.Channel) == 0 {
		return "", errors.New("no channels found")
	}

	// Decreasing sort by viewers. Probability that the channel with the most viewers is online is higher.
	slices.SortFunc(channelList.Channel, func(i, j GetChannelListChannel) int {
		icount, _ := i.Count.Float64()
		jcount, _ := j.Count.Float64()
		return int(jcount - icount)
	})

	for _, channel := range channelList.Channel {
		count, _ := channel.Count.Float64()
		// If the channel is not restricted and has less than 500 viewers, we can use it.
		if channel.Login.String() == "0" && count < 500 {
			return channel.ID, nil
		}
	}
	return "", errors.New("no unrestricted channels found")
}

// FindRestrictedStream finds the first restricted stream.
func (c *Client) FindRestrictedStream(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fc2ChannelListURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("non-ok http code returned: %d", resp.StatusCode)
	}

	var channelList GetChannelListResponse
	if err := utils.JSONDecodeAndPrintOnError(resp.Body, &channelList); err != nil {
		return "", err
	}

	if len(channelList.Channel) == 0 {
		return "", errors.New("no channels found")
	}

	// Decreasing sort by viewers. Probability that the channel with the most viewers is online is higher.
	slices.SortFunc(channelList.Channel, func(i, j GetChannelListChannel) int {
		icount, _ := i.Count.Float64()
		jcount, _ := j.Count.Float64()
		return int(jcount - icount)
	})

	for _, channel := range channelList.Channel {
		if channel.Login.String() == "1" {
			return channel.ID, nil
		}
	}
	return "", errors.New("no restricted channels found")
}
