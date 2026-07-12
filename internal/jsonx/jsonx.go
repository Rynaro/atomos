// Package jsonx provides the ORDERED, jq-exact JSON emission primitives
// shared by internal/ecl (the ECL envelope) and internal/compose (the
// externalize manifest) — the two hand-rolled emitters that must
// byte-reproduce the bash kernel's `jq -n` output: 2-space indent, insertion
// order, jq string escaping (quote, backslash, \b \f \n \r \t, \u00XX for
// other control chars and DEL; forward slash, non-ASCII UTF-8, and the
// unescaped '<','>','&' passed through verbatim — unlike Go's
// encoding/json, which HTML-escapes '<','>','&' and alphabetizes map keys
// under MarshalIndent).
//
// Two independent escapers would drift, and a drifted escaper is a silent
// byte-parity break — this package exists so there is exactly one.
//
// ANTI-SCOPE: this package only marshals bytes. No durable-storage call, no
// session-budget read, no rule-table evaluation, no host prompt-surface
// write, no wall clock.
package jsonx

import (
	"fmt"
	"strings"
)

// Indent returns level*2 spaces — jq -n's indent unit.
func Indent(level int) string { return strings.Repeat("  ", level) }

// WriteStr writes `"key": "value"[,]\n` at level, jq-escaping value.
func WriteStr(b *strings.Builder, level int, key, value string, comma bool) {
	WriteRaw(b, level, key, JSONString(value), comma)
}

// WriteRaw writes `"key": <rawValue>[,]\n` at level. rawValue is emitted
// verbatim — already-encoded JSON (a bool, a number, or the literal null) —
// never re-escaped.
func WriteRaw(b *strings.Builder, level int, key, rawValue string, comma bool) {
	b.WriteString(Indent(level))
	b.WriteString(JSONString(key))
	b.WriteString(": ")
	b.WriteString(rawValue)
	if comma {
		b.WriteString(",")
	}
	b.WriteString("\n")
}

// WriteObjOpen writes `"key": {\n` at level.
func WriteObjOpen(b *strings.Builder, level int, key string) {
	b.WriteString(Indent(level))
	b.WriteString(JSONString(key))
	b.WriteString(": {\n")
}

// WriteObjClose writes the matching closing `}[,]\n` at level.
func WriteObjClose(b *strings.Builder, level int, comma bool) {
	b.WriteString(Indent(level))
	b.WriteString("}")
	if comma {
		b.WriteString(",")
	}
	b.WriteString("\n")
}

// WriteArr writes a jq-exact JSON string array at level, confirmed
// empirically against a live `jq -n` capture:
//
//	"failed_approaches": [],                 <- empty: inline, no interior newline
//	"anchors": [
//	  "internal/compose/externalize.go:1"    <- populated: one element per line, indent level+1
//	],                                       <- closing bracket back at the key's own indent
//
// items must never be nil when the field is semantically "no entries" —
// pass an empty (non-nil) slice so it renders as the inline `[]` rather than
// being mistaken for an absent field by a caller inspecting the Go value.
func WriteArr(b *strings.Builder, level int, key string, items []string, comma bool) {
	b.WriteString(Indent(level))
	b.WriteString(JSONString(key))
	b.WriteString(": ")
	if len(items) == 0 {
		b.WriteString("[]")
	} else {
		b.WriteString("[\n")
		for i, it := range items {
			b.WriteString(Indent(level + 1))
			b.WriteString(JSONString(it))
			if i < len(items)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString(Indent(level))
		b.WriteString("]")
	}
	if comma {
		b.WriteString(",")
	}
	b.WriteString("\n")
}

// JSONString renders s as a jq-compatible JSON string literal: quote and
// backslash escaped, control chars (<0x20) and DEL (0x7f) escaped as \u00XX
// (with the standard \b \f \n \r \t shortcuts), everything else — including
// non-ASCII UTF-8 and the unescaped '/','<','>','&' — passed through
// verbatim. This mirrors jq's own string encoder, NOT Go's encoding/json
// (which HTML-escapes '<','>','&' by default).
func JSONString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
