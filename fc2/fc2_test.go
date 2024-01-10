package fc2

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	params := &DefaultParams
	params.OutFormat = filepath.Join(dir, DefaultParams.OutFormat)
	fc2 := New(http.DefaultClient, &DefaultParams, "000000")
	fName, name, err := fc2.prepareFile(&GetMetaData{}, "combined.mp4")
	fmt.Println(fName, name)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(fName, []byte("test"), 0o600))
	fName, name, err = fc2.prepareFile(&GetMetaData{}, "combined.mp4")
	require.NoError(t, err)
	fmt.Println(fName, name)
}
