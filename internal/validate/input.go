package validate

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/glebmish/ynab-cli/internal/cliexit"
)

func vErr(format string, args ...any) error {
	return &cliexit.ValidationError{Err: fmt.Errorf(format, args...)}
}

// PathParam validates that a user-supplied value is safe to embed in a URL path.
func PathParam(name, value string) error {
	if value == "" {
		return vErr("field %q: must not be empty", name)
	}
	if strings.Contains(value, "..") {
		return vErr("field %q: contains path traversal characters", name)
	}
	if strings.ContainsAny(value, "?#&") {
		return vErr("field %q: contains query injection characters", name)
	}
	if strings.Contains(value, "%") {
		return vErr("field %q: contains percent-encoded characters (provide raw values)", name)
	}
	for _, r := range value {
		if r < 0x20 {
			return vErr("field %q: contains control characters", name)
		}
	}
	return nil
}

// IntParam validates that a string parses as a Go int (allows negatives).
func IntParam(name, value string) error {
	if value == "" {
		return vErr("field %q: must not be empty", name)
	}
	if _, err := strconv.Atoi(value); err != nil {
		return vErr("field %q: expected integer, got %q", name, value)
	}
	return nil
}

// DateParam accepts ISO YYYY-MM-DD only.
func DateParam(name, value string) error {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return vErr("field %q: invalid date %q (expected YYYY-MM-DD)", name, value)
	}
	return nil
}

// JSONBody ensures the body is syntactically valid JSON with no rogue control chars.
func JSONBody(body string) error {
	if body == "" {
		return vErr("empty JSON body")
	}
	for i, r := range body {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return vErr("JSON body contains control character at position %d", i)
		}
	}
	if !json.Valid([]byte(body)) {
		return vErr("invalid JSON body")
	}
	return nil
}
