//go:build integration

package remux_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/video/remux"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	err := remux.Do("output.mp4", "input.ts")
	require.Equal(t, nil, err)

	err = remux.Do("output.m4a", "input.ts", remux.WithAudioOnly())
	require.Equal(t, nil, err)
}
