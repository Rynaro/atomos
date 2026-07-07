package ecl_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rynaro/atomos/internal/compose"
)

var parityVectors = []string{"defaults-only", "fully-populated", "narrative-open-vars"}

func fixtureDir(vector string) string {
	return filepath.Join("..", "..", "fixtures", "parity", vector)
}

func loadInput(t *testing.T, vector string) compose.HandoffInput {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtureDir(vector), "input.json"))
	if err != nil {
		t.Fatalf("read input.json for %s: %v", vector, err)
	}
	var in compose.HandoffInput
	if err := json.Unmarshal(data, &in); err != nil {
		t.Fatalf("unmarshal input.json for %s: %v", vector, err)
	}
	noSidecar := false
	in.WriteSidecar = &noSidecar
	return in
}

// AC-B13: T2 envelope parity — the internal/ecl ordered emitter's bytes
// (as produced by the real compose_handoff production path) must equal the
// kernel-captured golden envelope.json byte-for-byte.
func TestEnvelopeT2Parity(t *testing.T) {
	for _, v := range parityVectors {
		v := v
		t.Run(v, func(t *testing.T) {
			in := loadInput(t, v)
			out, err := compose.Handoff(in)
			if err != nil {
				t.Fatalf("compose.Handoff: %v", err)
			}
			golden, err := os.ReadFile(filepath.Join(fixtureDir(v), "envelope.json"))
			if err != nil {
				t.Fatalf("read golden envelope.json: %v", err)
			}
			if string(out.EnvelopeBytes) != string(golden) {
				t.Errorf("envelope byte mismatch for %s:\n got=%q\nwant=%q", v, out.EnvelopeBytes, golden)
			}
		})
	}
}
