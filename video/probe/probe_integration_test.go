//go:build integration

package probe_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/video/probe"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	err := probe.Do("input.mp4", "input.1.mp4")
	require.Equal(t, nil, err)
}
