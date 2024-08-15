//go:build integration

package concat_test

import (
	"context"
	_ "net/http/pprof"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/telemetry"
	"github.com/Darkness4/fc2-live-dl-go/video/concat"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	ctx := context.Background()
	shut, err := telemetry.SetupOTELSDK(ctx, telemetry.WithStdout())
	defer shut(ctx)
	require.NoError(t, err)
	err = concat.Do(ctx, "output.mp4", []string{"input.ts", "input.mp4"})
	require.NoError(t, err)
}

func TestWithPrefix(t *testing.T) {
	err := concat.WithPrefix(
		context.Background(),
		"mp4",
		"input",
		concat.IgnoreExtension(),
	)
	require.NoError(t, err)
}
