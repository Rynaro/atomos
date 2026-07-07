package tools_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Rynaro/atomos/internal/tools"
)

// AC-F01: the tool registry advertises exactly {compose_handoff,
// verify_envelope, verify_pins} — no more, no less.
func TestToolSurfaceIsExactlyFenced(t *testing.T) {
	want := []string{"compose_handoff", "verify_envelope", "verify_pins"}
	got := append([]string(nil), tools.Names()...)
	sort.Strings(got)
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	if len(got) != len(sortedWant) {
		t.Fatalf("Registry has %d tools, want %d: got=%v", len(got), len(sortedWant), got)
	}
	for i := range got {
		if got[i] != sortedWant[i] {
			t.Fatalf("Registry = %v, want %v", got, sortedWant)
		}
	}
}

// forbidden pairs a deny-list identifier with a word-boundary regexp so
// substrings inside unrelated words (or this very test/allowlist) don't
// false-positive (AC-F02; ADR §4 point 2).
var forbidden = []struct {
	label string
	re    *regexp.Regexp
}{
	{"meter", regexp.MustCompile(`\bmeter\b`)},
	{"utilization", regexp.MustCompile(`\butilization\b`)},
	{"zone", regexp.MustCompile(`\bzone\b`)},
	{"policy", regexp.MustCompile(`\bpolicy\b`)},
	{"decision-table/verdict-eval", regexp.MustCompile(`\bdecision_table\b|\bdecisionTable\b`)},
	{"compact-trigger", regexp.MustCompile(`\bcompact\(`)},
	{"handoff-fresh-trigger", regexp.MustCompile(`\bhandoff[-_]fresh\b`)},
	{"claude-dir", regexp.MustCompile(`\.claude/`)},
	{"mcp-json", regexp.MustCompile(`\.mcp\.json`)},
	{"additionalContext", regexp.MustCompile(`\badditionalContext\b`)},
	{"crystalium-client", regexp.MustCompile(`\bcrystalium\b`)},
	{"ingest", regexp.MustCompile(`\bingest\b`)},
	{"commit-verb", regexp.MustCompile(`\bcrystalium_commit\b|\bcommit_memory\b`)},
	{"persist", regexp.MustCompile(`\bpersist\b`)},
	{"recall", regexp.MustCompile(`\brecall\b`)},
}

// allowlistedFiles are permitted to mention deny-listed identifiers because
// they ARE the fence definition, its own test, or documentation ABOUT the
// fence — never executable tool-surface code.
var allowlistedFiles = map[string]bool{
	"registry_test.go": true,
}

// AC-F02: non-test, non-fixture Go source contains zero forbidden-surface
// identifiers.
func TestFenceNoForbiddenSurface(t *testing.T) {
	root := filepath.Join("..", "..")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "fixtures", "docs", ".github":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if allowlistedFiles[filepath.Base(path)] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		for _, f := range forbidden {
			if f.re.MatchString(content) {
				t.Errorf("%s: forbidden surface %q found (deny-listed — ADR §4 fence)", path, f.label)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

// AC-F04: no handler package calls time.Now — wall-clock enters only at the
// single server-layer seam.
func TestNoTimeNowInHandlerPackages(t *testing.T) {
	pkgs := []string{
		filepath.Join("..", "compose"),
		filepath.Join("..", "verify"),
		filepath.Join("..", "ecl"),
		filepath.Join("..", "hashing"),
	}
	timeNow := regexp.MustCompile(`\btime\.Now\(`)
	for _, dir := range pkgs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if timeNow.MatchString(string(data)) {
				t.Errorf("%s/%s: calls time.Now (forbidden outside the server-layer seam)", dir, e.Name())
			}
		}
	}
}
