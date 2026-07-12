package ecl

import (
	"encoding/json"
	"testing"
)

// AC-B03: envelope_version is exactly "1.0" — the kernel drift mirrored,
// never "fixed" to "2.0" (context_handoff.sh:220; parity trap 2).
func TestEnvelopeVersionIsOneDotZero(t *testing.T) {
	e := New("msg-context-handoff-20260707T000000Z", "thread-1", "1.2.3", "handoff-20260707T000000Z.md", "deadbeef", 42, "2026-07-07T00:00:00Z", "task head", false)
	if e.EnvelopeVersion != "1.0" {
		t.Fatalf("EnvelopeVersion = %q, want %q", e.EnvelopeVersion, "1.0")
	}

	var decoded map[string]any
	if err := json.Unmarshal(e.Marshal(), &decoded); err != nil {
		t.Fatalf("unmarshal marshaled envelope: %v", err)
	}
	if decoded["envelope_version"] != "1.0" {
		t.Fatalf("marshaled envelope_version = %v, want \"1.0\"", decoded["envelope_version"])
	}
}

func TestMarshalRoundTripsAllFields(t *testing.T) {
	e := New("msg-1", "thread-1", "n/a", "handoff-1.md", "abc123", 7, "2026-01-01T00:00:00Z", "head line", true)
	var decoded map[string]any
	if err := json.Unmarshal(e.Marshal(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["message_id"] != "msg-1" {
		t.Errorf("message_id = %v", decoded["message_id"])
	}
	if decoded["parent_id"] != nil {
		t.Errorf("parent_id = %v, want nil (JSON null)", decoded["parent_id"])
	}
	from, _ := decoded["from"].(map[string]any)
	if from["eidolon"] != FromEidolon || from["version"] != "n/a" {
		t.Errorf("from = %v", from)
	}
	artifact, _ := decoded["artifact"].(map[string]any)
	if artifact["size_bytes"] != float64(7) {
		t.Errorf("artifact.size_bytes = %v", artifact["size_bytes"])
	}
	if decoded["contains_tool_origin"] != true {
		t.Errorf("contains_tool_origin = %v", decoded["contains_tool_origin"])
	}
}

// String-escaping coverage moved to internal/jsonx/jsonx_test.go
// (TestJSONStringEscaping) alongside the primitives themselves (v0.2 H1
// extraction) — this package now calls jsonx.JSONString rather than owning
// its own copy. TestMarshalRoundTripsAllFields above still exercises the
// escaper indirectly through Envelope.Marshal.
