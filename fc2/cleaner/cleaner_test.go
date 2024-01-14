package cleaner_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/Darkness4/fc2-live-dl-go/fc2/cleaner"
	"github.com/stretchr/testify/require"
)

func TestScan(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	files := []string{
		"test.ts",
		"test.0.ts",
		"test.deleteme.ts",
		"test.deleteme.deleteme.ts",
		"test.mp4",
		"test.1.ts",
		"testnodelete.ts",
		"test.combined.ts",
		"trash.ts",
		"trash.mp4",
		"test/test.ts",
		"test/test.0.ts",
		"test/test.deleteme.ts",
		"test/test.deleteme.deleteme.ts",
		"test/test.mp4",
		"test/test.1.ts",
		"test/testnodelete.ts",
		"test/test.combined.ts",
		"test/trash.ts",
		"test/trash.mp4",
		"testb/testb.ts",
		"testb/testb.0.ts",
		"testb/testb.deleteme.ts",
		"testb/testb.deleteme.deleteme.ts",
		"testb/testb.mp4",
		"testb/testb.1.ts",
		"testb/testbnodelete.ts",
		"testb/testb.combined.ts",
		"testb/trash.ts",
		"testb/trash.mp4",
	}

	for _, file := range files {
		path := filepath.Join(dir, file)
		_ = os.MkdirAll(filepath.Dir(path), 0o0700)
		err = os.WriteFile(path, []byte("test"), 0o0700)
		if err != nil {
			panic(err)
		}
		err := os.Chtimes(path, time.Unix(0, 0), time.Unix(0, 0))
		if err != nil {
			panic(err)
		}
	}

	paths, err := cleaner.Scan(dir)
	require.NoError(t, err)
	requireSlicesEqual(t, []string{
		filepath.Join(dir, "test.ts"),
		filepath.Join(dir, "test.0.ts"),
		filepath.Join(dir, "test.deleteme.ts"),
		filepath.Join(dir, "test.deleteme.deleteme.ts"),
		filepath.Join(dir, "test.1.ts"),
		filepath.Join(dir, "test/test.ts"),
		filepath.Join(dir, "test/test.0.ts"),
		filepath.Join(dir, "test/test.deleteme.ts"),
		filepath.Join(dir, "test/test.deleteme.deleteme.ts"),
		filepath.Join(dir, "test/test.1.ts"),
		filepath.Join(dir, "testb/testb.ts"),
		filepath.Join(dir, "testb/testb.0.ts"),
		filepath.Join(dir, "testb/testb.deleteme.ts"),
		filepath.Join(dir, "testb/testb.deleteme.deleteme.ts"),
		filepath.Join(dir, "testb/testb.1.ts"),
	}, paths)
}

func requireSlicesEqual(t *testing.T, expected, actual []string) {
	// Check if the lengths are the same
	if len(expected) != len(actual) {
		fmt.Printf("expected: %v\n", expected)
		fmt.Printf("len expected: %v\n", len(expected))
		fmt.Printf("actual: %v\n", actual)
		fmt.Printf("len actual: %v\n", len(actual))
		t.FailNow()
		return
	}

	// Create a copy of the slices to avoid modifying the original slices
	copy1 := make([]string, len(expected))
	copy2 := make([]string, len(actual))
	copy(copy1, expected)
	copy(copy2, actual)

	// Sort the slices
	sort.Strings(copy1)
	sort.Strings(copy2)

	if !reflect.DeepEqual(copy1, copy2) {
		fmt.Printf("expected: %v\n", copy1)
		fmt.Printf("actual: %v\n", copy2)
		t.FailNow()
		return
	}
}
