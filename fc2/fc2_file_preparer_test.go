package fc2_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Darkness4/fc2-live-dl-go/fc2"
	"github.com/Darkness4/fc2-live-dl-go/fc2/api"
	"github.com/stretchr/testify/require"
)

func TestPrepareFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	format := fmt.Sprintf("%s/{{ .Title }}.{{ .Ext }}", dir)
	fName, err := fc2.PrepareFile(format, &api.GetMetaData{
		ChannelData: api.ChannelData{
			Title: "test",
		},
	}, fc2.DefaultParams.Labels, "mp4")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s/test.mp4", dir), fName)

	require.NoError(t, os.WriteFile(fName, []byte("test"), 0o600))

	fName, err = fc2.PrepareFile(format, &api.GetMetaData{
		ChannelData: api.ChannelData{
			Title: "test",
		},
	}, fc2.DefaultParams.Labels, "mp4")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s/test.1.mp4", dir), fName)
}
