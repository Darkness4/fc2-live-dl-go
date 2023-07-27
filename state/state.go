package state

import (
	"encoding/json"
	"sync"
)

type State struct {
	Channels map[string]*ChannelState `json:"channels"`
}

type ChannelState struct {
	DownloadState DownloadState `json:"state"`
	LastError     error         `json:"last_error"`
}

type DownloadState int

const (
	DownloadStateUnspecified DownloadState = iota
	DownloadStateIdle
	DownloadStateDownloading
)

func (d DownloadState) String() string {
	switch d {
	case DownloadStateUnspecified:
		return "UNSPECIFIED"
	case DownloadStateIdle:
		return "IDLE"
	case DownloadStateDownloading:
		return "DOWNLOADING"
	}
	return "UNSPECIFIED"
}

func DownloadStateFromString(s string) DownloadState {
	switch s {
	default:
		return DownloadStateUnspecified
	case "IDLE":
		return DownloadStateIdle
	case "DOWNLOADING":
		return DownloadStateDownloading
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
	state = State{
		Channels: make(map[string]*ChannelState),
	}
	mu sync.Mutex
)

func SetChannelState(name string, s DownloadState) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := state.Channels[name]; !ok {
		state.Channels[name] = &ChannelState{}
	}
	state.Channels[name].DownloadState = s
}

func SetChannelError(name string, err error) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := state.Channels[name]; !ok {
		state.Channels[name] = &ChannelState{}
	}
	state.Channels[name].LastError = err
}

func ReadState() State {
	return state
}
