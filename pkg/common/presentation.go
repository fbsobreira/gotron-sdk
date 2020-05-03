package common

import (
	"bytes"
	"encoding/json"
)

func JSONPrettyFormat(in string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(in), "", "  ")
	if err != nil {
		return in
	}
	return out.String()
}

// returns "{}" on failure case
func ToJSONUnsafe(payload interface{}, pretty bool) string {
	j, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	if pretty {
		return JSONPrettyFormat(string(j))
	}
	return string(j)
}
