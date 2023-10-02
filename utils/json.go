package utils

import (
	"bytes"
	"encoding/json"
)

func JSONMustEncode(v any) string {
	var bb bytes.Buffer
	if err := json.NewEncoder(&bb).Encode(v); err != nil {
		panic(err.Error())
	}
	return bb.String()
}
