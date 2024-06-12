package probe_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/video/probe"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	tests := []string{
		"input.ts",
		"input.aac",
		"input.mp4",
		"input.m4a",
	}

	err := probe.Do(tests)
	require.NoError(t, err)
}

func TestContainsVideoOrAudio(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"input.ts", true},
		{"input.aac", true},
		{"input.mp4", true},
		{"input.m4a", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ret, err := probe.ContainsVideoOrAudio(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, ret)
		})
	}
}

func TestIsMPEGTSOrAAC(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"input.ts", true},
		{"input.aac", true},
		{"input.mp4", false},
		{"input.m4a", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ret, err := probe.IsMPEGTSOrAAC(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, ret)
		})
	}
}
