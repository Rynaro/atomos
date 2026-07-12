// Package scripts holds Go-test-only coverage for scripts/regen-goldens.sh.
//
// There is no shell test harness in this repo (see CHANGELOG "Fixed" entry
// for the provenance-fail-loud fix this file backstops). These tests shell
// out to the REAL script against a scratch copy of the repo layout (never
// the real committed fixtures/) with one precondition deliberately broken
// at a time, and assert:
//
//   - the hard-fail guards (ecm_version, nexus_commit) exit non-zero, name
//     the input they couldn't resolve, and leave a pre-existing PROVENANCE
//     file byte-for-byte untouched (no half-written stamp on the failure
//     path);
//   - the warn-only guards (dirty oracle, unpushed commit) print their
//     warning when the precondition is violated and stay silent when it
//     isn't — proving the guard is not vacuous in either direction.
//
// _test.go is exempt from internal/tools' TestFenceNoForbiddenSurface (it
// only walks non-test .go files), so this file needs no allowlist entry.
package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const goodProvenance = "ECM_VERSION=0.1\n" +
	"NEXUS_COMMIT=deadbeefdeadbeefdeadbeefdeadbeefdeadbeef\n" +
	"CAPTURED_VIA=test-fixture, not a real regen run\n" +
	"VECTORS=x\n" +
	"MANIFEST_VECTORS=y\n" +
	"NOTE=test fixture sentinel value — a refusal path must never alter this.\n"

// newScratchRepo builds a minimal scratch tree — <tmp>/scripts/regen-goldens.sh
// (the REAL script, copied verbatim) plus fixtures/parity/PROVENANCE (a
// pre-existing "good" stamp) — so the script's own REPO_ROOT/HANDOFF_DIR
// resolution (relative to its own on-disk location) lands entirely inside
// the scratch tree, never anywhere near the real committed fixtures/.
func newScratchRepo(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()
	scriptsDir := filepath.Join(repoDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	src, err := os.ReadFile("regen-goldens.sh")
	if err != nil {
		t.Fatalf("read real script: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "regen-goldens.sh"), src, 0o755); err != nil {
		t.Fatalf("write scratch script: %v", err)
	}
	parityDir := filepath.Join(repoDir, "fixtures", "parity")
	if err := os.MkdirAll(parityDir, 0o755); err != nil {
		t.Fatalf("mkdir fixtures/parity: %v", err)
	}
	if err := os.WriteFile(filepath.Join(parityDir, "PROVENANCE"), []byte(goodProvenance), 0o644); err != nil {
		t.Fatalf("write scratch PROVENANCE: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoDir, "fixtures", "parity-manifest"), 0o755); err != nil {
		t.Fatalf("mkdir fixtures/parity-manifest: %v", err)
	}
	return repoDir
}

// newFakeNexus builds a minimal fake EIDOLONS_NEXUS: a present (not
// necessarily executable-for-real) cli/eidolons entrypoint file plus a
// roster/context-policy.yaml. withEcmVersion controls whether the yaml
// carries a resolvable 'ecm_version:' key. gitInit controls whether the
// directory is a real git repo (so `git rev-parse HEAD` can succeed).
func newFakeNexus(t *testing.T, withEcmVersion bool, gitInit bool) string {
	t.Helper()
	nexus := t.TempDir()
	if err := os.MkdirAll(filepath.Join(nexus, "cli"), 0o755); err != nil {
		t.Fatalf("mkdir cli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nexus, "cli", "eidolons"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatalf("write cli/eidolons: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(nexus, "roster"), 0o755); err != nil {
		t.Fatalf("mkdir roster: %v", err)
	}
	policy := "other_key: \"x\"\n"
	if withEcmVersion {
		policy = "ecm_version: \"0.1\"\nother_key: \"x\"\n"
	}
	if err := os.WriteFile(filepath.Join(nexus, "roster", "context-policy.yaml"), []byte(policy), 0o644); err != nil {
		t.Fatalf("write roster/context-policy.yaml: %v", err)
	}
	if gitInit {
		runGit(t, nexus, "init", "-q")
		runGit(t, nexus, "config", "user.email", "test@test.invalid")
		runGit(t, nexus, "config", "user.name", "test")
		runGit(t, nexus, "add", "-A")
		runGit(t, nexus, "commit", "-q", "-m", "init")
	}
	return nexus
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// runScript executes the scratch copy of regen-goldens.sh with
// EIDOLONS_NEXUS pointed at nexusDir, returning its combined output and
// exit error (nil on success).
func runScript(repoDir, nexusDir string) (combinedOutput string, err error) {
	cmd := exec.Command("bash", filepath.Join(repoDir, "scripts", "regen-goldens.sh"))
	cmd.Env = append(os.Environ(), "EIDOLONS_NEXUS="+nexusDir)
	out, runErr := cmd.CombinedOutput()
	return string(out), runErr
}

func readProvenance(t *testing.T, repoDir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoDir, "fixtures", "parity", "PROVENANCE"))
	if err != nil {
		t.Fatalf("read scratch PROVENANCE: %v", err)
	}
	return string(b)
}

// TestRegenGoldensRefusesUnresolvableEcmVersion proves the ecm_version
// hard-fail guard actually fires: a nexus roster/context-policy.yaml with
// no 'ecm_version:' key must make the script exit non-zero, name the
// input it couldn't resolve, and leave a pre-existing PROVENANCE
// untouched (never clobbered with a poisoned "unknown").
func TestRegenGoldensRefusesUnresolvableEcmVersion(t *testing.T) {
	repo := newScratchRepo(t)
	nexus := newFakeNexus(t, false /* no ecm_version key */, true /* git init'd */)

	out, err := runScript(repo, nexus)
	if err == nil {
		t.Fatalf("expected non-zero exit, got success. output:\n%s", out)
	}
	if !strings.Contains(out, "ecm_version") {
		t.Errorf("expected error output to name ecm_version, got:\n%s", out)
	}
	if !strings.Contains(out, "goldens were touched") {
		t.Errorf("expected error output to state goldens were touched (as in: none were), got:\n%s", out)
	}
	if got := readProvenance(t, repo); got != goodProvenance {
		t.Errorf("PROVENANCE was modified on the refusal path:\ngot=%q\nwant=%q", got, goodProvenance)
	}
}

// TestRegenGoldensRefusesUnresolvableNexusCommit proves the nexus_commit
// hard-fail guard: a nexus directory that isn't a git repository at all
// (git rev-parse HEAD fails, the same failure mode a read-only bind mount
// tripping git's ownership check produces) must make the script refuse
// the same way — exit non-zero and leave PROVENANCE untouched.
func TestRegenGoldensRefusesUnresolvableNexusCommit(t *testing.T) {
	repo := newScratchRepo(t)
	nexus := newFakeNexus(t, true /* ecm_version present */, false /* NOT a git repo */)

	out, err := runScript(repo, nexus)
	if err == nil {
		t.Fatalf("expected non-zero exit, got success. output:\n%s", out)
	}
	if !strings.Contains(out, "rev-parse HEAD") {
		t.Errorf("expected error output to mention resolving HEAD, got:\n%s", out)
	}
	if !strings.Contains(out, "goldens were touched") {
		t.Errorf("expected error output to state goldens were touched (as in: none were), got:\n%s", out)
	}
	if got := readProvenance(t, repo); got != goodProvenance {
		t.Errorf("PROVENANCE was modified on the refusal path:\ngot=%q\nwant=%q", got, goodProvenance)
	}
}

// TestRegenGoldensWarnsOnDirtyOracle proves the dirty-oracle guard fires
// exactly when cli/ has uncommitted changes, and stays silent when it
// doesn't — a contrast pair, so the guard is proven non-vacuous in both
// directions rather than merely "sometimes prints something".
func TestRegenGoldensWarnsOnDirtyOracle(t *testing.T) {
	repo := newScratchRepo(t)
	nexus := newFakeNexus(t, true, true)

	t.Run("clean", func(t *testing.T) {
		out, _ := runScript(repo, nexus)
		if strings.Contains(out, "UNCOMMITTED changes under cli/") {
			t.Errorf("did not expect a dirty-oracle warning for a clean nexus, got:\n%s", out)
		}
	})

	t.Run("dirty", func(t *testing.T) {
		// Dirty a real oracle-closure file under cli/ without committing.
		f := filepath.Join(nexus, "cli", "eidolons")
		orig, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if err := os.WriteFile(f, append(orig, []byte("\n# dirty-marker\n")...), 0o755); err != nil {
			t.Fatalf("dirty %s: %v", f, err)
		}
		defer func() {
			if err := os.WriteFile(f, orig, 0o755); err != nil {
				t.Fatalf("restore %s: %v", f, err)
			}
		}()

		out, _ := runScript(repo, nexus)
		if !strings.Contains(out, "UNCOMMITTED changes under cli/") {
			t.Errorf("expected a dirty-oracle warning, got:\n%s", out)
		}
		if !strings.Contains(out, "cli/eidolons") {
			t.Errorf("expected the warning to name the dirty path cli/eidolons, got:\n%s", out)
		}
	})
}

// TestRegenGoldensWarnsOnUnpushedCommit proves the unpushed-commit guard
// fires exactly when HEAD is unreachable from any remote-tracking branch
// (a commit-only-local, never-pushed scenario), and stays silent once a
// remote-tracking branch actually contains it — again a contrast pair.
func TestRegenGoldensWarnsOnUnpushedCommit(t *testing.T) {
	repo := newScratchRepo(t)
	nexus := newFakeNexus(t, true, true)

	bare := t.TempDir()
	runGit(t, bare, "init", "-q", "--bare")
	runGit(t, nexus, "remote", "add", "testremote", bare)

	t.Run("unpushed", func(t *testing.T) {
		out, _ := runScript(repo, nexus)
		if !strings.Contains(out, "not reachable from") {
			t.Errorf("expected an unpushed-commit warning, got:\n%s", out)
		}
	})

	t.Run("pushed_and_fetched", func(t *testing.T) {
		runGit(t, nexus, "push", "-q", "testremote", "HEAD:refs/heads/main")
		runGit(t, nexus, "fetch", "-q", "testremote")

		out, _ := runScript(repo, nexus)
		if strings.Contains(out, "not reachable from") {
			t.Errorf("did not expect an unpushed-commit warning once pushed+fetched, got:\n%s", out)
		}
	})
}

// TestRegenGoldensRefusalNeverLeavesTempFile proves the atomic
// temp-file-then-rename write pattern: even after a refusal, no
// PROVENANCE.tmp is left behind in the scratch fixtures/parity directory.
func TestRegenGoldensRefusalNeverLeavesTempFile(t *testing.T) {
	repo := newScratchRepo(t)
	nexus := newFakeNexus(t, false, true)

	if _, err := runScript(repo, nexus); err == nil {
		t.Fatalf("expected non-zero exit")
	}
	if _, err := os.Stat(filepath.Join(repo, "fixtures", "parity", "PROVENANCE.tmp")); !os.IsNotExist(err) {
		t.Errorf("expected no PROVENANCE.tmp to be left behind, stat err=%v", err)
	}
}
