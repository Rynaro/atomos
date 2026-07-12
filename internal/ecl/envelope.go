// Package ecl composes the ECM handoff-brief ECL envelope, using the shared
// internal/jsonx ORDERED emitter (2-space indent, insertion order
// (context_handoff.sh:219-233), jq string escaping, single trailing
// newline) to reproduce the bash kernel's `jq -n` byte output exactly. The
// ordered-emitter primitives themselves (the jq-exact string escaper and the
// object/array writers) moved to internal/jsonx in v0.2 so the externalize
// manifest composer (internal/compose) can share one escaper instead of
// growing a second, driftable one — this package now calls jsonx rather than
// hand-rolling its own copy.
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

	"github.com/Rynaro/atomos/internal/jsonx"
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
	jsonx.WriteStr(&b, 1, "envelope_version", e.EnvelopeVersion, true)
	jsonx.WriteStr(&b, 1, "message_id", e.MessageID, true)
	jsonx.WriteStr(&b, 1, "thread_id", e.ThreadID, true)
	jsonx.WriteRaw(&b, 1, "parent_id", "null", true)

	jsonx.WriteObjOpen(&b, 1, "from")
	jsonx.WriteStr(&b, 2, "eidolon", e.From.Eidolon, true)
	jsonx.WriteStr(&b, 2, "version", e.From.Version, false)
	jsonx.WriteObjClose(&b, 1, true)

	jsonx.WriteObjOpen(&b, 1, "to")
	jsonx.WriteStr(&b, 2, "eidolon", e.To.Eidolon, true)
	jsonx.WriteStr(&b, 2, "version", e.To.Version, false)
	jsonx.WriteObjClose(&b, 1, true)

	jsonx.WriteStr(&b, 1, "objective", e.Objective, true)
	jsonx.WriteStr(&b, 1, "performative", e.Performative, true)

	jsonx.WriteObjOpen(&b, 1, "artifact")
	jsonx.WriteStr(&b, 2, "kind", e.Artifact.Kind, true)
	jsonx.WriteStr(&b, 2, "schema_version", e.Artifact.SchemaVersion, true)
	jsonx.WriteStr(&b, 2, "path", e.Artifact.Path, true)
	jsonx.WriteStr(&b, 2, "sha256", e.Artifact.SHA256, true)
	jsonx.WriteRaw(&b, 2, "size_bytes", fmt.Sprintf("%d", e.Artifact.SizeBytes), false)
	jsonx.WriteObjClose(&b, 1, true)

	jsonx.WriteObjOpen(&b, 1, "integrity")
	jsonx.WriteStr(&b, 2, "method", e.Integrity.Method, true)
	jsonx.WriteStr(&b, 2, "value", e.Integrity.Value, false)
	jsonx.WriteObjClose(&b, 1, true)

	jsonx.WriteObjOpen(&b, 1, "trace")
	jsonx.WriteStr(&b, 2, "ts", e.Trace.TS, true)
	jsonx.WriteStr(&b, 2, "host", e.Trace.Host, true)
	jsonx.WriteStr(&b, 2, "model", e.Trace.Model, true)
	jsonx.WriteStr(&b, 2, "tier", e.Trace.Tier, false)
	jsonx.WriteObjClose(&b, 1, true)

	jsonx.WriteStr(&b, 1, "topic_key", e.TopicKey, true)
	jsonx.WriteRaw(&b, 1, "contains_tool_origin", fmt.Sprintf("%t", e.ContainsToolOrigin), false)
	b.WriteString("}\n")
	return []byte(b.String())
}
