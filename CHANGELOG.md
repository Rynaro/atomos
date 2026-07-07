# Changelog

All notable changes to atomos are documented here.

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
