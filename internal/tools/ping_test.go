package tools_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/beduardo/eve-realm-mcp/internal/tools"
)

// TestPingTool_Descriptor_Name verifies that the tool name is "ping".
func TestPingTool_Descriptor_Name(t *testing.T) {
	tool := tools.NewTool()
	if tool.Name != "ping" {
		t.Errorf("expected tool name %q, got %q", "ping", tool.Name)
	}
}

// TestPingTool_Descriptor_Description verifies the tool description text.
func TestPingTool_Descriptor_Description(t *testing.T) {
	tool := tools.NewTool()
	want := "Diagnostic tool that returns pong with server timestamp"
	if tool.Description != want {
		t.Errorf("expected description %q, got %q", want, tool.Description)
	}
}

// TestPingTool_Descriptor_InputSchema_NoRequiredParams verifies the input schema
// is an empty JSON object with no required parameters.
func TestPingTool_Descriptor_InputSchema_NoRequiredParams(t *testing.T) {
	tool := tools.NewTool()
	if tool.InputSchema != "{}" {
		t.Errorf("expected InputSchema %q, got %q", "{}", tool.InputSchema)
	}
}

// TestPingHandler_Invoke_OutputIsValidJSON verifies the handler returns valid JSON.
func TestPingHandler_Invoke_OutputIsValidJSON(t *testing.T) {
	tool := tools.NewTool()
	output, err := tool.Handler("")
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		t.Errorf("handler output is not valid JSON: %v", err)
	}
}

// TestPingHandler_Invoke_ReturnsPongMessage verifies that the handler output JSON
// contains a "message" field equal to "pong".
func TestPingHandler_Invoke_ReturnsPongMessage(t *testing.T) {
	tool := tools.NewTool()
	output, err := tool.Handler("")
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if result["message"] != "pong" {
		t.Errorf("expected message %q, got %q", "pong", result["message"])
	}
}

// TestPingHandler_Invoke_MessageIsPong is an alias assertion for the "pong" message value.
func TestPingHandler_Invoke_MessageIsPong(t *testing.T) {
	tool := tools.NewTool()
	output, err := tool.Handler("")
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if result["message"] != "pong" {
		t.Errorf("expected message field %q, got %q", "pong", result["message"])
	}
}

// TestPingHandler_Invoke_TimestampIsRFC3339 verifies the handler output contains
// a "timestamp" field that is a valid RFC 3339 string.
func TestPingHandler_Invoke_TimestampIsRFC3339(t *testing.T) {
	tool := tools.NewTool()
	output, err := tool.Handler("")
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	ts, ok := result["timestamp"]
	if !ok {
		t.Fatal("output JSON missing required field: timestamp")
	}
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Errorf("timestamp %q is not valid RFC 3339: %v", ts, err)
	}
}

// TestPingHandler_Invoke_TimestampParsesAsRFC3339 verifies time.Parse succeeds
// with time.RFC3339 layout on the returned timestamp field.
func TestPingHandler_Invoke_TimestampParsesAsRFC3339(t *testing.T) {
	tool := tools.NewTool()
	output, err := tool.Handler("")
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, result["timestamp"]); err != nil {
		t.Errorf("time.Parse(time.RFC3339, %q) failed: %v", result["timestamp"], err)
	}
}

// TestPingHandler_Invoke_TimestampWithinWindow verifies that the timestamp in the
// response falls within a [T1-1s, T2+1s] window recorded around the invocation.
func TestPingHandler_Invoke_TimestampWithinWindow(t *testing.T) {
	tool := tools.NewTool()
	tolerance := time.Second

	t1 := time.Now().Add(-tolerance)
	output, err := tool.Handler("")
	t2 := time.Now().Add(tolerance)

	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	parsed, err := time.Parse(time.RFC3339, result["timestamp"])
	if err != nil {
		t.Fatalf("timestamp %q is not valid RFC 3339: %v", result["timestamp"], err)
	}

	if parsed.Before(t1) || parsed.After(t2) {
		t.Errorf("timestamp %v is outside expected window [%v, %v]", parsed, t1, t2)
	}
}

// TestPingHandler_Invoke_OutputHasExactlyTwoFields verifies the output JSON object
// contains exactly two fields: "message" and "timestamp".
func TestPingHandler_Invoke_OutputHasExactlyTwoFields(t *testing.T) {
	tool := tools.NewTool()
	output, err := tool.Handler("")
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if len(raw) != 2 {
		t.Errorf("expected exactly 2 fields in output JSON, got %d: %v", len(raw), raw)
	}
	if _, ok := raw["message"]; !ok {
		t.Error("output JSON missing field: message")
	}
	if _, ok := raw["timestamp"]; !ok {
		t.Error("output JSON missing field: timestamp")
	}
}
