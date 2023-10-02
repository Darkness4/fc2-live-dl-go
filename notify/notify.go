package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
)

type Priority int

const (
	PriorityLow    = 0
	PriorityMedium = 7
	PriorityHigh   = 10
)

type Notifier interface {
	Notify(ctx context.Context, title string, message string, priority Priority) error
}

type dummyNotifier struct{}

func (*dummyNotifier) Notify(
	ctx context.Context,
	title string,
	message string,
	priority Priority,
) error {
	return nil
}

func NewDummyNotifier() Notifier {
	return &dummyNotifier{}
}

type goNotifierMessage struct {
	Title    string `json:"title"`
	Priority int    `json:"priority"`
	Message  string `json:"message"`
}

type gonotifier struct {
	*http.Client
	endpoint string
	token    string
}

func NewGoNotifier(client *http.Client, endpoint string, token string) Notifier {
	return &gonotifier{
		Client:   client,
		endpoint: endpoint,
		token:    token,
	}
}

func (n *gonotifier) Notify(
	ctx context.Context,
	title string,
	message string,
	priority Priority,
) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if message == "" {
		message = title
	}

	var bb bytes.Buffer
	if err := json.NewEncoder(&bb).Encode(goNotifierMessage{
		Title:    fmt.Sprintf("fc2-live-dl-go: %s", title),
		Message:  message,
		Priority: int(priority),
	}); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", n.endpoint+"/message", &bb)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", n.token))
	req = req.WithContext(ctx)

	resp, err := n.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notification failed: %s", string(out))
	}

	return nil
}

type shoutrrrNotifier struct {
	*router.ServiceRouter
}

func NewShoutrrrNotifier(urls ...string) Notifier {
	r, err := shoutrrr.CreateSender(urls...)
	if err != nil {
		panic(err.Error())
	}
	return &shoutrrrNotifier{r}
}

func (n *shoutrrrNotifier) Notify(
	ctx context.Context,
	title string,
	message string,
	priority Priority,
) error {
	if message == "" {
		message = title
	}
	errs := n.Send(message, &types.Params{
		"title":    fmt.Sprintf("fc2-live-dl-go: %s", title),
		"priority": strconv.Itoa(int(priority)),
	})
	return errors.Join(errs...)
}
