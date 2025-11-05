package state_test

import (
	"errors"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/state"
	"github.com/stretchr/testify/require"
)

func TestSetChannelState(t *testing.T) {
	// Arrange
	s := &state.State{
		Channels: make(map[string]*state.ChannelState),
	}

	// Test
	s.SetChannelState(
		"test",
		state.DownloadStateDownloading,
		state.WithExtra(map[string]any{
			"metadata": "meta",
		}),
	)

	// Assert
	require.Equal(t, state.DownloadStateDownloading, s.GetChannelState("test"))
	require.Equal(t, map[string]any{
		"metadata": "meta",
	}, s.ReadState().Channels["test"].Extra)
}

func TestSetChannelError(t *testing.T) {
	// Arrange
	state := &state.State{
		Channels: make(map[string]*state.ChannelState),
	}

	// Test
	state.SetChannelError("test", errors.New("error1"))
	state.SetChannelError("test", errors.New("error2"))

	// Assert
	require.Equal(t, "error1", state.ReadState().Channels["test"].Errors[0].Error)
	require.Equal(t, "error2", state.ReadState().Channels["test"].Errors[1].Error)
}
