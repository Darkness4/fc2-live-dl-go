package state

import (
	"encoding/json"
	"sync"
	"time"
)

type State struct {
	Channels map[string]*ChannelState `json:"channels"`

	mu sync.RWMutex
}

type ChannelState struct {
	DownloadState DownloadState          `json:"state"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
	Errors        []DownloadError        `json:"errors_log"`
}

type DownloadError struct {
	Timestamp string `json:"timestamp"`
	Error     string `json:"error"`
}

type DownloadState int

const (
	DownloadStateUnspecified DownloadState = iota
	DownloadStateIdle
	DownloadStatePreparingFiles
	DownloadStateDownloading
	DownloadStatePostProcessing
	DownloadStateFinished
	DownloadStateCanceled
)

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

func (d DownloadState) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *DownloadState) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	*d = DownloadStateFromString(s)

	return nil
}

var (
	DefaultState = State{
		Channels: make(map[string]*ChannelState),
	}
)

func (s *State) GetChannelState(name string) DownloadState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if c, ok := s.Channels[name]; ok {
		return c.DownloadState
	}
	return DownloadStateUnspecified
}

func (s *State) SetChannelState(name string, state DownloadState, extra map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.Channels[name]; !ok {
		s.Channels[name] = &ChannelState{
			Errors: make([]DownloadError, 0),
		}
	}
	s.Channels[name].DownloadState = state
	s.Channels[name].Extra = extra
}

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

func (s *State) ReadState() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s
}
