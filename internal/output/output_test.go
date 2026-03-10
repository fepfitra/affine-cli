package output

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestJSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	JSON(map[string]string{"key": "value"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if result["key"] != "value" {
		t.Errorf("key = %q, want 'value'", result["key"])
	}
}

func TestRawJSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RawJSON(json.RawMessage(`{"foo":"bar"}`))

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if result["foo"] != "bar" {
		t.Errorf("foo = %q, want 'bar'", result["foo"])
	}
}

func TestRawJSONInvalid(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	RawJSON(json.RawMessage(`not json`))

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if buf.Len() == 0 {
		t.Error("expected output for invalid JSON")
	}
}

func TestError(t *testing.T) {
	err := Error("test %s %d", "error", 42)
	if err == nil {
		t.Fatal("Error() returned nil")
	}
	if err.Error() != "test error 42" {
		t.Errorf("error = %q, want 'test error 42'", err.Error())
	}
}

func TestErrorWithCode(t *testing.T) {
	err := ErrorWithCode("INVALID_INPUT", "bad value: %s", "xyz")
	if err == nil {
		t.Fatal("ErrorWithCode() returned nil")
	}
	if err.Error() != "bad value: xyz" {
		t.Errorf("error = %q, want 'bad value: xyz'", err.Error())
	}
}

func TestFilteredJSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := json.RawMessage(`{"workspace":{"id":"abc","public":true,"enableAi":false}}`)
	FilteredJSON(data, []string{"id", "public"})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if result["id"] != "abc" {
		t.Errorf("id = %v, want 'abc'", result["id"])
	}
	if result["public"] != true {
		t.Errorf("public = %v, want true", result["public"])
	}
	if _, ok := result["enableAi"]; ok {
		t.Error("enableAi should be filtered out")
	}
}

func TestFilteredJSONEmpty(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := json.RawMessage(`{"id":"abc","name":"test"}`)
	FilteredJSON(data, nil) // no filter = passthrough

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if result["id"] != "abc" {
		t.Errorf("id = %v, want 'abc'", result["id"])
	}
}

func TestDryRun(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	DryRun("delete workspace", map[string]any{
		"workspace_id": "abc-123",
	})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)

	// In non-TTY (test), DryRun outputs JSON
	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("DryRun output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if result["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", result["dry_run"])
	}
	if result["action"] != "delete workspace" {
		t.Errorf("action = %v, want 'delete workspace'", result["action"])
	}
}
