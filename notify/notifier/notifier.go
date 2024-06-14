// Package notifier provides functions to notify the user about the status of the download.
package notifier

import (
	"context"

	"github.com/Darkness4/fc2-live-dl-go/notify"
)

// Notifier is the notifier used to notify the user about the status of the download.
var Notifier *notify.FormatedNotifier = notify.NewFormatedNotifier(
	notify.NewDummyNotifier(),
	notify.DefaultNotificationFormats,
)

// NotifyConfigReloaded notifies the user that the configuration has been reloaded.
func NotifyConfigReloaded(ctx context.Context) error {
	return Notifier.NotifyConfigReloaded(ctx)
}

// NotifyLoginFailed notifies the user that the login has failed.
func NotifyLoginFailed(ctx context.Context, capture error) error {
	return Notifier.NotifyLoginFailed(ctx, capture)
}

// NotifyPanicked notifies the user that the download has panicked.
func NotifyPanicked(ctx context.Context, capture any) error {
	return Notifier.NotifyPanicked(ctx, capture)
}

// NotifyIdle notifies the user that the stream is idle.
func NotifyIdle(ctx context.Context, channelID string, labels map[string]string) error {
	return Notifier.NotifyIdle(ctx, channelID, labels)
}

// NotifyPreparingFiles notifies the user that the program is preparing the files for the stream.
func NotifyPreparingFiles(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	return Notifier.NotifyPreparingFiles(ctx, channelID, labels, metadata)
}

// NotifyDownloading notifies the user that the program is downloading the stream.
func NotifyDownloading(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	return Notifier.NotifyDownloading(ctx, channelID, labels, metadata)
}

// NotifyPostProcessing notifies the user that the program is post processing the stream.
func NotifyPostProcessing(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	return Notifier.NotifyPostProcessing(ctx, channelID, labels, metadata)
}

// NotifyFinished notifies the user that the program has finished downloading the stream.
func NotifyFinished(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	return Notifier.NotifyFinished(ctx, channelID, labels, metadata)
}

// NotifyError notifies the user that the program has encountered an error.
func NotifyError(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	err error,
) error {
	return Notifier.NotifyError(ctx, channelID, labels, err)
}

// NotifyCanceled notifies the user that the program has canceled the download.
func NotifyCanceled(
	ctx context.Context,
	channelID string,
	labels map[string]string,
) error {
	return Notifier.NotifyCanceled(ctx, channelID, labels)
}

// NotifyUpdateAvailable notifies the user that an update is available.
func NotifyUpdateAvailable(ctx context.Context, version string) error {
	return Notifier.NotifyUpdateAvailable(ctx, version)
}
