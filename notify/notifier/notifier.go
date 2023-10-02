package notifier

import (
	"context"

	"github.com/Darkness4/fc2-live-dl-go/notify"
)

var Notifier notify.Notifier = notify.NewDummyNotifier()

func Notify(ctx context.Context, title string, message string, priority notify.Priority) error {
	return Notifier.Notify(ctx, title, message, priority)
}
