// Package ecl composes the ECM handoff-brief ECL envelope with a hand-rolled
// ORDERED emitter that reproduces the bash kernel's `jq -n` byte output
// exactly: 2-space indent, insertion order (context_handoff.sh:219-233), jq
// string escaping (quote/backslash/control-chars escaped; forward slash and
// non-ASCII UTF-8 left verbatim — jq does NOT HTML-escape '<','>','&' the way
// Go's default encoding/json does), and a single trailing newline.
//
// This package is a faithful EMITTER of the shape the kernel already writes.
// It does not define, validate against, or extend the ECL performative set or
// envelope schema — that authority stays with Rynaro/eidolons-ecl (ADR §5.3).
//
// ANTI-SCOPE: no durable-storage call, no session-budget read, no rule-table
// evaluation, no host prompt-surface write. This package only marshals bytes.
package ecl

import (
	"fmt"
	"strings"
)

// Parity-critical constants (verbatim, never derived from ATOMOS_VERSION or
// the actual host — see BUILD-SPEC.md "parity traps" 1-3).
const (
	// EnvelopeVersion is the kernel's emitted stamp (context_handoff.sh:220).
	// The verifier side expects "2.0" and only WARNS otherwise (parity trap
	// 2) — atomos's composer reproduces "1.0" verbatim; it is NEVER "fixed".
	EnvelopeVersion = "1.0"

	FromEidolon = "eidolons-context-kernel"
	ToEidolon   = "session_successor"
	ToVersion   = "n/a"

	Performative          = "INFORM"
	ArtifactKind          = "ecm/handoff-brief@0.1"
	ArtifactSchemaVersion = "0.1"
	IntegrityMethod       = "sha256"

	// TraceHost/TraceModel/TraceTier are hardcoded by the kernel regardless
	// of the actual host (context_handoff.sh:230, parity trap 3).
	TraceHost  = "claude-code"
	TraceModel = "n/a"
	TraceTier  = "standard"

	TopicKey = "session_handoff"

	// ObjectivePrefix precedes the task_state head in the objective field
	// (context_handoff.sh:206-207,226).
	ObjectivePrefix = "Session handoff brief for context-lifecycle succession (ECM P1): "
)

// Party is an envelope sender/receiver identity.
type Party struct {
	Eidolon string
	Version string
}

// Artifact describes the composed brief file.
type Artifact struct {
	Kind          string
	SchemaVersion string
	Path          string
	SHA256        string
	SizeBytes     int64
}

// Integrity carries the SHA-256 tag (identical value to Artifact.SHA256; the
// kernel emits it twice — once under artifact, once under integrity).
type Integrity struct {
	Method string
	Value  string
}

// Trace carries the wall-clock ts plus the hardcoded host/model/tier triple.
type Trace struct {
	TS    string
	Host  string
	Model string
	Tier  string
}

// Envelope is the ECM handoff-brief ECL envelope, field order fixed to match
// context_handoff.sh:219-233's jq -n insertion order exactly.
type Envelope struct {
	EnvelopeVersion    string
	MessageID          string
	ThreadID           string
	From               Party
	To                 Party
	Objective          string
	Performative       string
	Artifact           Artifact
	Integrity          Integrity
	Trace              Trace
	TopicKey           string
	ContainsToolOrigin bool
}

// New builds an Envelope for a composed handoff brief. fromVersion is the
// CALLER's from_version input, echoed verbatim (parity trap 1) — never
// ATOMOS_VERSION. taskStateHead is the first non-blank line of task_state
// (post-default), used to build the objective.
func New(messageID, threadID, fromVersion, artifactPath, sha string, sizeBytes int64, ts, taskStateHead string, containsToolOrigin bool) Envelope {
	return Envelope{
		EnvelopeVersion: EnvelopeVersion,
		MessageID:       messageID,
		ThreadID:        threadID,
		From:            Party{Eidolon: FromEidolon, Version: fromVersion},
		To:              Party{Eidolon: ToEidolon, Version: ToVersion},
		Objective:       ObjectivePrefix + taskStateHead,
		Performative:    Performative,
		Artifact: Artifact{
			Kind:          ArtifactKind,
			SchemaVersion: ArtifactSchemaVersion,
			Path:          artifactPath,
			SHA256:        sha,
			SizeBytes:     sizeBytes,
		},
		Integrity:          Integrity{Method: IntegrityMethod, Value: sha},
		Trace:              Trace{TS: ts, Host: TraceHost, Model: TraceModel, Tier: TraceTier},
		TopicKey:           TopicKey,
		ContainsToolOrigin: containsToolOrigin,
	}
}

// Marshal renders e as the EXACT byte sequence `jq -n` produces for the
// kernel's envelope construction. Deliberately NOT json.MarshalIndent (which
// sorts map keys and HTML-escapes '<','>','&') — an ordered, hand-written
// emitter is the only way to reproduce jq's insertion-order + escaping
// contract (verified empirically against a live jq -n capture; see
// fixtures/README.md).
func (e Envelope) Marshal() []byte {
	var b strings.Builder
	b.WriteString("{\n")
	writeStr(&b, 1, "envelope_version", e.EnvelopeVersion, true)
	writeStr(&b, 1, "message_id", e.MessageID, true)
	writeStr(&b, 1, "thread_id", e.ThreadID, true)
	writeRaw(&b, 1, "parent_id", "null", true)

	writeObjOpen(&b, 1, "from")
	writeStr(&b, 2, "eidolon", e.From.Eidolon, true)
	writeStr(&b, 2, "version", e.From.Version, false)
	writeObjClose(&b, 1, true)

	writeObjOpen(&b, 1, "to")
	writeStr(&b, 2, "eidolon", e.To.Eidolon, true)
	writeStr(&b, 2, "version", e.To.Version, false)
	writeObjClose(&b, 1, true)

	writeStr(&b, 1, "objective", e.Objective, true)
	writeStr(&b, 1, "performative", e.Performative, true)

	writeObjOpen(&b, 1, "artifact")
	writeStr(&b, 2, "kind", e.Artifact.Kind, true)
	writeStr(&b, 2, "schema_version", e.Artifact.SchemaVersion, true)
	writeStr(&b, 2, "path", e.Artifact.Path, true)
	writeStr(&b, 2, "sha256", e.Artifact.SHA256, true)
	writeRaw(&b, 2, "size_bytes", fmt.Sprintf("%d", e.Artifact.SizeBytes), false)
	writeObjClose(&b, 1, true)

	writeObjOpen(&b, 1, "integrity")
	writeStr(&b, 2, "method", e.Integrity.Method, true)
	writeStr(&b, 2, "value", e.Integrity.Value, false)
	writeObjClose(&b, 1, true)

	writeObjOpen(&b, 1, "trace")
	writeStr(&b, 2, "ts", e.Trace.TS, true)
	writeStr(&b, 2, "host", e.Trace.Host, true)
	writeStr(&b, 2, "model", e.Trace.Model, true)
	writeStr(&b, 2, "tier", e.Trace.Tier, false)
	writeObjClose(&b, 1, true)

	writeStr(&b, 1, "topic_key", e.TopicKey, true)
	writeRaw(&b, 1, "contains_tool_origin", fmt.Sprintf("%t", e.ContainsToolOrigin), false)
	b.WriteString("}\n")
	return []byte(b.String())
}

func indent(level int) string { return strings.Repeat("  ", level) }

func writeStr(b *strings.Builder, level int, key, value string, comma bool) {
	writeRaw(b, level, key, jsonString(value), comma)
}

func writeRaw(b *strings.Builder, level int, key, rawValue string, comma bool) {
	b.WriteString(indent(level))
	b.WriteString(jsonString(key))
	b.WriteString(": ")
	b.WriteString(rawValue)
	if comma {
		b.WriteString(",")
	}
	b.WriteString("\n")
}

func writeObjOpen(b *strings.Builder, level int, key string) {
	b.WriteString(indent(level))
	b.WriteString(jsonString(key))
	b.WriteString(": {\n")
}

func writeObjClose(b *strings.Builder, level int, comma bool) {
	b.WriteString(indent(level))
	b.WriteString("}")
	if comma {
		b.WriteString(",")
	}
	b.WriteString("\n")
}

// jsonString renders s as a jq-compatible JSON string literal: quote and
// backslash escaped, control chars (<0x20) and DEL (0x7f) escaped as \u00XX
// (with the standard \b \f \n \r \t shortcuts), everything else — including
// non-ASCII UTF-8 and the unescaped '/','<','>','&' — passed through
// verbatim. This mirrors jq's own string encoder, NOT Go's encoding/json
// (which HTML-escapes '<','>','&' by default).
func jsonString(s string) string {
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
