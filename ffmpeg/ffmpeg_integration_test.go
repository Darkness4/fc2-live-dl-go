//go:build integration

package ffmpeg_test

import (
	"context"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/ffmpeg"
	"github.com/stretchr/testify/require"
)

func TestExec(t *testing.T) {
	ctx := context.Background()
	err := ffmpeg.RemuxStream(ctx, "input.ts", "output.mp4")
	require.NoError(t, err)
}
