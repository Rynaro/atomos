# Changelog

All notable changes to atomos are documented here.

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
