// Package compose implements compose_handoff: a pure function that mirrors
// `eidolons context handoff` (cli/src/context_handoff.sh) byte-for-byte for
// the brief body, and via the internal/ecl ordered emitter for the envelope.
//
// PURE HANDLER (AC-F04): this package reads no wall clock of its own.
// ts/iso_ts are caller inputs; defaulting an omitted pair happens at the
// single server-layer seam (internal/server), never here.
package compose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rynaro/atomos/internal/ecl"
	"github.com/Rynaro/atomos/internal/hashing"
)

// DefaultTaskState mirrors context_handoff.sh:105.
const DefaultTaskState = "(no task-state summary provided)"

// DefaultOutDir mirrors ECM_SIDECAR_DIR (lib_context.sh).
const DefaultOutDir = ".eidolons/.context"

// AdvisoryTokenTarget is the hardcoded 1500-token advisory target
// (context_handoff.sh:173; atomos does NOT read the nexus's context
// configuration file — Rejected Alternative #6). WARN-only, never truncates.
const AdvisoryTokenTarget = 1500

// BytesPerToken mirrors ECM_BYTES_PER_TOKEN (lib_context.sh), used for the
// floor-divide token estimate (lib_context.sh:117-120).
const BytesPerToken = 4

// HandoffInput is the compose_handoff MCP tool input (ADR §2.1).
type HandoffInput struct {
	TaskState          string   `json:"task_state,omitempty"`
	Narrative          string   `json:"narrative,omitempty"`
	Anchors            []string `json:"anchors,omitempty"`
	Symbols            []string `json:"symbols,omitempty"`
	Decisions          []string `json:"decisions,omitempty"`
	FailedApproaches   []string `json:"failed_approaches,omitempty"`
	OpenVars           []string `json:"open_vars,omitempty"`
	NextSteps          []string `json:"next_steps,omitempty"`
	ThreadID           string   `json:"thread_id,omitempty"`
	SessionID          string   `json:"session_id,omitempty"`
	FromVersion        string   `json:"from_version,omitempty"`
	ContainsToolOrigin bool     `json:"contains_tool_origin,omitempty"`
	TS                 string   `json:"ts,omitempty"`
	ISOTS              string   `json:"iso_ts,omitempty"`
	WriteSidecar       *bool    `json:"write_sidecar,omitempty"`
	OutDir             string   `json:"out_dir,omitempty"`
}

// HandoffResult is the compose_handoff MCP tool output (ADR §2.1).
type HandoffResult struct {
	BriefMD      string         `json:"brief_md"`
	BriefSHA256  string         `json:"brief_sha256"`
	Envelope     map[string]any `json:"envelope"`
	BriefPath    *string        `json:"brief_path"`
	EnvelopePath *string        `json:"envelope_path"`
	TokensEst    int            `json:"tokens_est"`
	Oversize     bool           `json:"oversize"`

	// EnvelopeBytes carries the internal/ecl ordered emitter's raw bytes for
	// this call (excluded from the MCP wire response). It is the T2
	// byte-parity anchor consumed by internal/ecl's parity test — the SAME
	// bytes written to the sidecar envelope file, never recomputed
	// separately, so the parity test exercises the real production path.
	EnvelopeBytes []byte `json:"-"`
}

func writeSidecarDefault(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

// TaskStateHead returns the first non-blank line of taskState, mirroring
// `sed -n '/[^[:space:]]/{p;q;}'` (context_handoff.sh:206-207). If every
// line is blank (or taskState is empty), the raw taskState is returned
// verbatim (the bash fallback `_task_state_head="$TASK_STATE"`).
func TaskStateHead(taskState string) string {
	for _, line := range strings.Split(taskState, "\n") {
		if strings.TrimSpace(line) != "" {
			return line
		}
	}
	return taskState
}

// BuildBrief renders the session-handoff brief markdown byte-for-byte per
// the kernel's string-concatenation template (context_handoff.sh:107-168).
// taskState must already carry the resolved default
// (DefaultTaskState when the caller omitted it).
//
// Parity subtleties preserved exactly:
//   - "## Identifiers" has NO "(none recorded)" fallback (it can render as an
//     empty section).
//   - "## Failed approaches" / "## Next steps" fall back to "(none recorded)"
//     based on ARRAY LENGTH (not "any non-empty element") — mirroring bash's
//     `[ "${#ARR[@]}" -gt 0 ]` array-count check.
//   - "## Open variables" is emitted only when len(openVars) > 0 (same
//     array-count rule).
//   - Within any populated list, empty-string entries are skipped
//     (`[ -n "$x" ]` guards) without affecting the fallback decision.
//   - The narrative paragraph is appended only when narrative != "".
//   - The brief ends at the final '\n' after the boolean — no trailing
//     newline is appended beyond the template itself (printf '%s' semantics).
func BuildBrief(taskState, narrative string, anchors, symbols, decisions, failedApproaches, openVars, nextSteps []string, containsToolOrigin bool) string {
	var b strings.Builder
	b.WriteString("# Session Handoff Brief\n\n## Identifiers\n")
	for _, a := range anchors {
		if a != "" {
			b.WriteString("- anchor: " + a + "\n")
		}
	}
	for _, s := range symbols {
		if s != "" {
			b.WriteString("- symbol: " + s + "\n")
		}
	}
	for _, d := range decisions {
		if d != "" {
			b.WriteString("- decision: " + d + "\n")
		}
	}

	b.WriteString("\n## Failed approaches\n")
	if len(failedApproaches) > 0 {
		for _, f := range failedApproaches {
			if f != "" {
				b.WriteString("- " + f + "\n")
			}
		}
	} else {
		b.WriteString("(none recorded)\n")
	}

	b.WriteString("\n## Next steps\n")
	if len(nextSteps) > 0 {
		for _, n := range nextSteps {
			if n != "" {
				b.WriteString("- " + n + "\n")
			}
		}
	} else {
		b.WriteString("(none recorded)\n")
	}

	b.WriteString("\n## Narrative\n\nTask state: " + taskState + "\n")
	if narrative != "" {
		b.WriteString("\n" + narrative + "\n")
	}

	if len(openVars) > 0 {
		b.WriteString("\n## Open variables\n")
		for _, o := range openVars {
			if o != "" {
				b.WriteString("- " + o + "\n")
			}
		}
	}

	b.WriteString("\n## contains_tool_origin\n")
	if containsToolOrigin {
		b.WriteString("true\n")
	} else {
		b.WriteString("false\n")
	}
	return b.String()
}

// Handoff runs compose_handoff over in, returning the brief + envelope, and
// (when write_sidecar, default true) writing the byte-identical file pair to
// out_dir. in.TS and in.ISOTS MUST already be resolved by the caller (the
// server-layer seam) when either was omitted — this function never wall-clocks.
func Handoff(in HandoffInput) (HandoffResult, error) {
	taskState := in.TaskState
	if taskState == "" {
		taskState = DefaultTaskState
	}

	threadID := in.ThreadID
	if threadID == "" {
		if in.SessionID != "" {
			threadID = in.SessionID
		} else {
			threadID = "handoff-" + in.TS
		}
	}

	fromVersion := in.FromVersion
	if fromVersion == "" {
		fromVersion = "n/a"
	}

	briefMD := BuildBrief(taskState, in.Narrative, in.Anchors, in.Symbols, in.Decisions, in.FailedApproaches, in.OpenVars, in.NextSteps, in.ContainsToolOrigin)
	briefBytes := []byte(briefMD)
	briefSHA := hashing.SHA256Hex(briefBytes)
	tokensEst := len(briefBytes) / BytesPerToken
	oversize := tokensEst > AdvisoryTokenTarget

	artifactPath := fmt.Sprintf("handoff-%s.md", in.TS)
	messageID := "msg-context-handoff-" + in.TS
	taskStateHead := TaskStateHead(taskState)

	env := ecl.New(messageID, threadID, fromVersion, artifactPath, briefSHA, int64(len(briefBytes)), in.ISOTS, taskStateHead, in.ContainsToolOrigin)
	envBytes := env.Marshal()

	envMap, err := envelopeToMap(envBytes)
	if err != nil {
		return HandoffResult{}, fmt.Errorf("compose: marshal envelope map: %w", err)
	}

	result := HandoffResult{
		BriefMD:       briefMD,
		BriefSHA256:   briefSHA,
		Envelope:      envMap,
		TokensEst:     tokensEst,
		Oversize:      oversize,
		EnvelopeBytes: envBytes,
	}

	if writeSidecarDefault(in.WriteSidecar) {
		outDir := in.OutDir
		if outDir == "" {
			outDir = DefaultOutDir
		}
		// mkdir-p fail-soft (context_sidecar_dir, lib_context.sh) — a failure
		// here is swallowed; the subsequent WriteFile surfaces any real
		// filesystem problem.
		_ = os.MkdirAll(outDir, 0o755)

		briefPath := filepath.Join(outDir, artifactPath)
		envelopePath := filepath.Join(outDir, fmt.Sprintf("handoff-%s.envelope.json", in.TS))

		if err := os.WriteFile(briefPath, briefBytes, 0o644); err != nil {
			return HandoffResult{}, fmt.Errorf("compose: write brief: %w", err)
		}
		if err := os.WriteFile(envelopePath, envBytes, 0o644); err != nil {
			return HandoffResult{}, fmt.Errorf("compose: write envelope: %w", err)
		}
		result.BriefPath = &briefPath
		result.EnvelopePath = &envelopePath
	}

	return result, nil
}

// envelopeToMap decodes the ordered-emitter bytes into a map for embedding in
// the MCP tool response's "envelope" field. This is a response-shape
// convenience ONLY — the byte-parity contract (T2) is asserted directly
// against ecl.Envelope.Marshal's output (internal/ecl, and the sidecar file
// written above), never against this decoded, re-orderable map.
func envelopeToMap(envBytes []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(envBytes, &m); err != nil {
		return nil, err
	}
	return m, nil
}
