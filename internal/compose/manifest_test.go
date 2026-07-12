package compose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Rynaro/atomos/internal/hashing"
)

// AC-H09: no summary => the kernel's default sentence verbatim
// (context_externalize.sh:100).
func TestManifestDefaultSummary(t *testing.T) {
	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", WriteSidecar: boolPtr(false)}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	if out.Manifest["summary"] != ManifestDefaultSummary {
		t.Errorf("summary = %v, want %q", out.Manifest["summary"], ManifestDefaultSummary)
	}
}

// AC-H10: an empty or absent session_id renders as JSON null, never "".
func TestManifestSessionIDNull(t *testing.T) {
	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", WriteSidecar: boolPtr(false)}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	if out.Manifest["session_id"] != nil {
		t.Errorf("session_id = %v, want nil (JSON null)", out.Manifest["session_id"])
	}

	in2 := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", SessionID: "sess-1", WriteSidecar: boolPtr(false)}
	out2, err := ExternalizeManifest(in2)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	if out2.Manifest["session_id"] != "sess-1" {
		t.Errorf("session_id = %v, want sess-1", out2.Manifest["session_id"])
	}
}

// AC-H11 (M3): empty-string entries are absent from the emitted array.
func TestManifestDropsEmptyEntries(t *testing.T) {
	in := ManifestInput{
		CreatedAt:    "2026-01-01T00:00:00Z",
		Anchors:      []string{"", "kept:1", ""},
		WriteSidecar: boolPtr(false),
	}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	anchors, _ := out.Manifest["anchors"].([]any)
	if len(anchors) != 1 || anchors[0] != "kept:1" {
		t.Errorf("anchors = %v, want [\"kept:1\"]", anchors)
	}
}

// AC-H12 (M2): an embedded newline splits the entry into one array element
// per non-empty line — context_json_array semantics.
func TestManifestSplitsNewlineEntries(t *testing.T) {
	in := ManifestInput{
		CreatedAt:    "2026-01-01T00:00:00Z",
		Decisions:    []string{"multi\nline decision"},
		WriteSidecar: boolPtr(false),
	}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	decisions, _ := out.Manifest["decisions"].([]any)
	want := []any{"multi", "line decision"}
	if !reflect.DeepEqual(decisions, want) {
		t.Errorf("decisions = %v, want %v", decisions, want)
	}
}

// AC-H13: an absent list, or one whose every entry vanishes, renders as the
// inline empty array `[]` on a single line (jq-exact — no interior newline).
func TestManifestEmptyArrayInline(t *testing.T) {
	in := ManifestInput{
		CreatedAt:    "2026-01-01T00:00:00Z",
		Anchors:      []string{"", ""}, // every entry vanishes
		WriteSidecar: boolPtr(false),
	}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	if !strings.Contains(string(out.ManifestBytes), "\"anchors\": [],\n") {
		t.Errorf("expected inline empty array for anchors; bytes=%s", out.ManifestBytes)
	}
	if !strings.Contains(string(out.ManifestBytes), "\"symbols\": [],\n") {
		t.Errorf("expected inline empty array for symbols (absent field); bytes=%s", out.ManifestBytes)
	}
}

// AC-H14 (M6): a caller-supplied file_floor_reason is the FINAL key, after
// created_at.
func TestFileFloorReasonLast(t *testing.T) {
	in := ManifestInput{
		CreatedAt:       "2026-01-01T00:00:00Z",
		FileFloorReason: "crystalium absent",
		WriteSidecar:    boolPtr(false),
	}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	s := string(out.ManifestBytes)
	createdIdx := strings.Index(s, "\"created_at\"")
	reasonIdx := strings.Index(s, "\"file_floor_reason\"")
	if createdIdx == -1 || reasonIdx == -1 || reasonIdx < createdIdx {
		t.Fatalf("file_floor_reason must appear after created_at; bytes=%s", s)
	}
	// It must also be the actual last key: created_at gets a trailing comma,
	// and the object closes right after file_floor_reason's line.
	if !strings.HasSuffix(strings.TrimRight(s, "\n"), "}") {
		t.Fatalf("manifest does not end with a closing brace: %q", s)
	}
	lastKeyLine := s[strings.LastIndex(s, "\"file_floor_reason\""):]
	if strings.Contains(lastKeyLine[:strings.IndexByte(lastKeyLine, '\n')], ",") {
		t.Errorf("file_floor_reason line should have NO trailing comma (last key): %q", lastKeyLine)
	}
}

// AC-H15 (Q3): file_floor_reason omitted => exactly ten keys, no
// file_floor_reason at all — atomos never authors a reason it cannot
// observe.
func TestFileFloorReasonAbsentByDefault(t *testing.T) {
	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", WriteSidecar: boolPtr(false)}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	if _, ok := out.Manifest["file_floor_reason"]; ok {
		t.Errorf("file_floor_reason present when omitted: %v", out.Manifest)
	}
	if len(out.Manifest) != 10 {
		t.Errorf("manifest has %d keys, want 10: %v", len(out.Manifest), out.Manifest)
	}
}

// AC-H16 (M5): ecm_version is always the hardcoded literal "0.1" — read
// from no file.
func TestManifestEcmVersionLiteral(t *testing.T) {
	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", WriteSidecar: boolPtr(false)}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	if out.Manifest["ecm_version"] != "0.1" {
		t.Errorf("ecm_version = %v, want 0.1", out.Manifest["ecm_version"])
	}
}

// AC-H05 (M0): the sidecar file's own SHA-256 equals the reported
// manifest_sha256 — hash what you write.
func TestManifestSidecarBytesHashToReportedSHA(t *testing.T) {
	dir := t.TempDir()
	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", TS: "20260101T000000Z", OutDir: dir}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	if out.ManifestPath == nil {
		t.Fatal("ManifestPath is nil")
	}
	data, err := os.ReadFile(*out.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest file: %v", err)
	}
	got := hashing.SHA256Hex(data)
	if got != out.ManifestSHA256 {
		t.Errorf("file sha256 = %s, want %s (reported manifest_sha256)", got, out.ManifestSHA256)
	}
}

// AC-H06: manifest_sha256 does not depend on whether a sidecar was written.
func TestManifestSHAIndependentOfSidecar(t *testing.T) {
	base := ManifestInput{
		CreatedAt: "2026-01-01T00:00:00Z",
		Anchors:   []string{"a:1"},
		Summary:   "same inputs",
	}
	withSidecar := base
	withSidecar.TS = "20260101T000000Z"
	withSidecar.OutDir = t.TempDir()
	withSidecar.WriteSidecar = boolPtr(true)

	withoutSidecar := base
	withoutSidecar.WriteSidecar = boolPtr(false)

	out1, err := ExternalizeManifest(withSidecar)
	if err != nil {
		t.Fatalf("ExternalizeManifest (sidecar): %v", err)
	}
	out2, err := ExternalizeManifest(withoutSidecar)
	if err != nil {
		t.Fatalf("ExternalizeManifest (no sidecar): %v", err)
	}
	if out1.ManifestSHA256 != out2.ManifestSHA256 {
		t.Errorf("manifest_sha256 differs: with-sidecar=%s without-sidecar=%s", out1.ManifestSHA256, out2.ManifestSHA256)
	}
}

// AC-H07: write_sidecar:false => no file is written; response still carries
// manifest + manifest_sha256 with a null manifest_path.
func TestManifestWriteSidecarFalseDryRun(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", WriteSidecar: boolPtr(false)}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	if out.ManifestPath != nil {
		t.Errorf("expected nil ManifestPath in dry-run, got %v", *out.ManifestPath)
	}
	if out.Manifest == nil || out.ManifestSHA256 == "" {
		t.Errorf("dry-run response missing content: %+v", out)
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files written in dry-run, found %d entries", len(entries))
	}
}

// AC-H08: write_sidecar omitted (default true) => externalized-<ts>.json
// exists under out_dir (default .eidolons/.context).
func TestManifestSidecarDefaultWrite(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", TS: "20260101T000000Z"}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	if out.ManifestPath == nil {
		t.Fatal("expected non-nil ManifestPath")
	}
	want := filepath.Join(DefaultOutDir, "externalized-20260101T000000Z.json")
	if *out.ManifestPath != want {
		t.Errorf("ManifestPath = %s, want %s", *out.ManifestPath, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("manifest file not written: %v", err)
	}
}

// AC-H17: the only filesystem write is the single manifest file.
func TestManifestWritesOnlyOneFile(t *testing.T) {
	dir := t.TempDir()
	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", TS: "20260101T000000Z", OutDir: dir}
	if _, err := ExternalizeManifest(in); err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("expected exactly 1 file, got %d: %v", len(entries), names)
	}
	if entries[0].Name() != "externalized-20260101T000000Z.json" {
		t.Errorf("unexpected file written: %s", entries[0].Name())
	}
}

// AC-H24 (M0): the returned manifest object is decoded FROM the hashed
// ManifestBytes and deep-equals them — never built by a second,
// independent serialization path.
func TestManifestResponseObjectDecodedFromHashedBytes(t *testing.T) {
	in := ManifestInput{
		CreatedAt:       "2026-01-01T00:00:00Z",
		Anchors:         []string{"a:1", "b:2"},
		FileFloorReason: "crystalium absent",
		WriteSidecar:    boolPtr(false),
	}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	var independentlyDecoded map[string]any
	if err := json.Unmarshal(out.ManifestBytes, &independentlyDecoded); err != nil {
		t.Fatalf("unmarshal ManifestBytes: %v", err)
	}
	if !reflect.DeepEqual(out.Manifest, independentlyDecoded) {
		t.Errorf("out.Manifest != independently-decoded ManifestBytes:\n got=%v\nwant=%v", out.Manifest, independentlyDecoded)
	}
}

// AC-H25: a CRLF sequence splits ONLY on the LF, leaving the CR on the tail
// of the preceding element (kernel-confirmed: --anchor $'a\r\nb' emits
// ["a\r","b"]).
func TestManifestCROnlySplitsOnLF(t *testing.T) {
	in := ManifestInput{
		CreatedAt:    "2026-01-01T00:00:00Z",
		Anchors:      []string{"a\r\nb"},
		WriteSidecar: boolPtr(false),
	}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	anchors, _ := out.Manifest["anchors"].([]any)
	want := []any{"a\r", "b"}
	if !reflect.DeepEqual(anchors, want) {
		t.Errorf("anchors = %v, want %v", anchors, want)
	}
}

// AC-H26: a caller-supplied out_dir other than the default is honored.
func TestManifestCustomOutDir(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom-out")
	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", TS: "20260101T000000Z", OutDir: custom}
	out, err := ExternalizeManifest(in)
	if err != nil {
		t.Fatalf("ExternalizeManifest: %v", err)
	}
	want := filepath.Join(custom, "externalized-20260101T000000Z.json")
	if out.ManifestPath == nil || *out.ManifestPath != want {
		t.Errorf("ManifestPath = %v, want %s", out.ManifestPath, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("manifest file not written under custom out_dir: %v", err)
	}
}

// AC-H27: write_sidecar true and an out_dir whose file cannot be written
// returns a tool error rather than a success carrying a null path.
// MkdirAll is fail-soft; the WriteFile is the sole hard-error path.
func TestManifestSidecarWriteErrorSurfaces(t *testing.T) {
	dir := t.TempDir()
	// Create a REGULAR FILE at the path the manifest dir needs to occupy,
	// so MkdirAll fails (silently, fail-soft) and the subsequent WriteFile
	// fails loudly (ENOTDIR) — the sole hard-error path.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(blocker, "nested")

	in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", TS: "20260101T000000Z", OutDir: outDir}
	_, err := ExternalizeManifest(in)
	if err == nil {
		t.Fatal("expected an error when the sidecar cannot be written, got nil")
	}
}

// AC-H28: file_floor_reason is free-form, never a closed enum — the
// kernel's SECOND literal (context_externalize.sh:199) passes through
// verbatim, and so does an arbitrary third string.
func TestFileFloorReasonIsFreeForm(t *testing.T) {
	cases := []string{
		"crystalium absent",
		"crystalium commit unreachable or timed out (1.5s budget)",
		"a caller-supplied reason nothing in atomos ever hardcodes",
	}
	for _, reason := range cases {
		in := ManifestInput{CreatedAt: "2026-01-01T00:00:00Z", FileFloorReason: reason, WriteSidecar: boolPtr(false)}
		out, err := ExternalizeManifest(in)
		if err != nil {
			t.Fatalf("ExternalizeManifest(%q): %v", reason, err)
		}
		if out.Manifest["file_floor_reason"] != reason {
			t.Errorf("file_floor_reason = %v, want %q", out.Manifest["file_floor_reason"], reason)
		}
	}
}
