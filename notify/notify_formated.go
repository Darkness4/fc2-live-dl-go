package notify

import (
	"context"
	"strings"
	"text/template"

	"github.com/Darkness4/fc2-live-dl-go/utils/ptr"
)

// NotificationFormats is a collection of formats for notifications.
type NotificationFormats struct {
	ConfigReloaded  NotificationFormat `yaml:"configReloaded,omitempty"`
	LoginFailed     NotificationFormat `yaml:"loginFailed,omitempty"`
	Panicked        NotificationFormat `yaml:"panicked,omitempty"`
	Idle            NotificationFormat `yaml:"idle,omitempty"`
	PreparingFiles  NotificationFormat `yaml:"preparingFiles,omitempty"`
	Downloading     NotificationFormat `yaml:"downloading,omitempty"`
	PostProcessing  NotificationFormat `yaml:"postProcessing,omitempty"`
	Finished        NotificationFormat `yaml:"finished,omitempty"`
	Error           NotificationFormat `yaml:"error,omitempty"`
	Canceled        NotificationFormat `yaml:"canceled,omitempty"`
	UpdateAvailable NotificationFormat `yaml:"updateAvailable,omitempty"`
}

// NotificationFormat is a format for a notification.
type NotificationFormat struct {
	Enabled  *bool  `yaml:"enabled,omitempty"`
	Title    string `yaml:"title,omitempty"`
	Message  string `yaml:"message,omitempty"`
	Priority int    `yaml:"priority,omitempty"`
}

// NotificationTemplates is a collection of templates for notifications.
type NotificationTemplates struct {
	ConfigReloaded  NotificationTemplate
	LoginFailed     NotificationTemplate
	Panicked        NotificationTemplate
	Idle            NotificationTemplate
	PreparingFiles  NotificationTemplate
	Downloading     NotificationTemplate
	PostProcessing  NotificationTemplate
	Finished        NotificationTemplate
	Error           NotificationTemplate
	Canceled        NotificationTemplate
	UpdateAvailable NotificationTemplate
}

// NotificationTemplate is a template for a notification.
type NotificationTemplate struct {
	TitleTemplate   *template.Template
	MessageTemplate *template.Template
}

// DefaultNotificationFormats is the default notification formats.
var DefaultNotificationFormats = NotificationFormats{
	ConfigReloaded: NotificationFormat{
		Enabled:  ptr.Ref(true),
		Title:    "config reloaded",
		Message:  "",
		Priority: 10,
	},
	LoginFailed: NotificationFormat{
		Enabled:  ptr.Ref(true),
		Title:    "login failed",
		Message:  "{{ .Error }}",
		Priority: 10,
	},
	Panicked: NotificationFormat{
		Enabled:  ptr.Ref(true),
		Title:    "panicked",
		Message:  "{{ .Capture }}",
		Priority: 10,
	},
	Idle: NotificationFormat{
		Enabled: ptr.Ref(false),
		Title:   "watching {{ .ChannelID }}",
	},
	PreparingFiles: NotificationFormat{
		Enabled: ptr.Ref(false),
		Title:   "preparing files for {{ .MetaData.ProfileData.Name }}",
	},
	Downloading: NotificationFormat{
		Enabled:  ptr.Ref(true),
		Title:    "{{ .MetaData.ProfileData.Name }} is streaming",
		Message:  "{{ .MetaData.ChannelData.Title }}",
		Priority: 7,
	},
	PostProcessing: NotificationFormat{
		Enabled:  ptr.Ref(false),
		Title:    "post-processing {{ .MetaData.ProfileData.Name }}",
		Message:  "{{ .MetaData.ChannelData.Title }}",
		Priority: 7,
	},
	Finished: NotificationFormat{
		Enabled:  ptr.Ref(true),
		Title:    "{{ .MetaData.ProfileData.Name }} stream ended",
		Message:  "{{ .MetaData.ChannelData.Title }}",
		Priority: 7,
	},
	Error: NotificationFormat{
		Enabled:  ptr.Ref(true),
		Title:    "stream download of {{ .ChannelID }} failed",
		Message:  "{{ .Error }}",
		Priority: 10,
	},
	Canceled: NotificationFormat{
		Enabled:  ptr.Ref(true),
		Title:    "stream download of {{ .ChannelID }} canceled",
		Priority: 10,
	},
	UpdateAvailable: NotificationFormat{
		Enabled:  ptr.Ref(true),
		Title:    "update available ({{ .Version }})",
		Message:  "A new version ({{ .Version }}) of fc2-live-dl is available. Please update.",
		Priority: 7,
	},
}

func (old *NotificationFormat) applyNotificationFormatDefault(
	newFormat NotificationFormat,
) {
	if newFormat.Enabled != nil {
		old.Enabled = newFormat.Enabled
	}
	if newFormat.Title != "" {
		old.Title = newFormat.Title
	}
	if newFormat.Message != "" {
		old.Message = newFormat.Message
	}
	if newFormat.Priority != 0 {
		old.Priority = newFormat.Priority
	}
}

func applyNotificationFormatsDefault(newFormat NotificationFormats) NotificationFormats {
	formats := DefaultNotificationFormats
	formats.ConfigReloaded.applyNotificationFormatDefault(newFormat.ConfigReloaded)
	formats.LoginFailed.applyNotificationFormatDefault(newFormat.LoginFailed)
	formats.Panicked.applyNotificationFormatDefault(newFormat.Panicked)
	formats.Idle.applyNotificationFormatDefault(newFormat.Idle)
	formats.PreparingFiles.applyNotificationFormatDefault(newFormat.PreparingFiles)
	formats.Downloading.applyNotificationFormatDefault(newFormat.Downloading)
	formats.PostProcessing.applyNotificationFormatDefault(newFormat.PostProcessing)
	formats.Finished.applyNotificationFormatDefault(newFormat.Finished)
	formats.Error.applyNotificationFormatDefault(newFormat.Error)
	formats.Canceled.applyNotificationFormatDefault(newFormat.Canceled)
	formats.UpdateAvailable.applyNotificationFormatDefault(newFormat.UpdateAvailable)
	return formats
}

func initializeTemplate(format NotificationFormat) NotificationTemplate {
	return NotificationTemplate{
		TitleTemplate:   template.Must(template.New("ConfigReloaded").Parse(format.Title)),
		MessageTemplate: template.Must(template.New("ConfigReloaded").Parse(format.Message)),
	}
}

func initializeTemplates(formats NotificationFormats) NotificationTemplates {
	return NotificationTemplates{
		ConfigReloaded:  initializeTemplate(formats.ConfigReloaded),
		LoginFailed:     initializeTemplate(formats.LoginFailed),
		Panicked:        initializeTemplate(formats.Panicked),
		Idle:            initializeTemplate(formats.Idle),
		PreparingFiles:  initializeTemplate(formats.PreparingFiles),
		Downloading:     initializeTemplate(formats.Downloading),
		PostProcessing:  initializeTemplate(formats.PostProcessing),
		Finished:        initializeTemplate(formats.Finished),
		Error:           initializeTemplate(formats.Error),
		Canceled:        initializeTemplate(formats.Canceled),
		UpdateAvailable: initializeTemplate(formats.UpdateAvailable),
	}
}

// FormatedNotifier is a notifier that formats the notifications.
type FormatedNotifier struct {
	BaseNotifier
	NotificationFormats
	NotificationTemplates
}

// NewFormatedNotifier creates a new FormatedNotifier.
func NewFormatedNotifier(notifier BaseNotifier, formats NotificationFormats) *FormatedNotifier {
	formats = applyNotificationFormatsDefault(formats)
	return &FormatedNotifier{
		BaseNotifier:          notifier,
		NotificationFormats:   formats,
		NotificationTemplates: initializeTemplates(formats),
	}
}

// NotifyDownloading sends a notification that the download is starting.
func (n *FormatedNotifier) NotifyDownloading(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	if n.NotificationFormats.Downloading.Enabled == nil ||
		(n.NotificationFormats.Downloading.Enabled != nil &&
			!(*n.NotificationFormats.Downloading.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.Downloading.TitleTemplate.Execute(
		&titleSB,
		struct {
			ChannelID string
			MetaData  any
			Labels    map[string]string
		}{
			ChannelID: channelID,
			MetaData:  metadata,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.Downloading.MessageTemplate.Execute(
		&messageSB,
		struct {
			ChannelID string
			MetaData  any
			Labels    map[string]string
		}{
			ChannelID: channelID,
			MetaData:  metadata,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.Downloading.Priority,
	)
}

// NotifyError sends a notification that the download encountered an error.
func (n *FormatedNotifier) NotifyError(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	err error,
) error {
	if n.NotificationFormats.Error.Enabled == nil ||
		(n.NotificationFormats.Error.Enabled != nil &&
			!(*n.NotificationFormats.Error.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.Error.TitleTemplate.Execute(
		&titleSB,
		struct {
			ChannelID string
			Error     error
			Labels    map[string]string
		}{
			ChannelID: channelID,
			Error:     err,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.Error.MessageTemplate.Execute(
		&messageSB,
		struct {
			ChannelID string
			Error     error
			Labels    map[string]string
		}{
			ChannelID: channelID,
			Error:     err,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.Error.Priority,
	)
}

// NotifyFinished sends a notification that the download is finished.
func (n *FormatedNotifier) NotifyFinished(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	if n.NotificationFormats.Finished.Enabled == nil ||
		(n.NotificationFormats.Finished.Enabled != nil &&
			!(*n.NotificationFormats.Finished.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.Finished.TitleTemplate.Execute(
		&titleSB,
		struct {
			ChannelID string
			MetaData  any
			Labels    map[string]string
		}{
			ChannelID: channelID,
			MetaData:  metadata,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.Finished.MessageTemplate.Execute(
		&messageSB,
		struct {
			ChannelID string
			MetaData  any
			Labels    map[string]string
		}{
			ChannelID: channelID,
			MetaData:  metadata,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.Finished.Priority,
	)
}

// NotifyConfigReloaded sends a notification that the config was reloaded.
func (n *FormatedNotifier) NotifyConfigReloaded(ctx context.Context) error {
	if n.NotificationFormats.ConfigReloaded.Enabled == nil ||
		(n.NotificationFormats.ConfigReloaded.Enabled != nil &&
			!(*n.NotificationFormats.ConfigReloaded.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.ConfigReloaded.TitleTemplate.Execute(
		&titleSB,
		struct{}{},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.ConfigReloaded.MessageTemplate.Execute(
		&messageSB,
		struct{}{},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.ConfigReloaded.Priority,
	)
}

// NotifyIdle sends a notification that the download is idle.
func (n *FormatedNotifier) NotifyIdle(
	ctx context.Context,
	channelID string,
	labels map[string]string,
) error {
	if n.NotificationFormats.Idle.Enabled == nil ||
		(n.NotificationFormats.Idle.Enabled != nil &&
			!(*n.NotificationFormats.Idle.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.Idle.TitleTemplate.Execute(
		&titleSB,
		struct {
			ChannelID string
			Labels    map[string]string
		}{
			ChannelID: channelID,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.Idle.MessageTemplate.Execute(
		&messageSB,
		struct {
			ChannelID string
			Labels    map[string]string
		}{
			ChannelID: channelID,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.Idle.Priority,
	)
}

// NotifyLoginFailed sends a notification that the login failed.
func (n *FormatedNotifier) NotifyLoginFailed(ctx context.Context, capture error) error {
	if n.NotificationFormats.LoginFailed.Enabled == nil ||
		(n.NotificationFormats.LoginFailed.Enabled != nil &&
			!(*n.NotificationFormats.LoginFailed.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.LoginFailed.TitleTemplate.Execute(
		&titleSB,
		struct {
			Error error
		}{
			Error: capture,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.LoginFailed.MessageTemplate.Execute(
		&messageSB,
		struct {
			Error any
		}{
			Error: capture,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.LoginFailed.Priority,
	)
}

// NotifyPanicked sends a notification that the download panicked.
func (n *FormatedNotifier) NotifyPanicked(ctx context.Context, capture any) error {
	if n.NotificationFormats.Panicked.Enabled == nil ||
		(n.NotificationFormats.Panicked.Enabled != nil &&
			!(*n.NotificationFormats.Panicked.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.Panicked.TitleTemplate.Execute(
		&titleSB,
		struct {
			Capture any
		}{
			Capture: capture,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.Panicked.MessageTemplate.Execute(
		&messageSB,
		struct {
			Capture any
		}{
			Capture: capture,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.Panicked.Priority,
	)
}

// NotifyPreparingFiles sends a notification that the download is preparing files.
func (n *FormatedNotifier) NotifyPreparingFiles(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	if n.NotificationFormats.PreparingFiles.Enabled == nil ||
		(n.NotificationFormats.PreparingFiles.Enabled != nil &&
			!(*n.NotificationFormats.PreparingFiles.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.PreparingFiles.TitleTemplate.Execute(
		&titleSB,
		struct {
			ChannelID string
			MetaData  any
			Labels    map[string]string
		}{
			ChannelID: channelID,
			MetaData:  metadata,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.PreparingFiles.MessageTemplate.Execute(
		&messageSB,
		struct {
			ChannelID string
			MetaData  any
			Labels    map[string]string
		}{
			ChannelID: channelID,
			MetaData:  metadata,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.PreparingFiles.Priority,
	)
}

// NotifyPostProcessing sends a notification that the download is post-processing.
func (n *FormatedNotifier) NotifyPostProcessing(
	ctx context.Context,
	channelID string,
	labels map[string]string,
	metadata any,
) error {
	if n.NotificationFormats.PostProcessing.Enabled == nil ||
		(n.NotificationFormats.PostProcessing.Enabled != nil &&
			!(*n.NotificationFormats.PostProcessing.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.PostProcessing.TitleTemplate.Execute(
		&titleSB,
		struct {
			ChannelID string
			MetaData  any
			Labels    map[string]string
		}{
			ChannelID: channelID,
			MetaData:  metadata,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.PostProcessing.MessageTemplate.Execute(
		&messageSB,
		struct {
			ChannelID string
			MetaData  any
			Labels    map[string]string
		}{
			ChannelID: channelID,
			MetaData:  metadata,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.PostProcessing.Priority,
	)
}

// NotifyCanceled sends a notification that the download was canceled.
func (n *FormatedNotifier) NotifyCanceled(
	ctx context.Context,
	channelID string,
	labels map[string]string,
) error {
	if n.NotificationFormats.Canceled.Enabled == nil ||
		(n.NotificationFormats.Canceled.Enabled != nil &&
			!(*n.NotificationFormats.Canceled.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.Canceled.TitleTemplate.Execute(
		&titleSB,
		struct {
			ChannelID string
			Labels    map[string]string
		}{
			ChannelID: channelID,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.Canceled.MessageTemplate.Execute(
		&messageSB,
		struct {
			ChannelID string
			MetaData  any
			Labels    map[string]string
		}{
			ChannelID: channelID,
			Labels:    labels,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.Canceled.Priority,
	)
}

// NotifyUpdateAvailable sends a notification that an update is available.
func (n *FormatedNotifier) NotifyUpdateAvailable(
	ctx context.Context,
	version string,
) error {
	if n.NotificationFormats.UpdateAvailable.Enabled == nil ||
		(n.NotificationFormats.UpdateAvailable.Enabled != nil &&
			!(*n.NotificationFormats.UpdateAvailable.Enabled)) {
		return nil
	}
	var titleSB strings.Builder
	var messageSB strings.Builder
	if err := n.NotificationTemplates.UpdateAvailable.TitleTemplate.Execute(
		&titleSB,
		struct {
			Version string
		}{
			Version: version,
		},
	); err != nil {
		return err
	}
	if err := n.NotificationTemplates.UpdateAvailable.MessageTemplate.Execute(
		&messageSB,
		struct {
			Version string
		}{
			Version: version,
		},
	); err != nil {
		return err
	}
	return n.Notify(
		ctx,
		titleSB.String(),
		messageSB.String(),
		n.NotificationFormats.UpdateAvailable.Priority,
	)
}
