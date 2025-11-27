// Package notify provides the notifier for the notification.
package notify

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
)

// BaseNotifier is the interface for the notifier.
type BaseNotifier interface {
	Notify(
		ctx context.Context,
		title string,
		message string,
		priority int,
	) error
}

// DummyNotifier is the notifier which prints in the logs.
type DummyNotifier struct{}

// NewDummyNotifier creates a new Dummy notifier.
func NewDummyNotifier() *DummyNotifier {
	return &DummyNotifier{}
}

// Notify sends a notification over nothing.
func (*DummyNotifier) Notify(
	_ context.Context,
	title string,
	message string,
	_ int,
) error {
	fmt.Printf("dummy notify:\ntitle: %s\nmessage:%s\n", title, message)
	return nil
}

// ShoutrrrOptions is the options for the Shoutrrr notifier.
type ShoutrrrOptions struct {
	includeTitleInMessage bool
	noPriority            bool
}

// ShoutrrrOption is the option for the Shoutrrr notifier.
type ShoutrrrOption func(*ShoutrrrOptions)

// IncludeTitleInMessage is an option to include the title in the message.
func IncludeTitleInMessage(value ...bool) ShoutrrrOption {
	return func(no *ShoutrrrOptions) {
		no.includeTitleInMessage = true
		if len(value) > 0 {
			no.includeTitleInMessage = value[0]
		}
	}
}

// NoPriority is an option to not include the priority.
func NoPriority(value ...bool) ShoutrrrOption {
	return func(no *ShoutrrrOptions) {
		no.noPriority = true
		if len(value) > 0 {
			no.noPriority = value[0]
		}
	}
}

func applyShoutrrrOptions(opts []ShoutrrrOption) *ShoutrrrOptions {
	o := &ShoutrrrOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// Shoutrrr is the notifier for shoutrrr.
type Shoutrrr struct {
	*router.ServiceRouter
	opts *ShoutrrrOptions
}

// NewShoutrrr creates a new Shoutrrr notifier.
func NewShoutrrr(urls []string, opts ...ShoutrrrOption) *Shoutrrr {
	r, err := shoutrrr.CreateSender(urls...)
	if err != nil {
		panic(err.Error())
	}
	o := applyShoutrrrOptions(opts)
	return &Shoutrrr{
		ServiceRouter: r,
		opts:          o,
	}
}

// Notify sends a notification with Shoutrrr.
func (n *Shoutrrr) Notify(
	ctx context.Context,
	title string,
	message string,
	priority int,
) error {
	if message == "" {
		message = title
	}
	if n.opts.includeTitleInMessage {
		message = fmt.Sprintf("**%s**\n\n%s", title, message)
	}
	params := types.Params{
		"title": fmt.Sprintf("fc2-live-dl-go: %s", title),
	}
	if !n.opts.noPriority {
		params["priority"] = strconv.Itoa(priority)
	}
	errCh := n.SendAsync(message, &params)
	errs := []error{}

	for {
		select {
		case err, ok := <-errCh:
			if !ok {
				return errors.Join(errs...)
			}
			if err != nil {
				errs = append(errs, err)
			}
		case <-ctx.Done():
			return errors.Join(errs...)
		}
	}
}
