package fc2

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func PrepareFile(
	outFormat string,
	meta *GetMetaData,
	labels map[string]string,
	ext string,
) (fName string, err error) {
	n := 0
	// Find unique name
	for {
		var extn string
		if n == 0 {
			extn = ext
		} else {
			extn = fmt.Sprintf("%d.%s", n, ext)
		}
		fName, err = FormatOutput(outFormat, meta, labels, extn)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(fName); errors.Is(err, os.ErrNotExist) {
			break
		}
		n++
	}

	// Mkdir parents dirs
	if err := os.MkdirAll(filepath.Dir(fName), 0o755); err != nil {
		panic(err)
	}
	return fName, nil
}
