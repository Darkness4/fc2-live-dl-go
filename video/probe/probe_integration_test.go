//go:build integration

package probe_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/video/probe"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	err := probe.Do([]string{"input.ts", "input.1.ts"})
	require.NoError(t, err)
}

func TestIsVideo(t *testing.T) {
	ret, err := probe.ContainsVideoOrAudio("input.ts")
	require.NoError(t, err)
	require.True(t, ret)
}
