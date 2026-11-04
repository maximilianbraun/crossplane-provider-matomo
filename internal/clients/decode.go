package clients

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// DecodeID extracts an integer ID from a Matomo response. Matomo returns IDs
// inconsistently across methods: a bare number (3), a quoted number ("3"), or
// wrapped in a value envelope ({"value":3} / {"value":"3"}).
func DecodeID(raw json.RawMessage) (int, error) {
	trimmed := []byte(strings.TrimSpace(string(raw)))

	var n json.Number
	if err := json.Unmarshal(trimmed, &n); err == nil {
		if i, err := n.Int64(); err == nil {
			return int(i), nil
		}
	}
	var s string
	if err := json.Unmarshal(trimmed, &s); err == nil {
		if i, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return i, nil
		}
	}
	var wrap struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(trimmed, &wrap); err == nil && len(wrap.Value) > 0 {
		return DecodeID(wrap.Value)
	}
	return 0, fmt.Errorf("cannot decode id from %q", string(raw))
}

// DecodeString extracts a string from a Matomo response, unwrapping a
// {"value":"..."} envelope if present.
func DecodeString(raw json.RawMessage) (string, error) {
	trimmed := []byte(strings.TrimSpace(string(raw)))

	var s string
	if err := json.Unmarshal(trimmed, &s); err == nil {
		return s, nil
	}
	var wrap struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(trimmed, &wrap); err == nil && len(wrap.Value) > 0 {
		return DecodeString(wrap.Value)
	}
	return "", fmt.Errorf("cannot decode string from %q", string(raw))
}

// IsNotFound reports whether a Matomo API error denotes a missing resource,
// so Observe can translate it into "resource does not exist" rather than a
// hard error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, needle := range []string{
		"could not be found",
		"couldn't be found",
		"not found",
		"doesn't exist",
		"does not exist",
	} {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}
