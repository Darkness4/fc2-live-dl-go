package fc2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/notify/notifier"
	"github.com/rs/zerolog/log"
)

type LoginOption func(*LoginOptions)

type LoginOptions struct {
	client *http.Client
}

func WithHTTPClient(client *http.Client) LoginOption {
	return func(lo *LoginOptions) {
		lo.client = client
	}
}

func applyLoginOptions(opts []LoginOption) *LoginOptions {
	o := &LoginOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// memberLoginRegex is used to extract the next URL. No need for parsing HTML, just fetch the URL.
var memberLoginRegex = regexp.MustCompile(
	`href="(http://live\.fc2\.com/member_login/\?uid=[^&]+&cc=[^"]+)"`,
)

var usernameRegex = regexp.MustCompile(`<span class="m-hder01_uName">(.*?)</span>`)

// Login to FC2 and fill the CookieJar.
//
// You need to probably need to
func Login(ctx context.Context, opts ...LoginOption) error {
	o := applyLoginOptions(opts)
	client := http.DefaultClient
	if o.client != nil {
		client = o.client
	}

	// Phase 1: Redirect to https://id.fc2.com
	var memberLoginURL string
	if err := func() error {
		req, err := http.NewRequestWithContext(ctx, "GET", "https://live.fc2.com/login/", nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
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
		resp, err := client.Do(req)
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

func LoginLoop(
	ctx context.Context,
	duration time.Duration,
	opts ...LoginOption,
) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := Login(ctx, opts...); err != nil {
				if err := notifier.NotifyLoginFailed(ctx, err); err != nil {
					log.Err(err).Msg("notify failed")
				}
				log.Err(err).
					Msg("failed to login to id.fc2.com, we will try again, but you should extract new cookies")
			}
		case <-ctx.Done():
			return
		}
	}
}
