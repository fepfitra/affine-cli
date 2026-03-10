package validate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var (
	// UUID pattern: 8-4-4-4-12 hex chars
	uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// AFFiNE short doc ID: alphanumeric, 6-20 chars
	shortIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{2,64}$`)
)

// WorkspaceID validates a workspace ID (must be UUID).
func WorkspaceID(id string) error {
	if id == "" {
		return fmt.Errorf("workspace ID is required")
	}
	if !uuidPattern.MatchString(id) {
		return fmt.Errorf("invalid workspace ID %q: expected UUID format", id)
	}
	return nil
}

// DocID validates a document ID (UUID or short alphanumeric).
func DocID(id string) error {
	if id == "" {
		return fmt.Errorf("document ID is required")
	}
	if uuidPattern.MatchString(id) {
		return nil
	}
	if shortIDPattern.MatchString(id) {
		return nil
	}
	return fmt.Errorf("invalid document ID %q: expected UUID or alphanumeric ID", id)
}

// NotEmpty validates that a string field is not empty.
func NotEmpty(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// NoControlChars checks for control characters in user input.
func NoControlChars(field, value string) error {
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return fmt.Errorf("%s contains invalid control character (U+%04X)", field, r)
		}
	}
	return nil
}

// SafeString validates a string is non-empty and has no control chars.
func SafeString(field, value string) error {
	if err := NotEmpty(field, value); err != nil {
		return err
	}
	return NoControlChars(field, value)
}
