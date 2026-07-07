// Package verify implements verify_envelope and verify_pins: pure
// deterministic ports of `eidolons verify-envelope` (cli/src/verify_envelope.sh)
// and the ECM §3.2 pin-survival probe.
//
// PURE HANDLER (AC-F04): reads no wall clock. Never process-exits — exit-3
// enforcement (ECL §6.2.2) stays with the kernel/orchestrator (fail-open P0).
package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rynaro/atomos/internal/hashing"
)

// Verdict is the closed verify_envelope outcome set, matching the kernel's
// verdict matrix (verify_envelope.sh:45-47) exactly.
type Verdict string

const (
	VerdictPass            Verdict = "pass"
	VerdictTamper          Verdict = "tamper"
	VerdictInconsistent    Verdict = "inconsistent"
	VerdictUnverifiable    Verdict = "unverifiable"
	VerdictMissingPayload  Verdict = "missing_payload"
	VerdictUnsupportedAlgo Verdict = "unsupported_algo"
	VerdictMalformed       Verdict = "malformed"
)

// expectedEnvelopeVersion is what the kernel's verifier accepts without a
// warning (verify_envelope.sh:151-155). Any other value (including the
// composer's own "1.0" — parity trap 2) is accepted but WARNED on, never
// failed.
const expectedEnvelopeVersion = "2.0"

// blockingVerdicts are the verdicts that set blocked:true under mode=block
// (verify_envelope.sh:81-85).
var blockingVerdicts = map[Verdict]bool{
	VerdictTamper:          true,
	VerdictInconsistent:    true,
	VerdictMissingPayload:  true,
	VerdictUnsupportedAlgo: true,
}

// EnvelopeInput is the verify_envelope MCP tool input (ADR §2.2). Exactly one
// of Envelope/EnvelopePath must resolve to a JSON object; Payload/PayloadPath
// are optional (inline envelope with no payload given resolves to
// missing_payload, per the spec).
type EnvelopeInput struct {
	Envelope     map[string]any `json:"envelope,omitempty"`
	EnvelopePath string         `json:"envelope_path,omitempty"`
	Payload      *string        `json:"payload,omitempty"`
	PayloadPath  string         `json:"payload_path,omitempty"`
	Mode         string         `json:"mode,omitempty"` // "warn" (default) | "block"
}

// EnvelopeResult is the verify_envelope MCP tool output (ADR §2.2).
type EnvelopeResult struct {
	Verdict      Verdict `json:"verdict"`
	ExpectedSHA  string  `json:"expected_sha,omitempty"`
	ActualSHA    string  `json:"actual_sha,omitempty"`
	Blocked      bool    `json:"blocked"`
	Message      string  `json:"message"`
	From         string  `json:"from,omitempty"`
	To           string  `json:"to,omitempty"`
	Performative string  `json:"performative,omitempty"`
	Warning      string  `json:"warning,omitempty"`
}

var requiredFields = []string{
	"envelope_version",
	"from.eidolon",
	"to.eidolon",
	"performative",
	"artifact.path",
	"integrity.method",
	"integrity.value",
}

// Envelope runs the verify_envelope tool over in.
func Envelope(in EnvelopeInput) (EnvelopeResult, error) {
	mode := in.Mode
	if mode == "" {
		mode = "warn"
	}

	env, loadErr := resolveEnvelope(in)
	if loadErr != nil {
		return finish(VerdictMalformed, mode, "", "", loadErr.Error(), "", "", ""), nil
	}

	missing := missingFields(env)
	if len(missing) > 0 {
		return finish(VerdictMalformed, mode, "", "", "missing required fields: "+strings.Join(missing, ", "), "", "", ""), nil
	}

	from := stringAt(env, "from", "eidolon")
	to := stringAt(env, "to", "eidolon")
	perf, _ := env["performative"].(string)

	var warning string
	evVersion, _ := env["envelope_version"].(string)
	if evVersion != expectedEnvelopeVersion {
		warning = "unrecognized envelope_version '" + evVersion + "' (expected 2.0) — proceeding"
	}

	method := stringAt(env, "integrity", "method")
	if method != "sha256" {
		res := finish(VerdictUnsupportedAlgo, mode, "", "", "integrity.method '"+method+"' is not sha256", from, to, perf)
		res.Warning = warning
		return res, nil
	}

	integrityValue := stringAt(env, "integrity", "value")
	if isPlaceholder(integrityValue) {
		res := finish(VerdictUnverifiable, mode, "", "", "integrity.value is a placeholder ('"+integrityValue+"') — parent must fill the SHA before verification", from, to, perf)
		res.Warning = warning
		return res, nil
	}

	artifactSHA := stringAt(env, "artifact", "sha256")
	if artifactSHA != "" && artifactSHA != integrityValue {
		res := finish(VerdictInconsistent, mode, integrityValue, artifactSHA, "artifact.sha256 != integrity.value (envelope self-inconsistent)", from, to, perf)
		res.Warning = warning
		return res, nil
	}

	payload, payloadErr := resolvePayload(in, env)
	if payloadErr != nil {
		artifactPath := stringAt(env, "artifact", "path")
		res := finish(VerdictMissingPayload, mode, "", "", "payload not found at artifact.path '"+artifactPath+"'", from, to, perf)
		res.Warning = warning
		return res, nil
	}

	actualSHA := hashing.SHA256Hex(payload)
	if actualSHA == integrityValue {
		res := finish(VerdictPass, mode, integrityValue, actualSHA, "payload SHA-256 matches integrity tag", from, to, perf)
		res.Warning = warning
		return res, nil
	}
	res := finish(VerdictTamper, mode, integrityValue, actualSHA, "payload SHA-256 does NOT match integrity tag — possible tampering or stale envelope", from, to, perf)
	res.Warning = warning
	return res, nil
}

func finish(v Verdict, mode, expected, actual, message, from, to, perf string) EnvelopeResult {
	blocked := mode == "block" && blockingVerdicts[v]
	return EnvelopeResult{
		Verdict:      v,
		ExpectedSHA:  expected,
		ActualSHA:    actual,
		Blocked:      blocked,
		Message:      message,
		From:         from,
		To:           to,
		Performative: perf,
	}
}

// isPlaceholder mirrors verify_envelope.sh:167-170's placeholder guard.
func isPlaceholder(v string) bool {
	switch {
	case v == "":
		return true
	case v == "null":
		return true
	case strings.HasPrefix(v, "PARENT_FILLS_"):
		return true
	case strings.HasPrefix(v, "<"):
		return true
	case strings.HasPrefix(v, "TODO"):
		return true
	}
	return false
}

func missingFields(env map[string]any) []string {
	var missing []string
	for _, f := range requiredFields {
		parts := strings.SplitN(f, ".", 2)
		var present bool
		if len(parts) == 2 {
			present = stringAt(env, parts[0], parts[1]) != "" || anyAt(env, parts[0], parts[1])
		} else {
			present = env[f] != nil
		}
		if !present {
			missing = append(missing, f)
		}
	}
	return missing
}

// anyAt reports whether env[a][b] is present at all (non-nil), independent
// of its type — a required field could theoretically be non-string.
func anyAt(env map[string]any, a, b string) bool {
	sub, ok := env[a].(map[string]any)
	if !ok {
		return false
	}
	v, ok := sub[b]
	return ok && v != nil
}

func stringAt(env map[string]any, a, b string) string {
	sub, ok := env[a].(map[string]any)
	if !ok {
		return ""
	}
	s, _ := sub[b].(string)
	return s
}

// resolveEnvelope loads the envelope object from either the inline map or
// the on-disk path, returning a malformed error on invalid/unreadable JSON
// (verify_envelope.sh check 1).
func resolveEnvelope(in EnvelopeInput) (map[string]any, error) {
	if in.Envelope != nil {
		return in.Envelope, nil
	}
	if in.EnvelopePath == "" {
		return nil, errMalformed("no envelope or envelope_path given")
	}
	data, err := os.ReadFile(in.EnvelopePath)
	if err != nil {
		return nil, errMalformed("envelope not found: " + in.EnvelopePath)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, errMalformed("not valid JSON")
	}
	return m, nil
}

type malformedError struct{ msg string }

func (e *malformedError) Error() string { return e.msg }
func errMalformed(msg string) error     { return &malformedError{msg} }

// resolvePayload resolves the payload bytes: inline payload, else
// payload_path, else artifact.path relative to the envelope directory
// (with the sibling <name minus .envelope.json> fallback), matching
// verify_envelope.sh:177-187.
func resolvePayload(in EnvelopeInput, env map[string]any) ([]byte, error) {
	if in.Payload != nil {
		return []byte(*in.Payload), nil
	}
	if in.PayloadPath != "" {
		return os.ReadFile(in.PayloadPath)
	}
	if in.EnvelopePath == "" {
		return nil, errMalformed("no payload resolvable without envelope_path")
	}
	artifactPath := stringAt(env, "artifact", "path")
	envDir := filepath.Dir(in.EnvelopePath)
	candidate := filepath.Join(envDir, artifactPath)
	if data, err := os.ReadFile(candidate); err == nil {
		return data, nil
	}
	sibling := strings.TrimSuffix(in.EnvelopePath, ".envelope.json")
	if sibling != in.EnvelopePath {
		if data, err := os.ReadFile(sibling); err == nil {
			return data, nil
		}
	}
	return nil, errMalformed("missing payload")
}
