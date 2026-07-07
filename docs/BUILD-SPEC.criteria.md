## Acceptance Criteria

The frozen copy of this section is `docs/BUILD-SPEC.criteria.md`; its SHA-256
is recorded in the Freeze Record and in `BUILD-SPEC.state.json`.

#### Track A — server scaffold

### AC-A01 (event-driven)
GIVEN the atomos binary started with `serve` on stdio
WHEN an MCP client sends `initialize`
THEN the server completes the MCP handshake identifying itself as `atomos` with the ATOMOS_VERSION version string.
VERIFY: `go test ./internal/server -run TestInitializeHandshake`

### AC-A02 (event-driven)
GIVEN an initialized stdio session
WHEN the client sends `tools/list`
THEN the response advertises exactly `compose_handoff`, `verify_envelope`, `verify_pins` — no other tool.
VERIFY: `go test ./internal/server -run TestToolsListMatchesRegistry`

### AC-A03 (event-driven)
GIVEN the built binary
WHEN invoked as `atomos --version`
THEN stdout is exactly `0.1.0` followed by a single newline with exit code 0.
VERIFY: `test "$(go run ./cmd/atomos --version)" = "0.1.0"`

### AC-A04 (unwanted-behavior)
GIVEN an initialized session
WHEN `tools/call` names a tool outside the closed set (e.g. `context_meter`)
THEN the server returns a tool-not-found error without executing anything.
VERIFY: `go test ./internal/server -run TestUnknownToolRejected`

#### Track B — compose_handoff

### AC-B01 (event-driven)
GIVEN a parity vector `input.json` with frozen `ts`/`iso_ts`
WHEN `compose_handoff` runs on it
THEN the returned `brief_md` is byte-identical to the kernel golden `fixtures/parity/<vector>/brief.md` (T1, mandatory).
VERIFY: `go test ./internal/compose -run TestParityBriefBytes`

### AC-B02 (event-driven)
GIVEN the same parity vector
WHEN `compose_handoff` runs on it
THEN `brief_sha256` equals the kernel-recorded golden `sha256` for that vector.
VERIFY: `go test ./internal/compose -run TestParityBriefSHA`

### AC-B03 (ubiquitous)
GIVEN any compose_handoff invocation
WHEN the envelope is emitted
THEN `envelope_version` is exactly the string `"1.0"` — the kernel drift mirrored, never "fixed" to `"2.0"`.
VERIFY: `go test ./internal/ecl -run TestEnvelopeVersionIsOneDotZero`

### AC-B04 (ubiquitous)
GIVEN a caller-supplied `from_version`
WHEN the envelope is emitted
THEN `from.version` equals the caller's `from_version` input verbatim.
VERIFY: `go test ./internal/compose -run TestFromVersionEchoesCallerInput`

### AC-B05 (unwanted-behavior)
GIVEN an input that omits `from_version`
WHEN the envelope is emitted
THEN `from.version` is `"n/a"` — never the ATOMOS_VERSION string.
VERIFY: `go test ./internal/compose -run TestFromVersionNeverAtomosVersion`

### AC-B06 (event-driven)
GIVEN input with no `task_state`
WHEN the brief is composed
THEN the Narrative section reads `Task state: (no task-state summary provided)`.
VERIFY: `go test ./internal/compose -run TestTaskStateDefault`

### AC-B07 (event-driven)
GIVEN input with no `thread_id`
WHEN the envelope is emitted
THEN `thread_id` resolves by the kernel chain — `session_id` when present, else `handoff-<ts>`.
VERIFY: `go test ./internal/compose -run TestThreadIDDefaultChain`

### AC-B08 (ubiquitous)
GIVEN `write_sidecar` true
WHEN the brief file is written
THEN the file bytes equal `brief_md` exactly (printf-`%s` semantics — no extra trailing newline appended).
VERIFY: `go test ./internal/compose -run TestSidecarBriefBytesExact`

### AC-B09 (event-driven)
GIVEN input without `write_sidecar` (default true)
WHEN compose completes
THEN the pair `handoff-<ts>.md` + `handoff-<ts>.envelope.json` exists under `out_dir` (default `.eidolons/.context`).
VERIFY: `go test ./internal/compose -run TestSidecarDefaultWrite`

### AC-B10 (optional-feature)
GIVEN `write_sidecar: false`
WHEN compose completes
THEN no file is written while the response still carries `brief_md`, `brief_sha256`, `envelope` with null paths.
VERIFY: `go test ./internal/compose -run TestWriteSidecarFalseDryRun`

### AC-B11 (unwanted-behavior)
GIVEN inputs whose brief estimate exceeds the 1500-token advisory target
WHEN compose completes
THEN the response sets `oversize: true` with `brief_md` untruncated.
VERIFY: `go test ./internal/compose -run TestOversizeAdvisoryNeverTruncates`

### AC-B12 (event-driven)
GIVEN a multiline `task_state`
WHEN the envelope is emitted
THEN `objective` equals `"Session handoff brief for context-lifecycle succession (ECM P1): "` plus the first non-blank line of `task_state`.
VERIFY: `go test ./internal/compose -run TestObjectiveTaskStateHead`

### AC-B13 (event-driven)
GIVEN a frozen-`ts` parity vector
WHEN the envelope is emitted
THEN it satisfies T2 against the golden `envelope.json` — byte-equal via the ordered emitter, or the documented semantic-equivalence fallback (field-equal + verify pass) recorded in `fixtures/README.md`.
VERIFY: `go test ./internal/ecl -run TestEnvelopeT2Parity`

### AC-B14 (ubiquitous)
GIVEN any compose_handoff call
WHEN it completes
THEN the only filesystem writes are the brief+envelope pair — no crystalium call, no other write (kernel ingest at `context_handoff.sh:242-250` is out of fence).
VERIFY: `go test ./internal/compose -run TestComposeWritesOnlySidecarPair`

#### Track C — verify_envelope

### AC-C01 (event-driven)
GIVEN every fixture under `fixtures/conformant/` and `fixtures/failing/`
WHEN `verify_envelope` runs on it
THEN the verdict equals the fixture's expected verdict across the full matrix (pass · tamper · inconsistent · unverifiable · missing_payload · unsupported_algo · malformed).
VERIFY: `go test ./internal/verify -run TestVerdictMatrix`

### AC-C02 (unwanted-behavior)
GIVEN a payload modified after envelope emission
WHEN verified
THEN the verdict is `tamper` carrying the envelope tag as `expected_sha` alongside the recomputed `actual_sha`.
VERIFY: `go test ./internal/verify -run TestTamperShaReporting`

### AC-C03 (event-driven)
GIVEN an envelope with `envelope_version` `"1.0"` (or any value other than `"2.0"`)
WHEN verified
THEN the result records an advisory warning naming the unrecognized version while the verdict is still computed normally.
VERIFY: `go test ./internal/verify -run TestEnvelopeVersionWarnNotFail`

### AC-C04 (event-driven)
GIVEN `integrity.value` matching a placeholder pattern (empty, `PARENT_FILLS_*`, leading `<`, `TODO*`, `null`)
WHEN verified
THEN the verdict is `unverifiable` — never a failure verdict, never blocked.
VERIFY: `go test ./internal/verify -run TestPlaceholderUnverifiable`

### AC-C05 (event-driven)
GIVEN an `artifact.path` that does not resolve while the sibling `<name minus .envelope.json>` exists
WHEN verified
THEN the sibling file is used as the payload for the SHA comparison.
VERIFY: `go test ./internal/verify -run TestSiblingPayloadFallback`

### AC-C06 (state-driven)
GIVEN a verify_envelope call
WHEN the verdict is one of tamper · inconsistent · missing_payload · unsupported_algo under `mode: block`
THEN the response sets `blocked: true` (warn mode, plus all other verdicts in any mode, report `blocked: false`).
VERIFY: `go test ./internal/verify -run TestBlockedFlagMatrix`

### AC-C07 (ubiquitous)
GIVEN any verify_envelope outcome including failures
WHEN the result is produced
THEN atomos returns a normal MCP tool result without process exit — exit-3 enforcement stays with the kernel/orchestrator (fail-open P0).
VERIFY: `go test ./internal/verify -run TestVerifyNeverExitsProcess`

### AC-C08 (unwanted-behavior)
GIVEN an envelope missing any required field (envelope_version, from.eidolon, to.eidolon, performative, artifact.path, integrity.method, integrity.value)
WHEN verified
THEN the verdict is `malformed` with the missing field names listed in `message`.
VERIFY: `go test ./internal/verify -run TestMalformedMissingFields`

#### Track D — verify_pins

### AC-D01 (event-driven)
GIVEN a pin set plus an artifact containing every pin marker
WHEN `verify_pins` runs
THEN the response reports `survived: true` with every pin `present: true`.
VERIFY: `go test ./internal/verify -run TestAllPinsSurvive`

### AC-D02 (unwanted-behavior)
GIVEN an artifact missing at least one pin marker
WHEN `verify_pins` runs
THEN the response reports `survived: false` with each absent pin id listed in `missing[]`.
VERIFY: `go test ./internal/verify -run TestMissingPinReported`

### AC-D03 (ubiquitous)
GIVEN a pin with `marker: null`
WHEN the probe runs
THEN the pin's id token itself is used as the literal search marker.
VERIFY: `go test ./internal/verify -run TestMarkerDefaultsToID`

### AC-D04 (event-driven)
GIVEN a pin with an explicit regex marker
WHEN the probe runs
THEN presence is decided by regex match against the artifact.
VERIFY: `go test ./internal/verify -run TestRegexMarkerMatch`

### AC-D05 (ubiquitous)
GIVEN any verify_pins call
WHEN it completes
THEN the result is purely advisory — no `blocked` field, no re-injection, no filesystem write.
VERIFY: `go test ./internal/verify -run TestPinsAdvisoryNoWrites`

#### Track E — parity suite

### AC-E01 (ubiquitous)
GIVEN the MVP fixture tree
WHEN the parity suite enumerates vectors
THEN at least three frozen-`ts` vectors exist (`defaults-only`, `fully-populated`, `narrative-open-vars`) each complete with input.json, brief.md, envelope.json, sha256.
VERIFY: `go test ./internal/compose -run TestParityVectorsComplete`

### AC-E02 (event-driven)
GIVEN a machine with the eidolons kernel checkout available
WHEN `scripts/regen-goldens.sh` runs against an unchanged kernel
THEN it re-derives every vector's goldens via `eidolons context handoff --json` leaving `git diff` clean on `fixtures/parity`.
VERIFY: `bash scripts/regen-goldens.sh && git diff --exit-code fixtures/parity`

### AC-E03 (state-driven)
GIVEN `conformance.yml` executing in CI
WHEN the drift-guard step runs
THEN it checks out the nexus at the PROVENANCE-pinned commit, regenerates the goldens, then fails the build on any diff.
VERIFY: `grep -q 'regen-goldens' .github/workflows/conformance.yml`

### AC-E04 (ubiquitous)
GIVEN the golden fixtures
WHEN provenance is inspected
THEN `fixtures/parity/PROVENANCE` records the ECM_VERSION plus the nexus commit SHA the goldens were captured from.
VERIFY: `go test ./internal/compose -run TestGoldenProvenanceStamp`

### AC-E05 (ubiquitous)
GIVEN `scripts/regen-goldens.sh`
WHEN linted
THEN it contains no bash-4+ constructs (no `declare -A`, no `${var,,}`, no readarray/mapfile, no `&>>`) so macOS bash 3.2 can run it.
VERIFY: `shellcheck -x -S error scripts/regen-goldens.sh && ! grep -nE 'declare -A|readarray|mapfile|&>>' scripts/regen-goldens.sh`

#### Track F — fence suite

### AC-F01 (ubiquitous)
GIVEN the tool registry
WHEN `TestToolSurfaceIsExactlyFenced` runs
THEN it asserts the advertised tool list equals exactly `{compose_handoff, verify_envelope, verify_pins}` — no more, no less.
VERIFY: `go test ./internal/tools -run TestToolSurfaceIsExactlyFenced`

### AC-F02 (unwanted-behavior)
GIVEN the non-test, non-fixture Go source
WHEN `TestFenceNoForbiddenSurface` greps the deny-list (meter/utilization/zone reads; policy or decision-table verdicts; operation triggers; host-inject paths `.claude/`, `.mcp.json`, `additionalContext`; crystalium/ingest/commit clients)
THEN zero matches are found.
VERIFY: `go test ./internal/tools -run TestFenceNoForbiddenSurface`

### AC-F03 (ubiquitous)
GIVEN README.md
WHEN the fence section is read
THEN capability starvation is documented — `--cap-drop ALL`, `--security-opt no-new-privileges`, no network client linked, no crystalium mount.
VERIFY: `grep -q 'cap-drop ALL' README.md`

### AC-F04 (ubiquitous)
GIVEN the handler packages
WHEN their sources are scanned
THEN no handler package calls `time.Now` — wall-clock enters only at the single server-layer seam when `ts` is omitted.
VERIFY: `! grep -rn 'time.Now' internal/compose internal/verify internal/ecl internal/hashing`

#### Track G — packaging

### AC-G01 (event-driven)
GIVEN the repo
WHEN `docker build` runs
THEN a multi-stage build produces a distroless/static image with the atomos binary in serve mode as entrypoint.
VERIFY: `docker build -t atomos:ci . && grep -q 'distroless' Dockerfile`

### AC-G02 (ubiquitous)
GIVEN the repo root
WHEN version sources are compared
THEN the `ATOMOS_VERSION` file contains exactly `0.1.0` matching the binary's compiled Version constant.
VERIFY: `go test ./cmd/atomos -run TestVersionMatchesVersionFile`

### AC-G03 (ubiquitous)
GIVEN `.github/workflows/conformance.yml`
WHEN inspected
THEN it runs the four gates — `go test ./...`, gofmt check, multi-arch build check, golden drift-guard.
VERIFY: `grep -q 'go test' .github/workflows/conformance.yml && grep -qi 'gofmt' .github/workflows/conformance.yml && grep -q 'regen-goldens' .github/workflows/conformance.yml`

### AC-G04 (event-driven)
GIVEN a pushed release tag
WHEN `release.yml` runs
THEN it buildx-builds linux/amd64 plus linux/arm64, pushes `ghcr.io/rynaro/atomos`, then captures the index digest into the release manifest.
VERIFY: `grep -q 'ghcr.io/rynaro/atomos' .github/workflows/release.yml && grep -q 'linux/arm64' .github/workflows/release.yml`

### AC-G05 (ubiquitous)
GIVEN the repo root
WHEN housekeeping files are checked
THEN README.md, CHANGELOG.md, LICENSE, .gitignore, .dockerignore all exist.
VERIFY: `test -f README.md && test -f CHANGELOG.md && test -f LICENSE && test -f .gitignore && test -f .dockerignore`

---

