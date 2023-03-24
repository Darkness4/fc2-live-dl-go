package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/Darkness4/fc2-live-dl-go/logger"
	"go.uber.org/zap"
)

func Exec(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	logger.I.Info("ffmpeg start", zap.Strings("args", args))
	if err := cmd.Start(); err != nil {
		return err
	}

	reader := bufio.NewReader(stderr)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				logger.I.Error("ffmpeg failed to read bytes", zap.Error(err))
			}
			break
		}
		fmt.Println(string(line))
	}

	if err := cmd.Wait(); err != nil {
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
