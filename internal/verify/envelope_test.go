package verify

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixtureExpected struct {
	Verdict       string   `json:"verdict"`
	Mode          string   `json:"mode"`
	Blocked       bool     `json:"blocked"`
	ExpectedSHA   string   `json:"expected_sha"`
	ActualSHA     string   `json:"actual_sha"`
	MissingFields []string `json:"missing_fields"`
}

// fixtureCase is one <fixtures/{conformant,failing}>/<name> directory: an
// envelope JSON file (named "envelope.json" or "<basename>.envelope.json" to
// exercise the sibling-payload convention) plus an expected.json verdict.
type fixtureCase struct {
	dir          string
	envelopePath string
	expected     fixtureExpected
}

func loadFixtures(t *testing.T, root string) []fixtureCase {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read %s: %v", root, err)
	}
	var cases []fixtureCase
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		expData, err := os.ReadFile(filepath.Join(dir, "expected.json"))
		if err != nil {
			t.Fatalf("read expected.json in %s: %v", dir, err)
		}
		var exp fixtureExpected
		if err := json.Unmarshal(expData, &exp); err != nil {
			t.Fatalf("unmarshal expected.json in %s: %v", dir, err)
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		var envelopePath string
		for _, f := range files {
			if f.Name() == "expected.json" || f.IsDir() {
				continue
			}
			if filepath.Ext(f.Name()) == ".json" {
				envelopePath = filepath.Join(dir, f.Name())
				break
			}
		}
		if envelopePath == "" {
			t.Fatalf("no envelope JSON file found in %s", dir)
		}
		cases = append(cases, fixtureCase{dir: dir, envelopePath: envelopePath, expected: exp})
	}
	return cases
}

// AC-C01: every fixture under fixtures/conformant/ and fixtures/failing/
// yields its declared verdict across the full matrix.
func TestVerdictMatrix(t *testing.T) {
	roots := []string{
		filepath.Join("..", "..", "fixtures", "conformant"),
		filepath.Join("..", "..", "fixtures", "failing"),
	}
	for _, root := range roots {
		for _, c := range loadFixtures(t, root) {
			c := c
			t.Run(filepath.Base(root)+"/"+filepath.Base(c.dir), func(t *testing.T) {
				mode := c.expected.Mode
				if mode == "" {
					mode = "warn"
				}
				out, err := Envelope(EnvelopeInput{EnvelopePath: c.envelopePath, Mode: mode})
				if err != nil {
					t.Fatalf("Envelope: %v", err)
				}
				if string(out.Verdict) != c.expected.Verdict {
					t.Errorf("verdict = %s, want %s (message=%q)", out.Verdict, c.expected.Verdict, out.Message)
				}
				if out.Blocked != c.expected.Blocked {
					t.Errorf("blocked = %v, want %v", out.Blocked, c.expected.Blocked)
				}
				for _, mf := range c.expected.MissingFields {
					if !strings.Contains(out.Message, mf) {
						t.Errorf("message %q does not mention missing field %q", out.Message, mf)
					}
				}
			})
		}
	}
}

// AC-C02: a tampered payload verdicts tamper, carrying the envelope's tag as
// expected_sha alongside the recomputed actual_sha.
func TestTamperShaReporting(t *testing.T) {
	out, err := Envelope(EnvelopeInput{EnvelopePath: filepath.Join("..", "..", "fixtures", "failing", "tamper", "envelope.json")})
	if err != nil {
		t.Fatalf("Envelope: %v", err)
	}
	if out.Verdict != VerdictTamper {
		t.Fatalf("verdict = %s, want tamper", out.Verdict)
	}
	if out.ExpectedSHA == "" || out.ActualSHA == "" || out.ExpectedSHA == out.ActualSHA {
		t.Errorf("expected/actual SHA not properly reported: expected=%s actual=%s", out.ExpectedSHA, out.ActualSHA)
	}
}

// AC-C03: envelope_version != "2.0" (e.g. the composer's "1.0") records an
// advisory warning while the verdict is still computed normally (pass).
func TestEnvelopeVersionWarnNotFail(t *testing.T) {
	out, err := Envelope(EnvelopeInput{EnvelopePath: filepath.Join("..", "..", "fixtures", "conformant", "version-warn", "envelope.json")})
	if err != nil {
		t.Fatalf("Envelope: %v", err)
	}
	if out.Verdict != VerdictPass {
		t.Fatalf("verdict = %s, want pass despite version drift", out.Verdict)
	}
	if out.Warning == "" {
		t.Errorf("expected a non-empty advisory Warning for envelope_version != 2.0")
	}
}

// AC-C04: a placeholder integrity.value verdicts unverifiable — never a
// failure verdict, never blocked, in ANY mode.
func TestPlaceholderUnverifiable(t *testing.T) {
	for _, mode := range []string{"warn", "block"} {
		out, err := Envelope(EnvelopeInput{
			EnvelopePath: filepath.Join("..", "..", "fixtures", "failing", "unverifiable-placeholder", "envelope.json"),
			Mode:         mode,
		})
		if err != nil {
			t.Fatalf("Envelope: %v", err)
		}
		if out.Verdict != VerdictUnverifiable {
			t.Errorf("mode=%s verdict = %s, want unverifiable", mode, out.Verdict)
		}
		if out.Blocked {
			t.Errorf("mode=%s unverifiable must never block", mode)
		}
	}
}

// AC-C05: when artifact.path does not resolve directly, the sibling
// <envelope minus .envelope.json> is used as the payload.
func TestSiblingPayloadFallback(t *testing.T) {
	out, err := Envelope(EnvelopeInput{EnvelopePath: filepath.Join("..", "..", "fixtures", "conformant", "sibling-fallback", "case.envelope.json")})
	if err != nil {
		t.Fatalf("Envelope: %v", err)
	}
	if out.Verdict != VerdictPass {
		t.Fatalf("verdict = %s, want pass via sibling fallback (message=%s)", out.Verdict, out.Message)
	}
}

// AC-C06: blocked:true only for {tamper, inconsistent, missing_payload,
// unsupported_algo} under mode:block; every other verdict/mode combination
// reports blocked:false.
func TestBlockedFlagMatrix(t *testing.T) {
	cases := []struct {
		name        string
		fixturePath string
		wantVerdict Verdict
	}{
		{"tamper", filepath.Join("failing", "tamper", "envelope.json"), VerdictTamper},
		{"inconsistent", filepath.Join("failing", "inconsistent", "envelope.json"), VerdictInconsistent},
		{"missing_payload", filepath.Join("failing", "missing-payload", "envelope.json"), VerdictMissingPayload},
		{"unsupported_algo", filepath.Join("failing", "unsupported-algo", "envelope.json"), VerdictUnsupportedAlgo},
		{"unverifiable", filepath.Join("failing", "unverifiable-placeholder", "envelope.json"), VerdictUnverifiable},
		{"pass", filepath.Join("conformant", "pass", "envelope.json"), VerdictPass},
	}
	for _, c := range cases {
		for _, mode := range []string{"warn", "block"} {
			p := filepath.Join("..", "..", "fixtures", c.fixturePath)
			out, err := Envelope(EnvelopeInput{EnvelopePath: p, Mode: mode})
			if err != nil {
				t.Fatalf("%s/%s: Envelope: %v", c.name, mode, err)
			}
			if out.Verdict != c.wantVerdict {
				t.Fatalf("%s/%s: verdict = %s, want %s", c.name, mode, out.Verdict, c.wantVerdict)
			}
			wantBlocked := mode == "block" && blockingVerdicts[c.wantVerdict]
			if out.Blocked != wantBlocked {
				t.Errorf("%s/%s: blocked = %v, want %v", c.name, mode, out.Blocked, wantBlocked)
			}
		}
	}
}

// AC-C07: every outcome (including failures) returns a normal result, never
// a process exit — this test's own completion is part of the proof; it also
// asserts Envelope never returns a Go error for well-formed-but-failing
// fixtures (errors are reserved for true usage mistakes upstream of the
// tool call, e.g. bad JSON, which itself still yields VerdictMalformed
// rather than an exit).
func TestVerifyNeverExitsProcess(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "fixtures", "failing", "tamper", "envelope.json"),
		filepath.Join("..", "..", "fixtures", "failing", "inconsistent", "envelope.json"),
		filepath.Join("..", "..", "fixtures", "failing", "missing-payload", "envelope.json"),
		filepath.Join("..", "..", "fixtures", "failing", "unsupported-algo", "envelope.json"),
		filepath.Join("..", "..", "fixtures", "failing", "malformed-missing-fields", "envelope.json"),
		filepath.Join("..", "..", "fixtures", "failing", "malformed-json", "envelope.json"),
		filepath.Join("..", "..", "fixtures", "failing", "unverifiable-placeholder", "envelope.json"),
	}
	for _, p := range paths {
		if _, err := Envelope(EnvelopeInput{EnvelopePath: p, Mode: "block"}); err != nil {
			t.Fatalf("Envelope(%s) returned a Go error instead of a reported verdict: %v", p, err)
		}
	}
	// Reaching this line proves the process is still alive — no os.Exit was
	// ever called by the verify package.
}

// AC-C08: an envelope missing any required field verdicts malformed with the
// missing field names listed in message.
func TestMalformedMissingFields(t *testing.T) {
	out, err := Envelope(EnvelopeInput{EnvelopePath: filepath.Join("..", "..", "fixtures", "failing", "malformed-missing-fields", "envelope.json")})
	if err != nil {
		t.Fatalf("Envelope: %v", err)
	}
	if out.Verdict != VerdictMalformed {
		t.Fatalf("verdict = %s, want malformed", out.Verdict)
	}
	for _, want := range []string{"to.eidolon", "integrity.value"} {
		if !strings.Contains(out.Message, want) {
			t.Errorf("message %q missing field name %q", out.Message, want)
		}
	}
}
