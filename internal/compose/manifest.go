// Package compose (continued): the compose_externalize_manifest document
// type and its canonical byte form (BUILD-SPEC-v0.2 Track H2). M0, the
// single-document rule: every call produces exactly one manifest document —
// hash what you write; write what you return. Nothing is hashed that is not
// returned; nothing is written that is not hashed.
package compose

import (
	"fmt"
	"strings"

	"github.com/Rynaro/atomos/internal/jsonx"
)

// ManifestDefaultSummary lives in kernel_literals.go, a file that holds
// ONLY verbatim kernel-quoted string constants and is allowlisted (by
// name, structurally guarded — see that file's doc comment) against
// TestFenceNoForbiddenSurface's deny-list. This file (manifest.go) carries
// logic and stays under full fence strength — it is NOT allowlisted.

// ManifestEcmVersion is the kernel's hardcoded literal (M5) —
// context_externalize.sh:117,128 — read from no file. Reading it from a
// configuration file the kernel might one day change would be both a scope
// graze and a latent parity bug: the kernel would still stamp this literal
// regardless.
const ManifestEcmVersion = "0.1"

// Manifest is the compose_externalize_manifest document, field order fixed
// to context_externalize.sh:127-138's `jq -n` insertion order.
// FileFloorReason is nil by default (Q3/AC-H15: atomos never authors a
// reason it cannot observe) — when the caller supplies one it becomes the
// document's 11th and FINAL key, after CreatedAt (M6/AC-H14).
type Manifest struct {
	EcmVersion         string
	Summary            string
	Anchors            []string
	Symbols            []string
	Decisions          []string
	FailedApproaches   []string
	OpenVars           []string
	ContainsToolOrigin bool
	SessionID          string // "" => JSON null in the document (M4)
	CreatedAt          string
	FileFloorReason    *string
}

// Marshal renders m as the EXACT byte sequence the kernel's `jq -n` manifest
// build produces (context_externalize.sh:127-138), through the shared
// internal/jsonx ordered emitter — deliberately NOT json.MarshalIndent
// (which alphabetizes map keys and HTML-escapes '<','>','&'). This is the
// M0 hashed-and-written byte form: the caller (internal/compose's tool
// handler) hashes and (optionally) writes exactly these bytes, and decodes
// the response's "manifest" object FROM them rather than building it by a
// second, independent path (AC-H24).
func (m Manifest) Marshal() []byte {
	var b strings.Builder
	b.WriteString("{\n")
	jsonx.WriteStr(&b, 1, "ecm_version", m.EcmVersion, true)
	jsonx.WriteStr(&b, 1, "summary", m.Summary, true)
	jsonx.WriteArr(&b, 1, "anchors", m.Anchors, true)
	jsonx.WriteArr(&b, 1, "symbols", m.Symbols, true)
	jsonx.WriteArr(&b, 1, "decisions", m.Decisions, true)
	jsonx.WriteArr(&b, 1, "failed_approaches", m.FailedApproaches, true)
	jsonx.WriteArr(&b, 1, "open_vars", m.OpenVars, true)
	jsonx.WriteRaw(&b, 1, "contains_tool_origin", fmt.Sprintf("%t", m.ContainsToolOrigin), true)
	if m.SessionID == "" {
		jsonx.WriteRaw(&b, 1, "session_id", "null", true)
	} else {
		jsonx.WriteStr(&b, 1, "session_id", m.SessionID, true)
	}
	hasReason := m.FileFloorReason != nil
	jsonx.WriteStr(&b, 1, "created_at", m.CreatedAt, hasReason)
	if hasReason {
		jsonx.WriteStr(&b, 1, "file_floor_reason", *m.FileFloorReason, false)
	}
	b.WriteString("}\n")
	return []byte(b.String())
}
