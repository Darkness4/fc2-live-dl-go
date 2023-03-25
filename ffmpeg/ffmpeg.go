package ffmpeg

import (
	"context"
	"os"
	"os/exec"

	"github.com/Darkness4/fc2-live-dl-go/logger"
	"go.uber.org/zap"
)

func init() {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		logger.I.Panic("ffmpeg not in PATH", zap.Error(err))
	}
}

func Exec(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	cmd.Stderr = os.Stdout

	logger.I.Info("ffmpeg start", zap.Strings("args", args))
	if err := cmd.Run(); err != nil {
		return err
	}

	logger.I.Info("ffmpeg finished", zap.Strings("args", args))

	return nil
}

func RemuxStream(ctx context.Context, inFile string, outFile string, extraFlags ...string) error {
	muxFlags := []string{
		"-y",
		"-hide_banner",
		"-stats",
		"-i",
		inFile,
	}
	muxFlags = append(muxFlags, extraFlags...)
	muxFlags = append(
		muxFlags,
		"-c",
		"copy",
		"-movflags",
		"faststart",
		outFile,
	)
	return Exec(ctx, muxFlags...)
}
