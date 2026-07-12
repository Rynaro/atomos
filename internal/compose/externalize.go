// Package compose (continued): compose_externalize_manifest — a pure
// function mirroring ONLY the manifest-build portion of `eidolons context
// externalize` (cli/src/context_externalize.sh:116-146): the MANIFEST_JSON
// object plus the file-floor writer. Everything past that point in the
// kernel — durable memory, a ledger append, and a decision-log write — is
// permanently out of reach here (ADR §0.1, §2.4); see
// internal/ecl/envelope.go's package doc for the register this package
// follows when describing that boundary in prose.
//
// PURE HANDLER (AC-H18): reads no wall clock of its own. created_at/ts are
// caller inputs; an omitted pair is resolved at the single server-layer
// seam (internal/server), never here.
package compose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rynaro/atomos/internal/hashing"
)

// ManifestInput is the compose_externalize_manifest MCP tool input
// (BUILD-SPEC-v0.2 Track H3 / ADR §2.4).
type ManifestInput struct {
	Summary            string   `json:"summary,omitempty"`
	Anchors            []string `json:"anchors,omitempty"`
	Symbols            []string `json:"symbols,omitempty"`
	Decisions          []string `json:"decisions,omitempty"`
	FailedApproaches   []string `json:"failed_approaches,omitempty"`
	OpenVars           []string `json:"open_vars,omitempty"`
	ContainsToolOrigin bool     `json:"contains_tool_origin,omitempty"`
	SessionID          string   `json:"session_id,omitempty"`
	CreatedAt          string   `json:"created_at,omitempty"` // PARITY-CRITICAL; inside the hashed document (M1)
	TS                 string   `json:"ts,omitempty"`         // sidecar FILENAME ONLY — never enters the document
	FileFloorReason    string   `json:"file_floor_reason,omitempty"`
	WriteSidecar       *bool    `json:"write_sidecar,omitempty"`
	OutDir             string   `json:"out_dir,omitempty"`
}

// ManifestResult is the compose_externalize_manifest MCP tool output.
type ManifestResult struct {
	Manifest       map[string]any `json:"manifest"`
	ManifestSHA256 string         `json:"manifest_sha256"`
	ManifestPath   *string        `json:"manifest_path"`

	// ManifestBytes carries the Manifest.Marshal() bytes for this call — the
	// SAME bytes that are hashed and (when write_sidecar) written, never
	// recomputed separately (M0; mirrors HandoffResult.EnvelopeBytes).
	ManifestBytes []byte `json:"-"`
}

// NormalizeList mirrors the kernel's context_json_array (lib_context.sh:
// `jq -R -s 'split("\n") | map(select(length > 0))'` over a newline-joined
// dump of the arguments): split every element on "\n" and keep only its
// non-empty parts. Only "\n" splits — "\r" is not special (AC-H25: a
// "\r\n" entry keeps a trailing "\r" on the preceding element). Nothing
// surviving yields an empty, non-nil slice, so it marshals as the inline
// `[]` (jsonx.WriteArr) rather than being mistaken for an absent field.
func NormalizeList(items []string) []string {
	out := []string{}
	for _, it := range items {
		for _, part := range strings.Split(it, "\n") {
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// ExternalizeManifest runs compose_externalize_manifest over in, returning
// the manifest + its SHA-256 (M0), and (when write_sidecar, default true)
// writing the byte-identical file-floor sidecar to out_dir. in.CreatedAt and
// in.TS MUST already be resolved by the caller (the server-layer seam) when
// omitted — this function never wall-clocks (AC-H18).
func ExternalizeManifest(in ManifestInput) (ManifestResult, error) {
	summary := in.Summary
	if summary == "" {
		summary = ManifestDefaultSummary
	}

	var reason *string
	if in.FileFloorReason != "" {
		r := in.FileFloorReason
		reason = &r
	}

	m := Manifest{
		EcmVersion:         ManifestEcmVersion,
		Summary:            summary,
		Anchors:            NormalizeList(in.Anchors),
		Symbols:            NormalizeList(in.Symbols),
		Decisions:          NormalizeList(in.Decisions),
		FailedApproaches:   NormalizeList(in.FailedApproaches),
		OpenVars:           NormalizeList(in.OpenVars),
		ContainsToolOrigin: in.ContainsToolOrigin,
		SessionID:          in.SessionID,
		CreatedAt:          in.CreatedAt,
		FileFloorReason:    reason,
	}

	manifestBytes := m.Marshal()
	sha := hashing.SHA256Hex(manifestBytes)

	manifestMap, err := manifestToMap(manifestBytes)
	if err != nil {
		return ManifestResult{}, fmt.Errorf("compose: marshal manifest map: %w", err)
	}

	result := ManifestResult{
		Manifest:       manifestMap,
		ManifestSHA256: sha,
		ManifestBytes:  manifestBytes,
	}

	if writeSidecarDefault(in.WriteSidecar) {
		outDir := in.OutDir
		if outDir == "" {
			outDir = DefaultOutDir
		}
		// mkdir-p fail-soft (context_sidecar_dir, lib_context.sh) — a failure
		// here is swallowed; the subsequent WriteFile surfaces any real
		// filesystem problem as the sole hard-error path in this tool
		// (AC-H27).
		_ = os.MkdirAll(outDir, 0o755)

		manifestPath := filepath.Join(outDir, fmt.Sprintf("externalized-%s.json", in.TS))
		if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
			return ManifestResult{}, fmt.Errorf("compose: write manifest: %w", err)
		}
		result.ManifestPath = &manifestPath
	}

	return result, nil
}

// manifestToMap decodes the Marshal()'d bytes into a map for the MCP tool
// response's "manifest" field (AC-H24: the response object is decoded FROM
// the hashed bytes and deep-equals them — never built by a second,
// independent serialization path, exactly as compose.Handoff's
// envelopeToMap does for the envelope).
func manifestToMap(data []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
