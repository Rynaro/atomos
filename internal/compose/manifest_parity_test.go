package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var manifestParityVectors = []string{"manifest-defaults", "manifest-populated", "manifest-tool-origin"}

func manifestFixtureDir(vector string) string {
	return filepath.Join("..", "..", "fixtures", "parity-manifest", vector)
}

func loadManifestVectorInput(t *testing.T, vector string) ManifestInput {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(manifestFixtureDir(vector), "input.json"))
	if err != nil {
		t.Fatalf("read input.json for %s: %v", vector, err)
	}
	var in ManifestInput
	if err := json.Unmarshal(data, &in); err != nil {
		t.Fatalf("unmarshal input.json for %s: %v", vector, err)
	}
	noSidecar := false
	in.WriteSidecar = &noSidecar
	return in
}

// AC-H03: the canonical manifest bytes are byte-identical to the kernel
// golden manifest.json (the vector's committed input.json carries the
// caller-supplied file_floor_reason, exactly as the kernel's sole manifest
// writer unconditionally appends one — M7).
func TestManifestParityBytes(t *testing.T) {
	for _, v := range manifestParityVectors {
		v := v
		t.Run(v, func(t *testing.T) {
			in := loadManifestVectorInput(t, v)
			out, err := ExternalizeManifest(in)
			if err != nil {
				t.Fatalf("ExternalizeManifest: %v", err)
			}
			golden, err := os.ReadFile(filepath.Join(manifestFixtureDir(v), "manifest.json"))
			if err != nil {
				t.Fatalf("read golden manifest.json: %v", err)
			}
			if string(out.ManifestBytes) != string(golden) {
				t.Errorf("manifest byte mismatch for %s:\n got=%q\nwant=%q", v, out.ManifestBytes, golden)
			}
		})
	}
}

// AC-H04: manifest_sha256 equals the golden sha256 recorded for those bytes.
func TestManifestParitySHA(t *testing.T) {
	for _, v := range manifestParityVectors {
		v := v
		t.Run(v, func(t *testing.T) {
			in := loadManifestVectorInput(t, v)
			out, err := ExternalizeManifest(in)
			if err != nil {
				t.Fatalf("ExternalizeManifest: %v", err)
			}
			shaBytes, err := os.ReadFile(filepath.Join(manifestFixtureDir(v), "sha256"))
			if err != nil {
				t.Fatalf("read golden sha256: %v", err)
			}
			want := strings.TrimSpace(string(shaBytes))
			if out.ManifestSHA256 != want {
				t.Errorf("sha256 mismatch for %s: got=%s want=%s", v, out.ManifestSHA256, want)
			}
		})
	}
}

// AC-H22: the SAME vector run WITHOUT file_floor_reason (the default,
// 10-key path most callers get) is byte-identical to the vector's
// jq-derived core.json.
func TestManifestCoreParityBytes(t *testing.T) {
	for _, v := range manifestParityVectors {
		v := v
		t.Run(v, func(t *testing.T) {
			in := loadManifestVectorInput(t, v)
			in.FileFloorReason = "" // the default path (Q3/AC-H15)
			out, err := ExternalizeManifest(in)
			if err != nil {
				t.Fatalf("ExternalizeManifest: %v", err)
			}
			golden, err := os.ReadFile(filepath.Join(manifestFixtureDir(v), "core.json"))
			if err != nil {
				t.Fatalf("read golden core.json: %v", err)
			}
			if string(out.ManifestBytes) != string(golden) {
				t.Errorf("core manifest byte mismatch for %s:\n got=%q\nwant=%q", v, out.ManifestBytes, golden)
			}
		})
	}
}

// AC-H23: the reason-less run's manifest_sha256 equals core.sha256.
func TestManifestCoreParitySHA(t *testing.T) {
	for _, v := range manifestParityVectors {
		v := v
		t.Run(v, func(t *testing.T) {
			in := loadManifestVectorInput(t, v)
			in.FileFloorReason = ""
			out, err := ExternalizeManifest(in)
			if err != nil {
				t.Fatalf("ExternalizeManifest: %v", err)
			}
			shaBytes, err := os.ReadFile(filepath.Join(manifestFixtureDir(v), "core.sha256"))
			if err != nil {
				t.Fatalf("read golden core.sha256: %v", err)
			}
			want := strings.TrimSpace(string(shaBytes))
			if out.ManifestSHA256 != want {
				t.Errorf("core sha256 mismatch for %s: got=%s want=%s", v, out.ManifestSHA256, want)
			}
		})
	}
}

// AC-I05: at least three frozen-created_at manifest vectors exist, each
// complete with input.json, manifest.json, sha256, core.json, core.sha256.
func TestManifestVectorsComplete(t *testing.T) {
	if len(manifestParityVectors) < 3 {
		t.Fatalf("expected >= 3 manifest parity vectors, got %d", len(manifestParityVectors))
	}
	want := map[string]bool{"manifest-defaults": false, "manifest-populated": false, "manifest-tool-origin": false}
	for _, v := range manifestParityVectors {
		if _, ok := want[v]; ok {
			want[v] = true
		}
		for _, f := range []string{"input.json", "manifest.json", "sha256", "core.json", "core.sha256"} {
			p := filepath.Join(manifestFixtureDir(v), f)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("vector %s missing %s: %v", v, f, err)
			}
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("required vector %q not present in manifestParityVectors", name)
		}
	}
}
