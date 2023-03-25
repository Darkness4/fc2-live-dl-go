//go:build integration

package remux_test

import (
	"io"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/remux"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	err := remux.Do("input.ts", "output.mp4", false)
	require.Equal(t, io.EOF, err)

	err = remux.Do("input.ts", "output.m4a", true)
	require.Equal(t, io.EOF, err)
}
