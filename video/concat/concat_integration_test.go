//go:build integration

package concat_test

import (
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/video/concat"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	err := concat.Do("output.mp4", []string{"input.mp4", "input.1.mp4"})
	require.Equal(t, nil, err)
}
