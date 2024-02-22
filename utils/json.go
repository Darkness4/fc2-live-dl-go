package utils

import (
	"bytes"
	"encoding/json"
)

// MustJSONEncode encodes the value to JSON or panic.
func MustJSONEncode(v any) string {
	var bb bytes.Buffer
	if err := json.NewEncoder(&bb).Encode(v); err != nil {
		panic(err.Error())
	}
	return bb.String()
}
