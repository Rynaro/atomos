# Changelog

All notable changes to atomos are documented here.

## [Unreleased]

### Fixed

- `scripts/regen-goldens.sh`: the PROVENANCE stamp's two inputs
  (`ECM_VERSION` from the nexus's `roster/context-policy.yaml`,
  `NEXUS_COMMIT` from `git rev-parse HEAD`) no longer degrade silently to
  the literal string `unknown` and overwrite a previously-good pin. Both
  are now resolved and validated UP FRONT — before jq is even required,
  before either regen arm runs, and before any golden or PROVENANCE itself
  is touched — and a regen that can't resolve one now REFUSES loudly
  (names the exact input, says what to do, exits non-zero) instead of
  writing a poisoned stamp. `NEXUS_COMMIT=unknown` used to be caught only
  LATE, in CI (`.github/workflows/conformance.yml`'s "has no usable
  NEXUS_COMMIT pin" check); `ECM_VERSION=unknown` had no downstream safety
  net at all. Reproduced the field incident exactly (a read-only
  bind-mounted nexus checkout trips git's dubious-ownership check) to
  confirm the new guard fires.
- `scripts/regen-goldens.sh`: two adjacent provenance-integrity gaps
  closed at the same capture point — (1) a dirty nexus working tree under
  `cli/` (the kernel's full source closure for both regen arms) now prints
  an unmissable warning AND stamps a durable `ORACLE_DIRTY_AT_CAPTURE` line
  into PROVENANCE itself (reviewable in the PR diff, not just a scrolling
  CI log) — warn rather than block, since previewing an in-flight nexus
  kernel change's golden impact before committing it is a legitimate
  workflow; (2) a nexus `HEAD` unreachable from any known remote-tracking
  branch (a commit that hasn't been pushed yet) now warns locally instead
  of failing confusingly in CI's `actions/checkout` step later.
- `scripts/regen-goldens.sh`: the PROVENANCE file is now written via a
  temp-file-then-rename, so no failure path can leave a half-written stamp
  in place of a good one.
- Added `scripts/regen_goldens_test.go`: shells out to the real script
  against a scratch repo copy with one precondition broken at a time
  (unresolvable `ecm_version`, unresolvable `git rev-parse HEAD`, a dirty
  oracle, an unpushed commit) and asserts the guard actually fires (or, for
  the warn-only guards, stays silent in the contrast case) and that a
  refusal never touches a pre-existing PROVENANCE file.

## [0.2.0] — 2026-07-12

Closes the ADR-declared tool set (RAMZA build spec `docs/BUILD-SPEC-v0.2.md`).

### Added

- `compose_externalize_manifest`: the 4th and final tool. Mirrors ONLY the
  manifest-build portion of `eidolons context externalize`
  (`context_externalize.sh:116-146`) — the identifier manifest (anchors,
  symbols, decisions, failed approaches, open variables,
  `contains_tool_origin`, session, `created_at`) plus its SHA-256 (M0: the
  digest covers exactly the canonical bytes returned and — when
  `write_sidecar` is true (the default) — written to the file-floor
  sidecar). Never reaches the kernel's durable-memory chain
  (`context_externalize.sh:154-223`) — permanently out of fence.
- `internal/jsonx`: the jq-exact ordered-emitter primitives (string escaper,
  object writer, and a new array writer) extracted from `internal/ecl` so
  the manifest composer shares one escaper with the envelope composer
  instead of growing a second, driftable one.
- Golden parity fixtures for the manifest
  (`fixtures/parity-manifest/{manifest-defaults,manifest-populated,
  manifest-tool-origin}`), captured from the kernel's file-floor write —
  both the 11-key form (caller-supplied `file_floor_reason`, byte-identical
  to a kernel-written file) and the jq-derived 10-key `core.json` form (the
  default path most callers get, matching the kernel's in-memory manifest
  object).
- Four new handoff parity vectors (`empty-sections`, `oversize-brief`,
  `multiline-task-state`, `empty-list-entries`) pinning kernel edge-case
  behavior, each with a captured `advisory.json` (`tokens_est`/`oversize`)
  — now present for all seven handoff vectors.
- `scripts/regen-goldens.sh`: a manifest-capture arm (driven through the
  kernel's `file_floor_path` side effect), index-driven array-flag
  extraction (fixes a live v0.1.0 defect where newline-bearing or
  empty-string list entries were mangled before ever reaching the kernel),
  a seeded `FLAGS=(--json)` array (fixes a live v0.1.0 bash-3.2
  `unbound variable` abort on an empty flag array), and a fixture-authoring
  guard that refuses a vector whose list element ends in a newline.
- CI: the drift-guard job now runs on `[ubuntu-latest, macos-latest]` (so
  the bash-3.2 empty-array fix is exercised on a real bash 3.2, not merely
  linted) and checks `git status --porcelain --untracked-files=all` over
  both fixture trees (catches an untracked golden the executor forgot to
  `git add` — blind to `git diff --exit-code`).
- `internal/tools/registry.go`: the tool surface is now closed at **four**;
  `TestToolSurfaceIsExactlyFenced` is tightened to exact equality
  (never a superset/"at least" assertion).
- `internal/compose/kernel_literals.go`: a narrowly-scoped, structurally
  guarded fence exemption. The kernel's verbatim default-summary sentence
  (`context_externalize.sh:100`) contains the English word "policy" as
  ordinary prose, which collides with `TestFenceNoForbiddenSurface`'s
  deny-list. Rather than obscuring the token from the grep (a first-draft
  string-concatenation trick was reviewed and rejected — it hid the byte
  match instead of exempting it, which is exactly the "source lies about
  its own contents" failure mode the fence exists to catch), the constant
  now lives in its own file, allowlisted by name
  (`internal/tools/registry_test.go`, the same mechanism v0.1's Risk R4
  already named for this class of false positive), with a new structural
  guard (`kernel_literals_test.go: TestKernelLiteralsAreConstOnly`, `go/ast`)
  that fails the build if that file is ever anything but bare `const`
  string declarations. `manifest.go` and `externalize.go` remain fully
  fenced — only the one file holding zero logic is exempt.

### Fixed

- `scripts/regen-goldens.sh` no longer mangles newline-bearing or
  empty-string list entries before they reach the kernel oracle (a
  `read`-based loop previously split a multiline entry into multiple CLI
  flags and dropped empty entries via a non-empty guard) — a latent defect
  since v0.1.0 that only affected the not-yet-existent edge-case vectors.
- `scripts/regen-goldens.sh` no longer aborts under bash 3.2 (macOS system
  bash) when a vector's flag array is empty (`set -u` + an empty
  `"${FLAGS[@]}"` expansion) — present since v0.1.0's `defaults-only`
  vector, never caught because CI's drift-guard ran on `ubuntu-latest`
  (bash 5) only.

## [0.1.0] — 2026-07-07

Initial MVP release — the compose/verify executor MCP for ECM (FORGE ADR,
settled 2026-07-07; RAMZA build spec `docs/BUILD-SPEC.md`).

### Added

- `cmd/atomos`: stdio MCP entrypoint (`atomos serve` / `atomos --version`).
- `compose_handoff`: session-handoff brief + ECL `INFORM` envelope
  (`ecm/handoff-brief@0.1`), **T1 byte/SHA parity** with
  `eidolons context handoff` and **T2 envelope byte parity** via a hand-rolled
  ordered JSON emitter (`internal/ecl`).
- `verify_envelope`: full kernel verdict matrix (`pass` / `tamper` /
  `inconsistent` / `unverifiable` / `missing_payload` / `unsupported_algo` /
  `malformed`), including the "1.0"-vs-"2.0" envelope-version warn.
- `verify_pins`: ECM spec §3.2 pin-survival probe (literal id-token or
  regex marker).
- Golden parity fixtures (`fixtures/parity/{defaults-only,fully-populated,
  narrative-open-vars}`) captured from the `Rynaro/eidolons` kernel, plus
  `scripts/regen-goldens.sh` (bash 3.2 compatible) and a CI drift-guard.
- Verdict-matrix fixtures (`fixtures/conformant/`, `fixtures/failing/`)
  covering the full `verify_envelope` outcome space.
- Fence suite (`internal/tools`): `TestToolSurfaceIsExactlyFenced`,
  `TestFenceNoForbiddenSurface`, `TestNoTimeNowInHandlerPackages`.
- Distroless multi-stage `Dockerfile`; `conformance.yml` (vet + test + gofmt
  + multi-arch build check + drift-guard) and `release.yml` (buildx
  `linux/amd64`+`linux/arm64` → `ghcr.io/rynaro/atomos`, index-digest capture)
  GitHub Actions workflows.

### Not in this release

- `compose_externalize_manifest` (deferred to v0.2, ADR §2.4).
- Nexus rostering (`roster/mcps.yaml`, `atomos.mcp.json.tmpl`,
  `eidolons.mcp.lock`) — Phase 3, a separate ESL change in the nexus repo.
