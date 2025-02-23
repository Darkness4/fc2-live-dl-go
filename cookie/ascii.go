package cookie

import (
	"strings"
	"unicode"
)

// Is returns whether s is ASCII.
func asciiIs(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// asciiIsPrint returns whether s is ASCII and printable according to
// https://tools.ietf.org/html/rfc20#section-4.2.
func asciiIsPrint(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > '~' {
			return false
		}
	}
	return true
}

// ToLower returns the lowercase version of s if s is ASCII and printable.
func asciiToLower(s string) (lower string, ok bool) {
	if !asciiIsPrint(s) {
		return "", false
	}
	return strings.ToLower(s), true
}
