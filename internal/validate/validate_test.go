package validate

import (
	"testing"
)

func TestWorkspaceID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"50a4cd2e-250c-4f22-a36d-af6fbebaa8e2", false},
		{"00000000-0000-0000-0000-000000000000", false},
		{"", true},
		{"not-a-uuid", true},
		{"50a4cd2e250c4f22a36daf6fbebaa8e2", true}, // no dashes
		{"50a4cd2e-250c-4f22-a36d-af6fbebaa8e", true}, // too short
	}
	for _, tt := range tests {
		err := WorkspaceID(tt.id)
		if (err != nil) != tt.wantErr {
			t.Errorf("WorkspaceID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
		}
	}
}

func TestDocID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"50a4cd2e-250c-4f22-a36d-af6fbebaa8e2", false}, // UUID
		{"yco8IHar80", false},                             // short ID
		{"abc-123_XY", false},                             // alphanumeric with dash/underscore
		{"", true},
		{"has space", true},
		{"has/slash", true},
	}
	for _, tt := range tests {
		err := DocID(tt.id)
		if (err != nil) != tt.wantErr {
			t.Errorf("DocID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
		}
	}
}

func TestNotEmpty(t *testing.T) {
	if err := NotEmpty("field", "value"); err != nil {
		t.Errorf("NotEmpty with value: %v", err)
	}
	if err := NotEmpty("field", ""); err == nil {
		t.Error("NotEmpty with empty string should error")
	}
	if err := NotEmpty("field", "   "); err == nil {
		t.Error("NotEmpty with whitespace should error")
	}
}

func TestNoControlChars(t *testing.T) {
	if err := NoControlChars("f", "normal text\nwith newlines\tand tabs"); err != nil {
		t.Errorf("should allow newlines/tabs: %v", err)
	}
	if err := NoControlChars("f", "has\x00null"); err == nil {
		t.Error("should reject null byte")
	}
	if err := NoControlChars("f", "has\x07bell"); err == nil {
		t.Error("should reject bell character")
	}
}

func TestSafeString(t *testing.T) {
	if err := SafeString("f", "hello"); err != nil {
		t.Errorf("SafeString with valid input: %v", err)
	}
	if err := SafeString("f", ""); err == nil {
		t.Error("SafeString with empty should error")
	}
	if err := SafeString("f", "has\x00null"); err == nil {
		t.Error("SafeString with control char should error")
	}
}
