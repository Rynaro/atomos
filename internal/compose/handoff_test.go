package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

// AC-B04: from.version equals the caller's from_version input verbatim.
func TestFromVersionEchoesCallerInput(t *testing.T) {
	in := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", FromVersion: "7.7.7-custom", WriteSidecar: boolPtr(false)}
	out, err := Handoff(in)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	from, _ := out.Envelope["from"].(map[string]any)
	if from["version"] != "7.7.7-custom" {
		t.Errorf("from.version = %v, want 7.7.7-custom", from["version"])
	}
}

// AC-B05: from_version omitted => "n/a", NEVER the atomos build version.
func TestFromVersionNeverAtomosVersion(t *testing.T) {
	in := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", WriteSidecar: boolPtr(false)}
	out, err := Handoff(in)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	from, _ := out.Envelope["from"].(map[string]any)
	if from["version"] != "n/a" {
		t.Errorf("from.version = %v, want n/a", from["version"])
	}
	if from["version"] == "0.1.0" {
		t.Errorf("from.version leaked ATOMOS_VERSION")
	}
}

// AC-B06: no task_state => Narrative reads the exact default sentence.
func TestTaskStateDefault(t *testing.T) {
	in := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", WriteSidecar: boolPtr(false)}
	out, err := Handoff(in)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	want := "Task state: (no task-state summary provided)\n"
	if !strings.Contains(out.BriefMD, want) {
		t.Errorf("brief missing default task-state line; brief=%q", out.BriefMD)
	}
}

// AC-B07: thread_id resolves session_id -> handoff-<ts> chain when omitted.
func TestThreadIDDefaultChain(t *testing.T) {
	// No thread_id, no session_id -> handoff-<ts>.
	in := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", WriteSidecar: boolPtr(false)}
	out, err := Handoff(in)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if got, want := out.Envelope["thread_id"], "handoff-20260101T000000Z"; got != want {
		t.Errorf("thread_id = %v, want %v", got, want)
	}

	// No thread_id, session_id given -> session_id wins.
	in2 := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", SessionID: "sess-42", WriteSidecar: boolPtr(false)}
	out2, err := Handoff(in2)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if got, want := out2.Envelope["thread_id"], "sess-42"; got != want {
		t.Errorf("thread_id = %v, want %v", got, want)
	}

	// Explicit thread_id wins over session_id.
	in3 := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", SessionID: "sess-42", ThreadID: "explicit-thread", WriteSidecar: boolPtr(false)}
	out3, err := Handoff(in3)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if got, want := out3.Envelope["thread_id"], "explicit-thread"; got != want {
		t.Errorf("thread_id = %v, want %v", got, want)
	}
}

// AC-B08: write_sidecar true => the brief file bytes equal brief_md exactly
// (printf-%s semantics, no extra trailing newline appended).
func TestSidecarBriefBytesExact(t *testing.T) {
	dir := t.TempDir()
	in := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", TaskState: "x", OutDir: dir}
	out, err := Handoff(in)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if out.BriefPath == nil {
		t.Fatal("BriefPath is nil")
	}
	data, err := os.ReadFile(*out.BriefPath)
	if err != nil {
		t.Fatalf("read brief file: %v", err)
	}
	if string(data) != out.BriefMD {
		t.Errorf("brief file bytes != brief_md;\nfile=%q\nresp=%q", data, out.BriefMD)
	}
}

// AC-B09: write_sidecar omitted (default true) => the handoff-<ts>.md +
// handoff-<ts>.envelope.json pair exists under out_dir (default
// .eidolons/.context).
func TestSidecarDefaultWrite(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	in := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z"}
	out, err := Handoff(in)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if out.BriefPath == nil || out.EnvelopePath == nil {
		t.Fatalf("expected non-nil sidecar paths, got brief=%v envelope=%v", out.BriefPath, out.EnvelopePath)
	}
	wantBrief := filepath.Join(DefaultOutDir, "handoff-20260101T000000Z.md")
	wantEnv := filepath.Join(DefaultOutDir, "handoff-20260101T000000Z.envelope.json")
	if *out.BriefPath != wantBrief {
		t.Errorf("BriefPath = %s, want %s", *out.BriefPath, wantBrief)
	}
	if *out.EnvelopePath != wantEnv {
		t.Errorf("EnvelopePath = %s, want %s", *out.EnvelopePath, wantEnv)
	}
	if _, err := os.Stat(wantBrief); err != nil {
		t.Errorf("brief file not written: %v", err)
	}
	if _, err := os.Stat(wantEnv); err != nil {
		t.Errorf("envelope file not written: %v", err)
	}
}

// AC-B10: write_sidecar:false => no file is written; response still carries
// brief_md/brief_sha256/envelope with null paths.
func TestWriteSidecarFalseDryRun(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	in := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", WriteSidecar: boolPtr(false)}
	out, err := Handoff(in)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if out.BriefPath != nil || out.EnvelopePath != nil {
		t.Errorf("expected nil paths in dry-run, got brief=%v envelope=%v", out.BriefPath, out.EnvelopePath)
	}
	if out.BriefMD == "" || out.BriefSHA256 == "" || out.Envelope == nil {
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

// AC-B11: brief estimate exceeds the 1500-token advisory target =>
// oversize:true with brief_md untruncated.
func TestOversizeAdvisoryNeverTruncates(t *testing.T) {
	// 1500 tokens * 4 bytes/token = 6000 bytes threshold; force it well over.
	huge := strings.Repeat("x", 10000)
	in := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", Narrative: huge, WriteSidecar: boolPtr(false)}
	out, err := Handoff(in)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	if !out.Oversize {
		t.Errorf("expected Oversize=true, tokens_est=%d", out.TokensEst)
	}
	if !strings.Contains(out.BriefMD, huge) {
		t.Errorf("brief was truncated despite oversize advisory")
	}
}

// AC-B12: objective equals the fixed prefix plus the first non-blank line of
// a multiline task_state.
func TestObjectiveTaskStateHead(t *testing.T) {
	in := HandoffInput{
		TS:           "20260101T000000Z",
		ISOTS:        "2026-01-01T00:00:00Z",
		TaskState:    "\n   \nFirst real line here\nSecond line ignored",
		WriteSidecar: boolPtr(false),
	}
	out, err := Handoff(in)
	if err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	want := "Session handoff brief for context-lifecycle succession (ECM P1): First real line here"
	if out.Envelope["objective"] != want {
		t.Errorf("objective = %v, want %v", out.Envelope["objective"], want)
	}
}

// AC-B14: the only filesystem writes are the brief+envelope pair.
func TestComposeWritesOnlySidecarPair(t *testing.T) {
	dir := t.TempDir()
	in := HandoffInput{TS: "20260101T000000Z", ISOTS: "2026-01-01T00:00:00Z", OutDir: dir}
	if _, err := Handoff(in); err != nil {
		t.Fatalf("Handoff: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("expected exactly 2 files, got %d: %v", len(entries), names)
	}
	for _, e := range entries {
		if e.Name() != "handoff-20260101T000000Z.md" && e.Name() != "handoff-20260101T000000Z.envelope.json" {
			t.Errorf("unexpected file written: %s", e.Name())
		}
	}
}

func TestTaskStateHeadFallback(t *testing.T) {
	if got, want := TaskStateHead("   \n\t\n  "), "   \n\t\n  "; got != want {
		t.Errorf("TaskStateHead(all-blank) = %q, want %q (raw fallback)", got, want)
	}
	if got, want := TaskStateHead("\nreal line\nnext"), "real line"; got != want {
		t.Errorf("TaskStateHead = %q, want %q", got, want)
	}
}

func TestBuildBriefIdentifiersNoFallback(t *testing.T) {
	brief := BuildBrief("t", "", nil, nil, nil, nil, nil, nil, false)
	if !strings.Contains(brief, "## Identifiers\n\n## Failed approaches\n") {
		t.Errorf("Identifiers section should be empty with no fallback text; brief=%q", brief)
	}
}

// A non-empty array containing only empty-string entries still takes the
// "array has entries" branch (bash `[ "${#ARR[@]}" -gt 0 ]` is a COUNT
// check, not a "has any non-empty element" check) — so it renders NEITHER
// the "(none recorded)" fallback NOR any bullets: an empty section body.
func TestBuildBriefEmptyStringEntriesSkipped(t *testing.T) {
	brief := BuildBrief("t", "", []string{""}, nil, nil, []string{"", ""}, nil, []string{""}, false)
	if strings.Contains(brief, "(none recorded)") {
		t.Errorf("a non-empty array of blank entries should NOT fall back to (none recorded): %q", brief)
	}
	if !strings.Contains(brief, "## Failed approaches\n\n## Next steps\n") {
		t.Errorf("failed-approaches section should render empty (no fallback, no bullets): %q", brief)
	}
	if strings.Contains(brief, "- anchor: \n") {
		t.Errorf("empty anchor entry should be skipped: %q", brief)
	}
}
