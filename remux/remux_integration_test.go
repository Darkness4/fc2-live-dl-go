//go:build integration

package remux_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/remux"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	err := remux.Do("input.ts", "output.mp4")
	require.Equal(t, nil, err)

	err = remux.Do("input.ts", "output.m4a", remux.WithAudioOnly())
	require.Equal(t, nil, err)
}
