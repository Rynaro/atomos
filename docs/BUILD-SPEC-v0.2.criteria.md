## Acceptance Criteria

The frozen copy of this section is `docs/BUILD-SPEC-v0.2.criteria.md`; its SHA-256
is recorded in the Freeze Record and in `BUILD-SPEC-v0.2.state.json`. All VERIFY
commands run from the atomos repo root.

#### Track H — compose_externalize_manifest

### AC-H01 (event-driven)
GIVEN an initialized stdio session
WHEN the client sends `tools/list`
THEN the response advertises exactly `compose_handoff`, `verify_envelope`, `verify_pins`, `compose_externalize_manifest` — no other tool.
VERIFY: `grep -rq 'func TestToolsListMatchesRegistry(' internal/server && go test ./internal/server -run TestToolsListMatchesRegistry`

### AC-H02 (ubiquitous)
GIVEN the tool registry
WHEN `TestToolSurfaceIsExactlyFenced` runs
THEN it asserts the advertised list has exactly four members, element-wise equal to the closed set — never a superset assertion.
VERIFY: `grep -rq 'func TestToolSurfaceIsExactlyFenced(' internal/tools && go test ./internal/tools -run TestToolSurfaceIsExactlyFenced && grep -q 'compose_externalize_manifest' internal/tools/registry_test.go && ! grep -qE 'len\(got\) >=|at least' internal/tools/registry_test.go`

### AC-H03 (event-driven)
GIVEN a manifest parity vector with a frozen `created_at`
WHEN `compose_externalize_manifest` runs on it
THEN the canonical manifest bytes are byte-identical to the kernel golden `fixtures/parity-manifest/<vector>/manifest.json`.
VERIFY: `grep -rq 'func TestManifestParityBytes(' internal/compose && go test ./internal/compose -run TestManifestParityBytes`

### AC-H04 (event-driven)
GIVEN the same manifest parity vector
WHEN `compose_externalize_manifest` runs on it
THEN `manifest_sha256` equals the golden `sha256` recorded for those bytes.
VERIFY: `grep -rq 'func TestManifestParitySHA(' internal/compose && go test ./internal/compose -run TestManifestParitySHA`

### AC-H05 (ubiquitous)
GIVEN any call with `write_sidecar` true
WHEN the sidecar file is written
THEN the SHA-256 of the file's bytes equals the returned `manifest_sha256` (M0 — hash what you write).
VERIFY: `grep -rq 'func TestManifestSidecarBytesHashToReportedSHA(' internal/compose && go test ./internal/compose -run TestManifestSidecarBytesHashToReportedSHA`

### AC-H06 (state-driven)
GIVEN two calls whose inputs are identical except for `write_sidecar`
WHEN both complete
THEN both return the same `manifest_sha256` (the digest never depends on whether a file was written).
VERIFY: `grep -rq 'func TestManifestSHAIndependentOfSidecar(' internal/compose && go test ./internal/compose -run TestManifestSHAIndependentOfSidecar`

### AC-H07 (optional-feature)
GIVEN `write_sidecar: false`
WHEN the call completes
THEN no file is written while the response still carries `manifest` and `manifest_sha256` with a null `manifest_path`.
VERIFY: `grep -rq 'func TestManifestWriteSidecarFalseDryRun(' internal/compose && go test ./internal/compose -run TestManifestWriteSidecarFalseDryRun`

### AC-H08 (event-driven)
GIVEN input that omits `write_sidecar` (default true)
WHEN the call completes
THEN `externalized-<ts>.json` exists under `out_dir` (default `.eidolons/.context`).
VERIFY: `grep -rq 'func TestManifestSidecarDefaultWrite(' internal/compose && go test ./internal/compose -run TestManifestSidecarDefaultWrite`

### AC-H09 (event-driven)
GIVEN input with no `summary`
WHEN the manifest is built
THEN `summary` is exactly the kernel's default sentence from `context_externalize.sh:100`.
VERIFY: `grep -rq 'func TestManifestDefaultSummary(' internal/compose && go test ./internal/compose -run TestManifestDefaultSummary`

### AC-H10 (ubiquitous)
GIVEN an empty or absent `session_id`
WHEN the manifest is emitted
THEN `session_id` renders as JSON `null`, never as an empty string.
VERIFY: `grep -rq 'func TestManifestSessionIDNull(' internal/compose && go test ./internal/compose -run TestManifestSessionIDNull`

### AC-H11 (unwanted-behavior)
GIVEN a list input containing empty-string entries
WHEN the manifest arrays are built
THEN those entries are absent from the emitted array (M3).
VERIFY: `grep -rq 'func TestManifestDropsEmptyEntries(' internal/compose && go test ./internal/compose -run TestManifestDropsEmptyEntries`

### AC-H12 (event-driven)
GIVEN a list entry containing an embedded newline
WHEN the manifest arrays are built
THEN the entry is split into one array element per non-empty line (M2 — `context_json_array` semantics).
VERIFY: `grep -rq 'func TestManifestSplitsNewlineEntries(' internal/compose && go test ./internal/compose -run TestManifestSplitsNewlineEntries`

### AC-H13 (ubiquitous)
GIVEN a list input that is absent, or whose every entry vanishes
WHEN the manifest is emitted
THEN the field renders as the inline empty array `[]` on a single line (jq-exact).
VERIFY: `grep -rq 'func TestManifestEmptyArrayInline(' internal/compose && go test ./internal/compose -run TestManifestEmptyArrayInline`

### AC-H14 (optional-feature)
GIVEN a caller-supplied `file_floor_reason`
WHEN the manifest is emitted
THEN `file_floor_reason` is the final key of the document, after `created_at` (M6).
VERIFY: `grep -rq 'func TestFileFloorReasonLast(' internal/compose && go test ./internal/compose -run TestFileFloorReasonLast`

### AC-H15 (unwanted-behavior)
GIVEN input that omits `file_floor_reason`
WHEN the manifest is emitted
THEN the document has exactly ten keys and no `file_floor_reason` — atomos never authors a reason it cannot observe.
VERIFY: `grep -rq 'func TestFileFloorReasonAbsentByDefault(' internal/compose && go test ./internal/compose -run TestFileFloorReasonAbsentByDefault`

### AC-H16 (ubiquitous)
GIVEN any manifest
WHEN it is emitted
THEN `ecm_version` is the hardcoded literal `"0.1"`, read from no file (M5).
VERIFY: `grep -rq 'func TestManifestEcmVersionLiteral(' internal/compose && go test ./internal/compose -run TestManifestEcmVersionLiteral`

### AC-H17 (ubiquitous)
GIVEN any `compose_externalize_manifest` call
WHEN it completes
THEN the only filesystem write is the single manifest file — no budget ledger, no meter read, no policy log, no durable-memory call (`context_externalize.sh:154-223` is out of fence).
VERIFY: `grep -rq 'func TestManifestWritesOnlyOneFile(' internal/compose && go test ./internal/compose -run TestManifestWritesOnlyOneFile`

### AC-H18 (ubiquitous)
GIVEN the handler packages `internal/compose`, `internal/verify`, `internal/ecl`, `internal/hashing`, `internal/jsonx`
WHEN `TestNoTimeNowInHandlerPackages` scans them
THEN none calls `time.Now` — wall-clock enters only at the single server-layer seam.
VERIFY: `grep -rq 'func TestNoTimeNowInHandlerPackages(' internal/tools && grep -q 'jsonx' internal/tools/registry_test.go && go test ./internal/tools -run TestNoTimeNowInHandlerPackages`

### AC-H19 (event-driven)
GIVEN a `compose_externalize_manifest` call that omits both `created_at` and `ts`
WHEN the server dispatches it
THEN both are filled at the server seam from a single UTC clock reading.
VERIFY: `grep -rq 'func TestManifestTimestampSeam(' internal/server && go test ./internal/server -run TestManifestTimestampSeam`

### AC-H20 (ubiquitous)
GIVEN the `internal/jsonx` extraction of the ordered-emitter primitives
WHEN the handoff envelope parity suite runs
THEN every existing vector still matches the kernel envelope byte-for-byte (the refactor is behavior-preserving).
VERIFY: `grep -rq 'func TestEnvelopeT2Parity(' internal/ecl && go test ./internal/ecl -run TestEnvelopeT2Parity`

### AC-H21 (unwanted-behavior)
GIVEN the new non-test Go sources (`internal/compose/externalize.go`, `internal/compose/manifest.go`, `internal/jsonx/`)
WHEN `TestFenceNoForbiddenSurface` greps the deny-list
THEN zero matches are found.
VERIFY: `grep -rq 'func TestFenceNoForbiddenSurface(' internal/tools && go test ./internal/tools -run TestFenceNoForbiddenSurface`

### AC-H22 (event-driven)
GIVEN a manifest parity vector run WITHOUT `file_floor_reason` (the default 10-key path most callers get)
WHEN `compose_externalize_manifest` runs on it
THEN the canonical manifest bytes are byte-identical to the vector's `core.json` (the jq-derived reason-less golden).
VERIFY: `grep -rq 'func TestManifestCoreParityBytes(' internal/compose && go test ./internal/compose -run TestManifestCoreParityBytes`

### AC-H23 (event-driven)
GIVEN the same reason-less run
WHEN it completes
THEN `manifest_sha256` equals the vector's `core.sha256`.
VERIFY: `grep -rq 'func TestManifestCoreParitySHA(' internal/compose && go test ./internal/compose -run TestManifestCoreParitySHA`

### AC-H24 (ubiquitous)
GIVEN any `compose_externalize_manifest` call
WHEN the MCP response is assembled
THEN the returned `manifest` object is decoded from the hashed `ManifestBytes` and deep-equals them — never built by a second serialization path (M0: nothing is hashed that is not returned).
VERIFY: `grep -rq 'func TestManifestResponseObjectDecodedFromHashedBytes(' internal/compose && go test ./internal/compose -run TestManifestResponseObjectDecodedFromHashedBytes`

### AC-H25 (event-driven)
GIVEN a list entry containing a CRLF sequence
WHEN the manifest arrays are built
THEN the entry splits on the LF only, leaving the CR on the tail of the preceding element (kernel-confirmed: `--anchor $'a\r\nb'` emits `["a\r","b"]`).
VERIFY: `grep -rq 'func TestManifestCROnlySplitsOnLF(' internal/compose && go test ./internal/compose -run TestManifestCROnlySplitsOnLF`

### AC-H26 (optional-feature)
GIVEN a caller-supplied `out_dir` other than the default
WHEN the sidecar is written
THEN the manifest file lands under that directory and `manifest_path` reports it.
VERIFY: `grep -rq 'func TestManifestCustomOutDir(' internal/compose && go test ./internal/compose -run TestManifestCustomOutDir`

### AC-H27 (unwanted-behavior)
GIVEN `write_sidecar` true and an `out_dir` whose file cannot be written
WHEN the call runs
THEN it returns a tool error rather than a success carrying a null path — `MkdirAll` is fail-soft, the write is not (the sole hard-error path in this tool).
VERIFY: `grep -rq 'func TestManifestSidecarWriteErrorSurfaces(' internal/compose && go test ./internal/compose -run TestManifestSidecarWriteErrorSurfaces`

### AC-H28 (ubiquitous)
GIVEN any `file_floor_reason` value, including the kernel's second literal (`"crystalium commit unreachable or timed out (…s budget)"`, `context_externalize.sh:199`)
WHEN the manifest is emitted
THEN the string is passed through verbatim — the field is free-form, never a closed set.
VERIFY: `grep -rq 'func TestFileFloorReasonIsFreeForm(' internal/compose && go test ./internal/compose -run TestFileFloorReasonIsFreeForm`

#### Track I — expanded parity vectors

### AC-I01 (ubiquitous)
GIVEN the handoff parity vector set
WHEN it is enumerated
THEN it contains `empty-sections`, `oversize-brief`, `multiline-task-state` and `empty-list-entries` alongside the three v0.1 vectors, each complete with `input.json`, `brief.md`, `envelope.json`, `sha256`, `advisory.json`.
VERIFY: `grep -rq 'func TestParityVectorsComplete(' internal/compose && go test ./internal/compose -run TestParityVectorsComplete`

### AC-I02 (event-driven)
GIVEN each new handoff edge-case vector
WHEN `compose_handoff` runs on it
THEN `brief_md` is byte-identical to the kernel golden `brief.md`.
VERIFY: `grep -rq 'func TestParityBriefBytes(' internal/compose && go test ./internal/compose -run TestParityBriefBytes`

### AC-I03 (event-driven)
GIVEN any handoff parity vector
WHEN `compose_handoff` runs on it
THEN `tokens_est` and `oversize` equal the kernel's own values captured in that vector's `advisory.json`.
VERIFY: `grep -rq 'func TestParityAdvisoryMatchesKernel(' internal/compose && go test ./internal/compose -run TestParityAdvisoryMatchesKernel`

### AC-I04 (unwanted-behavior)
GIVEN the `oversize-brief` vector
WHEN `compose_handoff` completes
THEN `brief_md` is untruncated and byte-equal to the golden despite `oversize: true`.
VERIFY: `grep -rq 'func TestOversizeAdvisoryNeverTruncates(' internal/compose && go test ./internal/compose -run TestOversizeAdvisoryNeverTruncates`

### AC-I05 (ubiquitous)
GIVEN `fixtures/parity-manifest/`
WHEN it is enumerated
THEN at least three frozen-`created_at` vectors exist (`manifest-defaults`, `manifest-populated`, `manifest-tool-origin`), each complete with `input.json`, `manifest.json`, `sha256`, `core.json`, `core.sha256`.
VERIFY: `grep -rq 'func TestManifestVectorsComplete(' internal/compose && go test ./internal/compose -run TestManifestVectorsComplete`

### AC-I06 (event-driven)
GIVEN `EIDOLONS_NEXUS` pointing at the PROVENANCE-pinned nexus checkout
WHEN `scripts/regen-goldens.sh` runs against that unchanged kernel
THEN it re-derives both fixture trees and leaves `git diff` clean.
VERIFY: `bash scripts/regen-goldens.sh && git diff --exit-code fixtures/parity fixtures/parity-manifest`

### AC-I07 (event-driven)
GIVEN the regen script's manifest arm
WHEN it captures a manifest golden
THEN it locates the artifact through `eidolons context externalize --json`'s `file_floor_path` rather than constructing the path itself (this grep pins the *mechanism* only — byte-fidelity to the kernel is enforced by AC-I06's diff-clean re-derivation, which is the real guard against a hand-authored golden).
VERIFY: `grep -q 'file_floor_path' scripts/regen-goldens.sh`

### AC-I08 (ubiquitous)
GIVEN the regen script's array-flag building
WHEN a vector's list contains an empty-string or newline-bearing element
THEN the element reaches the kernel as ONE argument, because the loop is index-driven and carries no non-empty guard (a `read`-split newline would become TWO `--decision` flags and capture a golden showing the brief SPLIT — the exact inverse of the behaviour the vector exists to pin).
VERIFY: `! grep -qE '\[ -n "\$v" \][[:space:]]*&&[[:space:]]*FLAGS' scripts/regen-goldens.sh`

### AC-I09 (state-driven)
GIVEN the `drift-guard` job in `conformance.yml`
WHEN it checks the re-derived goldens
THEN it detects **untracked** files too — `git status --porcelain --untracked-files=all` over both fixture trees, not `git diff --exit-code`, which is blind to a golden the executor forgot to `git add` (the failure mode is worst exactly where v0.2 adds an entirely new tree).
VERIFY: `awk '/^  drift-guard:/,0' .github/workflows/conformance.yml | grep -vE '^[[:space:]]*#' | grep -q 'untracked-files=all' && awk '/^  drift-guard:/,0' .github/workflows/conformance.yml | grep -vE '^[[:space:]]*#' | grep -q 'fixtures/parity-manifest'`

### AC-I10 (ubiquitous)
GIVEN `fixtures/parity/PROVENANCE`
WHEN it is inspected
THEN it records `ECM_VERSION`, `NEXUS_COMMIT` and the manifest vector list for the same capture run.
VERIFY: `grep -rq 'func TestGoldenProvenanceStamp(' internal/compose && go test ./internal/compose -run TestGoldenProvenanceStamp && grep -q 'MANIFEST_VECTORS' fixtures/parity/PROVENANCE`

### AC-I11 (ubiquitous)
GIVEN `scripts/regen-goldens.sh` after the manifest arm is added
WHEN it is linted
THEN it stays bash 3.2 clean — no `declare -A`, no `${var,,}`, no `readarray`/`mapfile`, no `&>>`.
VERIFY: `shellcheck -x -S error scripts/regen-goldens.sh && ! grep -nE 'declare -A|readarray|mapfile|&>>' scripts/regen-goldens.sh`

### AC-I12 (unwanted-behavior)
GIVEN a vector whose semantic fields are all absent (`defaults-only`, `manifest-defaults`), so the flag array is EMPTY
WHEN `scripts/regen-goldens.sh` runs under macOS system bash 3.2 in CI
THEN it completes without an `unbound variable` abort — the empty-array expansion trap is exercised on a real bash 3.2, not merely linted.
VERIFY: `awk '/^  drift-guard:/,0' .github/workflows/conformance.yml | grep -q 'macos-latest'`

### AC-I13 (ubiquitous)
GIVEN `scripts/regen-goldens.sh`
WHEN the kernel flag array is expanded
THEN it is never expanded empty — the array is seeded with the always-present `--json` flag before any conditional flag is appended.
VERIFY: `grep -q 'FLAGS=(--json)' scripts/regen-goldens.sh`

### AC-I14 (unwanted-behavior)
GIVEN a vector whose `input.json` contains a list element ending with a newline (which command substitution would silently strip, yielding a wrong golden)
WHEN `scripts/regen-goldens.sh` processes that vector
THEN it aborts with an explicit fixture error instead of capturing the golden.
VERIFY: `grep -q 'endswith("\\n")' scripts/regen-goldens.sh`

#### Track V — release hygiene

### AC-V01 (ubiquitous)
GIVEN the repo root
WHEN version sources are compared
THEN `ATOMOS_VERSION` contains exactly `0.2.0`, matching `cmd/atomos.Version`.
VERIFY: `grep -rq 'func TestVersionMatchesVersionFile(' cmd/atomos && go test ./cmd/atomos -run TestVersionMatchesVersionFile`

### AC-V02 (ubiquitous)
GIVEN the second compiled version constant
WHEN it is compared to the file
THEN `internal/server.Version` also equals the `ATOMOS_VERSION` contents (closing v0.1's unpinned twin).
VERIFY: `grep -rq 'func TestServerVersionMatchesVersionFile(' internal/server && go test ./internal/server -run TestServerVersionMatchesVersionFile`

### AC-V03 (ubiquitous)
GIVEN `CHANGELOG.md`
WHEN it is inspected
THEN it carries a `[0.2.0]` entry that names `compose_externalize_manifest`.
VERIFY: `grep -q '## \[0.2.0\]' CHANGELOG.md && grep -q 'compose_externalize_manifest' CHANGELOG.md`

### AC-V04 (ubiquitous)
GIVEN `README.md`
WHEN the tool surface is read
THEN it documents `compose_externalize_manifest` together with its file-floor-only nature.
VERIFY: `grep -q 'compose_externalize_manifest' README.md && grep -q 'file-floor' README.md`

### AC-V05 (ubiquitous)
GIVEN `fixtures/README.md`
WHEN it is read
THEN it documents the `parity-manifest` tree together with the single-document rule and the `core.json` derivation.
VERIFY: `grep -q 'parity-manifest' fixtures/README.md && grep -q 'core.json' fixtures/README.md && grep -qi 'single-document' fixtures/README.md`

---

