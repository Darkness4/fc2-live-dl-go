package notifier

import (
	"context"

	"github.com/Darkness4/fc2-live-dl-go/notify"
)

var Notifier notify.FormatedNotifier = notify.NewFormatedNotifier(
	notify.NewDummyNotifier(),
	notify.DefaultNotificationFormats,
)

func NotifyConfigReloaded(ctx context.Context) error {
	return Notifier.NotifyConfigReloaded(ctx)
}

func NotifyLoginFailed(ctx context.Context, capture error) error {
	return Notifier.NotifyLoginFailed(ctx, capture)
}

func NotifyPanicked(ctx context.Context, capture any) error {
	return Notifier.NotifyPanicked(ctx, capture)
}

func NotifyIdle(ctx context.Context, channelID string, labels map[string]string) error {
	return Notifier.NotifyIdle(ctx, channelID, labels)
}

func NotifyPreparingFiles(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	return Notifier.NotifyPreparingFiles(ctx, channelID, labels, metadata)
}

func NotifyDownloading(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	return Notifier.NotifyDownloading(ctx, channelID, labels, metadata)
}

func NotifyPostProcessing(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	return Notifier.NotifyPostProcessing(ctx, channelID, labels, metadata)
}

func NotifyFinished(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	return Notifier.NotifyFinished(ctx, channelID, labels, metadata)
}

func NotifyError(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	err error,
) error {
	return Notifier.NotifyError(ctx, channelID, labels, err)
}

func NotifyCanceled(
	ctx context.Context,
	channelID string,
	labels map[string]string,
) error {
	return Notifier.NotifyCanceled(ctx, channelID, labels)
}
