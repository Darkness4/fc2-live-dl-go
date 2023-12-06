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

type BaseNotifier interface {
	Notify(
		ctx context.Context,
		title string,
		message string,
		priority int,
	) error
}

type dummyNotifier struct{}

func NewDummyNotifier() BaseNotifier {
	return &dummyNotifier{}
}

func (*dummyNotifier) Notify(
	ctx context.Context,
	title string,
	message string,
	priority int,
) error {
	fmt.Printf("dummy notify:\ntitle: %s\nmessage:%s\n", title, message)
	return nil
}

type ShoutrrrOptions struct {
	includeTitleInMessage bool
	noPriority            bool
}

type ShoutrrrOption func(*ShoutrrrOptions)

func IncludeTitleInMessage(value ...bool) ShoutrrrOption {
	return func(no *ShoutrrrOptions) {
		no.includeTitleInMessage = true
		if len(value) > 0 {
			no.includeTitleInMessage = value[0]
		}
	}
}

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

type Shoutrrr struct {
	*router.ServiceRouter
	opts *ShoutrrrOptions
}

func NewShoutrrr(urls []string, opts ...ShoutrrrOption) BaseNotifier {
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
	errs := n.Send(message, &params)
	return errors.Join(errs...)
}
