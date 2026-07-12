package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var parityVectors = []string{
	"defaults-only", "fully-populated", "narrative-open-vars",
	"empty-sections", "oversize-brief", "multiline-task-state", "empty-list-entries",
}

func fixtureDir(vector string) string {
	return filepath.Join("..", "..", "fixtures", "parity", vector)
}

func loadVectorInput(t *testing.T, vector string) HandoffInput {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtureDir(vector), "input.json"))
	if err != nil {
		t.Fatalf("read input.json for %s: %v", vector, err)
	}
	var in HandoffInput
	if err := json.Unmarshal(data, &in); err != nil {
		t.Fatalf("unmarshal input.json for %s: %v", vector, err)
	}
	noSidecar := false
	in.WriteSidecar = &noSidecar
	return in
}

// AC-B01: brief_md is byte-identical to the kernel golden brief.md.
func TestParityBriefBytes(t *testing.T) {
	for _, v := range parityVectors {
		v := v
		t.Run(v, func(t *testing.T) {
			in := loadVectorInput(t, v)
			out, err := Handoff(in)
			if err != nil {
				t.Fatalf("Handoff: %v", err)
			}
			golden, err := os.ReadFile(filepath.Join(fixtureDir(v), "brief.md"))
			if err != nil {
				t.Fatalf("read golden brief.md: %v", err)
			}
			if out.BriefMD != string(golden) {
				t.Errorf("brief mismatch for %s:\n got=%q\nwant=%q", v, out.BriefMD, string(golden))
			}
		})
	}
}

// AC-B02: brief_sha256 equals the kernel-recorded golden sha256.
func TestParityBriefSHA(t *testing.T) {
	for _, v := range parityVectors {
		v := v
		t.Run(v, func(t *testing.T) {
			in := loadVectorInput(t, v)
			out, err := Handoff(in)
			if err != nil {
				t.Fatalf("Handoff: %v", err)
			}
			shaBytes, err := os.ReadFile(filepath.Join(fixtureDir(v), "sha256"))
			if err != nil {
				t.Fatalf("read golden sha256: %v", err)
			}
			want := strings.TrimSpace(string(shaBytes))
			if out.BriefSHA256 != want {
				t.Errorf("sha256 mismatch for %s: got=%s want=%s", v, out.BriefSHA256, want)
			}
		})
	}
}

// AC-I01: the handoff parity vector set contains empty-sections,
// oversize-brief, multiline-task-state and empty-list-entries alongside the
// three v0.1 vectors, each complete with input.json, brief.md,
// envelope.json, sha256, advisory.json.
func TestParityVectorsComplete(t *testing.T) {
	if len(parityVectors) < 7 {
		t.Fatalf("expected >= 7 parity vectors, got %d", len(parityVectors))
	}
	want := map[string]bool{
		"defaults-only": false, "fully-populated": false, "narrative-open-vars": false,
		"empty-sections": false, "oversize-brief": false, "multiline-task-state": false, "empty-list-entries": false,
	}
	for _, v := range parityVectors {
		if _, ok := want[v]; ok {
			want[v] = true
		}
		for _, f := range []string{"input.json", "brief.md", "envelope.json", "sha256", "advisory.json"} {
			p := filepath.Join(fixtureDir(v), f)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("vector %s missing %s: %v", v, f, err)
			}
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("required vector %q not present in parityVectors", name)
		}
	}
}

// AC-I03: tokens_est and oversize equal the kernel's own values captured in
// each vector's advisory.json — grounding the advisory assertion in the
// ORACLE's arithmetic, not atomos's own, and auto-guarding the hardcoded
// 1500-token target (if the nexus ever changes it, the kernel's oversize
// flips, the golden drifts, and this test goes red).
func TestParityAdvisoryMatchesKernel(t *testing.T) {
	for _, v := range parityVectors {
		v := v
		t.Run(v, func(t *testing.T) {
			in := loadVectorInput(t, v)
			out, err := Handoff(in)
			if err != nil {
				t.Fatalf("Handoff: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(fixtureDir(v), "advisory.json"))
			if err != nil {
				t.Fatalf("read advisory.json for %s: %v", v, err)
			}
			var advisory struct {
				TokensEst int  `json:"tokens_est"`
				Oversize  bool `json:"oversize"`
			}
			if err := json.Unmarshal(data, &advisory); err != nil {
				t.Fatalf("unmarshal advisory.json for %s: %v", v, err)
			}
			if out.TokensEst != advisory.TokensEst {
				t.Errorf("tokens_est mismatch for %s: got=%d want=%d", v, out.TokensEst, advisory.TokensEst)
			}
			if out.Oversize != advisory.Oversize {
				t.Errorf("oversize mismatch for %s: got=%v want=%v", v, out.Oversize, advisory.Oversize)
			}
		})
	}
}

// AC-E04: fixtures/parity/PROVENANCE records ECM_VERSION plus the nexus
// commit SHA the goldens were captured from.
func TestGoldenProvenanceStamp(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "parity", "PROVENANCE"))
	if err != nil {
		t.Fatalf("read PROVENANCE: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "ECM_VERSION") {
		t.Errorf("PROVENANCE missing ECM_VERSION stamp:\n%s", s)
	}
	if !strings.Contains(s, "NEXUS_COMMIT") {
		t.Errorf("PROVENANCE missing NEXUS_COMMIT stamp:\n%s", s)
	}
}
