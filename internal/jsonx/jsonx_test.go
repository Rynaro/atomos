package jsonx

import (
	"strings"
	"testing"
)

// TestJSONStringEscaping moved here verbatim from internal/ecl/envelope_test.go
// (v0.2 H1 extraction) — the escaper itself did not change, only its package.
func TestJSONStringEscaping(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`a/b`, `"a/b"`},
		{`a<b>c&d`, `"a<b>c&d"`},
		{"a\"b\\c", `"a\"b\\c"`},
		{"line1\nline2", `"line1\nline2"`},
		{"tab\ttab", `"tab\ttab"`},
		{"café", "\"café\""},
	}
	for _, c := range cases {
		if got := JSONString(c.in); got != c.want {
			t.Errorf("JSONString(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

// TestWriteArrEmptyInline pins the empty-array shape confirmed empirically
// against a live jq -n capture: inline `[]`, no interior newline.
func TestWriteArrEmptyInline(t *testing.T) {
	var b strings.Builder
	WriteArr(&b, 1, "anchors", []string{}, true)
	want := "  \"anchors\": [],\n"
	if got := b.String(); got != want {
		t.Errorf("WriteArr(empty) = %q, want %q", got, want)
	}
}

// TestWriteArrPopulatedShape pins the populated-array shape: one element per
// line at level+1, closing bracket back at the key's own indent.
func TestWriteArrPopulatedShape(t *testing.T) {
	var b strings.Builder
	WriteArr(&b, 1, "anchors", []string{"a:1", "b:2"}, true)
	want := "  \"anchors\": [\n    \"a:1\",\n    \"b:2\"\n  ],\n"
	if got := b.String(); got != want {
		t.Errorf("WriteArr(populated) = %q, want %q", got, want)
	}
}

// TestWriteArrNoTrailingComma confirms the comma=false path (last key in an
// object) omits the trailing comma.
func TestWriteArrNoTrailingComma(t *testing.T) {
	var b strings.Builder
	WriteArr(&b, 1, "open_vars", []string{"x"}, false)
	want := "  \"open_vars\": [\n    \"x\"\n  ]\n"
	if got := b.String(); got != want {
		t.Errorf("WriteArr(no comma) = %q, want %q", got, want)
	}
}
