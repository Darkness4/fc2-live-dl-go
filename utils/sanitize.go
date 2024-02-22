package utils

import (
	"regexp"
	"strings"
	"unicode"
)

// SanitizeFilename sanitizes the filename.
func SanitizeFilename(filename string) string {
	// Replace forbidden characters with underscores
	re := regexp.MustCompile(`[\\/:*?\"<>|]+`)
	filename = re.ReplaceAllString(filename, "_")

	// Define the list of reserved Windows filenames
	var reservedNames = []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}

	// Remove ASCII control characters
	filename = strings.Map(func(r rune) rune {
		if unicode.IsGraphic(r) {
			return r
		}
		return -1
	}, filename)

	// Remove leading and trailing whitespace
	filename = strings.TrimSpace(filename)

	// Remove leading and trailing dots
	filename = strings.Trim(filename, ".")

	// Remove reserved Windows filenames
	for _, name := range reservedNames {
		if strings.EqualFold(filename, name) {
			filename = "_" + filename
			break
		}
	}

	// Ensure the filename is not empty
	if len(filename) == 0 {
		filename = "_"
	}

	return filename
}
