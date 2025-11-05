// Package state implements state for debugging.
package state

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// State represents the state of the program.
type State struct {
	Channels map[string]*ChannelState `json:"channels"`

	mu sync.RWMutex
}

// ChannelState represents the state of a channel.
type ChannelState struct {
	DownloadState DownloadState     `json:"state"`
	Extra         map[string]any    `json:"extra,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Errors        []DownloadError   `json:"errors_log"`
}

// DownloadError represents an error during a download.
type DownloadError struct {
	Timestamp string `json:"timestamp"`
	Error     string `json:"error"`
}

// DownloadState represents the state of a download.
type DownloadState int

const (
	// DownloadStateUnspecified is used when the download state is unspecified.
	DownloadStateUnspecified DownloadState = iota
	// DownloadStateIdle is used when the download is idle.
	DownloadStateIdle
	// DownloadStatePreparingFiles is used when the download is preparing files.
	DownloadStatePreparingFiles
	// DownloadStateDownloading is used when the download is downloading.
	DownloadStateDownloading
	// DownloadStatePostProcessing is used when the download is post processing.
	DownloadStatePostProcessing
	// DownloadStateFinished is used when the download is finished.
	DownloadStateFinished
	// DownloadStateCanceled is used when the download is canceled.
	DownloadStateCanceled
)

// String returns a string representation of a DownloadState.
func (d DownloadState) String() string {
	switch d {
	case DownloadStateUnspecified:
		return "UNSPECIFIED"
	case DownloadStateIdle:
		return "IDLE"
	case DownloadStatePreparingFiles:
		return "PREPARING_FILES"
	case DownloadStateDownloading:
		return "DOWNLOADING"
	case DownloadStatePostProcessing:
		return "POST_PROCESSING"
	case DownloadStateFinished:
		return "FINISHED"
	case DownloadStateCanceled:
		return "CANCELED"
	}
	return "UNSPECIFIED"
}

// DownloadStateFromString returns a DownloadState from a string.
func DownloadStateFromString(s string) DownloadState {
	switch s {
	default:
		return DownloadStateUnspecified
	case "IDLE":
		return DownloadStateIdle
	case "PREPARING_FILES":
		return DownloadStatePreparingFiles
	case "DOWNLOADING":
		return DownloadStateDownloading
	case "POST_PROCESSING":
		return DownloadStatePostProcessing
	case "FINISHED":
		return DownloadStateFinished
	case "CANCELED":
		return DownloadStateCanceled
	}
}

// MarshalJSON marshals a DownloadState into a string.
func (d DownloadState) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// UnmarshalJSON unmarshals a string into a DownloadState.
func (d *DownloadState) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	*d = DownloadStateFromString(s)

	return nil
}

var (
	// DefaultState is the default state.
	DefaultState = State{
		Channels: make(map[string]*ChannelState),
	}
)

// GetChannelState returns the state for a channel.
func (s *State) GetChannelState(name string) DownloadState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if c, ok := s.Channels[name]; ok {
		return c.DownloadState
	}
	return DownloadStateUnspecified
}

type setChannelStateOptions struct {
	labels map[string]string
	extra  map[string]any
}

// SetChannelStateOptions represents options for SetChannelState.
type SetChannelStateOptions func(*setChannelStateOptions)

// WithLabels sets labels for a channel.
func WithLabels(labels map[string]string) SetChannelStateOptions {
	return func(o *setChannelStateOptions) {
		o.labels = labels
	}
}

// WithExtra sets extra data for a channel.
func WithExtra(extra map[string]any) SetChannelStateOptions {
	return func(o *setChannelStateOptions) {
		o.extra = extra
	}
}

// SetChannelState sets the state for a channel.
func (s *State) SetChannelState(
	name string,
	state DownloadState,
	opts ...SetChannelStateOptions,
) {
	o := &setChannelStateOptions{}
	for _, opt := range opts {
		opt(o)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Channels[name]; !ok {
		s.Channels[name] = &ChannelState{
			Errors: make([]DownloadError, 0),
		}
	}
	s.Channels[name].DownloadState = state
	s.Channels[name].Extra = o.extra
	s.Channels[name].Labels = o.labels
	setStateMetrics(context.Background(), name, state, o.labels)
}

// SetChannelError sets an error for a channel.
func (s *State) SetChannelError(name string, err error) {
	if err == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Channels[name]; !ok {
		s.Channels[name] = &ChannelState{
			Errors: make([]DownloadError, 0),
		}
	}

	s.Channels[name].Errors = append(s.Channels[name].Errors, DownloadError{
		Timestamp: time.Now().UTC().String(),
		Error:     err.Error(),
	})
}

// ReadState returns the current state.
func (s *State) ReadState() *State {
	return s
}
