//go:build integration

package concat_test

import (
	"context"
	_ "net/http/pprof"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/video/concat"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	err := concat.Do(context.Background(), "output.mp4", []string{"input.ts", "input.mp4"})
	require.NoError(t, err)
}

func TestWithPrefix(t *testing.T) {
	err := concat.WithPrefix(
		context.Background(),
		"m4a",
		"input",
		concat.IgnoreExtension(),
		concat.WithAudioOnly(),
	)
	require.NoError(t, err)
}
