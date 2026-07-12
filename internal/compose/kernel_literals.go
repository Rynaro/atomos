// Package compose (continued): kernel_literals.go holds VERBATIM prose
// quoted from the bash kernel oracle — nothing else. This file exists
// because one such quotation legitimately contains a deny-listed English
// word (context_externalize.sh:100's default summary sentence contains
// "policy", as in "ECM P1 policy operation" — ordinary prose, not a
// capability atomos possesses). TestFenceNoForbiddenSurface
// (internal/tools/registry_test.go) greps non-test Go source for that
// token because the fence is worried about CODE that evaluates a
// decision table or reads a policy configuration — not about a string
// literal that echoes the kernel's own wording so atomos can byte-match
// it (AC-H09).
//
// This file is allowlisted BY NAME in
// internal/tools/registry_test.go's allowlistedFiles (keyed on
// filepath.Base, matching the v0.1 mechanism, not a new one) — the
// allowlist is the sanctioned escape hatch v0.1's Risk R4 named for
// exactly this class of false positive ("the word 'zone' in a comment").
// It is deliberately narrow: ONLY this file is exempt, never
// manifest.go or externalize.go, which carry logic and stay under full
// fence strength.
//
// The exemption is guarded, not just declared: kernel_literals_test.go's
// TestKernelLiteralsAreConstOnly parses this file with go/ast and fails
// the build if it contains anything but `const` string declarations — no
// funcs, no vars, no types, no imports, ever. An allowlisted file with a
// structural guard on its shape is a fence; an allowlisted file without
// one is a hole. That guard is what makes it safe to keep the true kernel
// bytes visible in the source rather than obscuring them from grep.
package compose

// ManifestDefaultSummary mirrors context_externalize.sh:100's default
// sentence verbatim (AC-H09) — the exact bytes the kernel emits when the
// caller supplies no summary.
const ManifestDefaultSummary = "Eidolons context externalize checkpoint: identifier manifest recorded while cheap (ECM P1 policy operation)."
