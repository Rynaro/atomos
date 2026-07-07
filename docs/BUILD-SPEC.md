# atomos MCP MVP — Build Spec (v0.1.0)

**Plan:** `atomos-mcp-mvp` · **Tier:** full (mechanical right-size, score 7) · **Planner:** RAMZA · **Executor:** Vivi
**Upstream:** FORGE ADR `docs/ARCHITECTURE.md` (settled 2026-07-07 — spec'd TO, not relitigated)
**Oracle (READ-ONLY):** the eidolons nexus bash kernel — `cli/src/context_handoff.sh`, `cli/src/verify_envelope.sh`, `cli/src/lib_context.sh`
**State:** `docs/BUILD-SPEC.state.json` · **Frozen criteria:** `docs/BUILD-SPEC.criteria.md` (SHA in Freeze Record)

---

## Framing

atomos is the **compose/verify executor MCP** for ECM — the 5th sibling, the
tonberry-analog (tonberry:ESL :: atomos:ECM). A Go stdio MCP server, distroless
OCI image, exposing a **closed** three-tool surface: `compose_handoff`,
`verify_envelope`, `verify_pins`. It is an *alternate surface* to the
always-canonical `eidolons context …` bash kernel: same inputs ⇒ same bytes.
The kernel is ground truth; atomos conforms to it, never the reverse.

The MVP's single load-bearing claim: **brief-SHA byte-identical parity** with
`eidolons context handoff` for the same inputs, proven by golden fixtures
captured from the kernel itself and guarded against drift in CI.

### Mirror, don't fix (parity traps — bind every track)

| # | Trap | Rule | Kernel anchor |
|---|------|------|---------------|
| 1 | `from_version` | A caller **INPUT**, echoed verbatim into `from.version`. **NEVER** `ATOMOS_VERSION`. Absent input ⇒ `"n/a"` (the kernel's `${EIDOLONS_VERSION:-n/a}` default). | `context_handoff.sh:212,224` |
| 2 | Envelope-version drift | Composer emits `envelope_version: "1.0"`; verifier expects `"2.0"` and **warns** (never fails) on anything else. atomos reproduces **both** sides. A parity fixture will show a benign version WARN — that is correct. Reconciling the drift is a nexus/ECL decision, not atomos's. | `context_handoff.sh:220`; `verify_envelope.sh:151-155` |
| 3 | Hardcoded trace | `trace: {host: "claude-code", model: "n/a", tier: "standard"}` is emitted verbatim by the kernel regardless of actual host. Reproduce verbatim. | `context_handoff.sh:230` |

---

## Right-sizing

Mechanical gate (`ramza-rightsize`, observable signals only):

| Signal | Value | Points |
|---|---|---|
| files-est | ~25 (new repo, full layout) | 2 |
| new-dep | `github.com/modelcontextprotocol/go-sdk v1.0.0` (tonberry's SDK, same major) | 1 |
| public-api | closed MCP tool surface is a published API | 1 |
| migration | none | 0 |
| security | SHA-256 integrity gate + structural fence | 1 |
| novel | new repo, 5th sibling executor | 1 |
| stakes | med (additive alternate surface; kernel stays canonical) | 1 |
| **Score** | | **7 → full** |

Full tier ⇒ this spec carries Stories, Confidence, Rejected Alternatives,
Risks, EARS criteria with a freeze, and a declared drift fence.

---

## Scope

**Repo:** new `Rynaro/atomos` (working tree `/home/rynaro/workspace/oss/agents/atomos/`). Layout per ADR §6, mirroring tonberry:

```
cmd/atomos/main.go                 # stdio MCP entrypoint: `atomos serve` / `atomos --version`
internal/server/server.go          # MCP protocol handler; dispatches from the registry
internal/tools/registry.go         # THE closed tool set (single source of truth)
internal/compose/handoff.go        # compose_handoff (+ handoff_test.go, parity_test.go)
internal/verify/envelope.go        # verify_envelope (+ envelope_test.go)
internal/verify/pins.go            # verify_pins (+ pins_test.go)
internal/ecl/envelope.go           # ordered jq-exact envelope emitter (T2)
internal/hashing/sha256.go         # sha256 helpers (parity with lib_context.sh:91-113)
fixtures/parity/<vector>/          # input.json, brief.md, envelope.json, sha256 (kernel goldens)
fixtures/parity/PROVENANCE         # ECM_VERSION + nexus commit SHA of capture
fixtures/{conformant,failing}/     # verify_envelope verdict fixtures
scripts/regen-goldens.sh           # runs the bash kernel as oracle (bash 3.2 safe)
Dockerfile                         # golang build → distroless/static, ENTRYPOINT serve
ATOMOS_VERSION                     # "0.1.0"
go.mod go.sum
.github/workflows/conformance.yml  # go test + parity + fence + gofmt + multi-arch + drift-guard
.github/workflows/release.yml      # buildx multi-arch → ghcr.io/rynaro/atomos, capture index digest
README.md CHANGELOG.md LICENSE .gitignore .dockerignore
```

**In scope (MVP tracks):**
- **A** — Go stdio MCP server scaffold: handshake, `tools/list`, exactly the closed set.
- **B** — `compose_handoff`: semantic brief fields + caller `from_version` + `thread_id` + pinned `ts` → brief md + ECL `INFORM` / `ecm/handoff-brief@0.1` envelope; T1 byte/SHA parity mandatory.
- **C** — `verify_envelope`: full kernel verdict matrix + the "1.0"-vs-"2.0" warn.
- **D** — `verify_pins`: pin-survival probe over an injected artifact + pin set.
- **E** — Parity suite: golden fixtures, `scripts/regen-goldens.sh`, CI drift-guard.
- **F** — Fence suite: `TestToolSurfaceIsExactlyFenced`, source deny-list test, capability-starvation doc.
- **G** — Packaging: distroless `Dockerfile`, conformance + release CI, `ATOMOS_VERSION` 0.1.0, housekeeping files.

**Non-goals (OUT — refuse if it drifts in):**
- `compose_externalize_manifest` — v0.2 (ADR §2.4).
- Nexus rostering (`roster/mcps.yaml` entry, `atomos.mcp.json.tmpl`, `eidolons.mcp.lock`) — Phase 3, a **separate ESL change in the nexus repo**.
- Anything metering/policy/trigger/inject/persist: no `meter`/`status`, no P1–P7 decision-table evaluation, no `compact`/`handoff-fresh` firing, no host prompt-surface writes, no crystalium client.
- Extending ECL's ten performatives or its envelope schema; re-declaring CRYSTALIUM layers/tiers.
- "Fixing" the kernel's envelope-version drift, or any nexus kernel edit whatsoever.
- One-shot CLI op mode (`atomos <op> …`, tonberry-style) — not in the ADR MVP; defer.
- npm/pip/brew publishing; the OCI-image flow is the design.

---

## Stories

- **S1 — session boundary.** As the orchestrating host at handoff time, I call `mcp__atomos__compose_handoff` and get a brief + envelope **byte-identical** to what `eidolons context handoff` would have written, so successor-session recall and ECL verification behave identically on either path.
- **S2 — receiver gate.** As a receiving Eidolon (or the orchestrator pre-dispatch), I call `verify_envelope` and get the same verdict the bash gate gives — pass/tamper/inconsistent/unverifiable/missing_payload/unsupported_algo/malformed — with `blocked` advisory only.
- **S3 — post-compaction probe.** As the host's post-compaction hook, I call `verify_pins` with the pin set and the re-injected context to learn which pins died; the kernel (not atomos) re-injects.
- **S4 — parity maintainer.** As the atomos maintainer, I run `scripts/regen-goldens.sh` against a pinned nexus commit; CI fails on golden drift so a kernel change can never silently break the alternate-surface claim.
- **S5 — Phase-3 rosterer.** As the nexus maintainer later, I consume `ghcr.io/rynaro/atomos@<digest>` with a tonberry-shaped run template — nothing in this MVP presumes nexus code changes.

---

## Approach

### Track A — server scaffold
`cmd/atomos/main.go` mirrors tonberry's entry (dual dispatch on `os.Args[1]`,
MVP arms: `serve`, `version|--version|-h`). MCP via
`github.com/modelcontextprotocol/go-sdk v1.0.0` (exactly tonberry's dependency;
`internal/server` wraps it the way tonberry's `internal/mcpserver` does).
Server identity: name `atomos`, version = compiled `Version` constant
(`"0.1.0"`), which a test pins to the `ATOMOS_VERSION` file.
`internal/tools/registry.go` is the **single** declaration site: an ordered
slice of `{name, description, inputSchema, handler}`; the server enumerates
`tools/list` from it and dispatches `tools/call` through it. No tool is
registered anywhere else.

### Track B — compose_handoff
Pure function `(HandoffInput) → (HandoffResult, error)` in
`internal/compose/handoff.go`. Input/output schemas exactly per ADR §2.1.
The brief is built by **string concatenation replicating the kernel byte-for-byte**
(`context_handoff.sh:107-168`). The byte template (normative — `\n` shown explicit):

```
# Session Handoff Brief\n
\n
## Identifiers\n
{for each non-empty anchor}   "- anchor: <a>\n"
{for each non-empty symbol}   "- symbol: <s>\n"
{for each non-empty decision} "- decision: <d>\n"
\n
## Failed approaches\n
{items "- <f>\n" each | else "(none recorded)\n"}
\n
## Next steps\n
{items "- <n>\n" each | else "(none recorded)\n"}
\n
## Narrative\n
\n
Task state: <task_state>\n
{if narrative != ""}  "\n<narrative>\n"
{if open_vars non-empty}  "\n## Open variables\n" then "- <o>\n" each
\n
## contains_tool_origin\n
<true|false>\n
```

Parity subtleties Vivi MUST preserve: **Identifiers has no `(none recorded)`
fallback** (it can be an empty section — kernel behavior); empty-string list
entries are **skipped** (`[ -n … ]` guards); the file is written with
`printf '%s'` semantics — brief bytes end at the final `\n` after the boolean,
nothing appended. Defaults: `task_state` ⇒ `"(no task-state summary provided)"`
(`:105`), `thread_id` ⇒ `session_id` else `handoff-<ts>` (`:104`).
`brief_sha256` = SHA-256 over the brief bytes (`:196`) — timestamp-free, the
**primary parity assertion**. Token estimate = `len(bytes)/4` floor
(`lib_context.sh:117-120`), advisory target fixed at 1500 (`oversize` flag
WARN-only, never truncates; atomos does **not** read `context-policy.yaml`).
Envelope: emitted by `internal/ecl/envelope.go`, an **ordered emitter**
reproducing `jq -n` output exactly — insertion order per
`context_handoff.sh:219-233`, 2-space indent, jq string escaping, trailing
newline, `parent_id: null`, numeric `size_bytes` (= brief byte length),
`artifact.path` = `handoff-<ts>.md` (relative, not absolute),
`message_id` = `msg-context-handoff-<ts>`, `objective` = fixed prefix + first
non-blank line of `task_state` (`:206-207`). `ts` format `YYYYMMDDTHHMMSSZ`;
`ts`/`iso_ts` are inputs — wall-clock only at the single server-layer seam when
omitted. Sidecar write (default on): `out_dir` (default `.eidolons/.context`,
mkdir-p fail-soft) gets `handoff-<ts>.md` + `handoff-<ts>.envelope.json`.
**Stop there** — the kernel's crystalium ingest (`:242-250`) is out of fence.

### Track C — verify_envelope
Pure function in `internal/verify/envelope.go` porting the kernel's exact
check order (`verify_envelope.sh`): (1) envelope readable/valid JSON else
`malformed`; (2) required fields — `envelope_version`, `from.eidolon`,
`to.eidolon`, `performative`, `artifact.path`, `integrity.method`,
`integrity.value` — else `malformed` listing them (`:133-149`); (3) version:
`!= "2.0"` ⇒ advisory warn, proceed (`:151-155`); (4) `integrity.method !=
"sha256"` ⇒ `unsupported_algo` (`:158-159`); (5) placeholder guard on
`integrity.value` (`""`, `PARENT_FILLS_*`, leading `<`, `TODO*`, `null`) ⇒
`unverifiable` (`:167-170`); (6) `artifact.sha256` present and ≠
`integrity.value` ⇒ `inconsistent` (`:172-175`); (7) payload resolution:
`artifact.path` relative to envelope dir, sibling fallback
`<envelope minus .envelope.json>`, else `missing_payload` (`:177-187`);
(8) recompute SHA-256 ⇒ `pass`/`tamper` (`:199-203`). Advisory size_bytes
cross-check warn (`:192-196`). `blocked: true` **only** when verdict ∈
{tamper, inconsistent, missing_payload, unsupported_algo} and `mode == block`
(`:81-85`); atomos never process-exits — exit-3 enforcement stays with the
kernel/orchestrator (fail-open P0). Inputs accept inline `envelope`/`payload`
or `_path` variants; inline envelope with no payload given resolves to
`missing_payload`.

### Track D — verify_pins
`internal/verify/pins.go` — ECM spec §3.2 probe as a pure function. Caller
supplies the pin list (`{id, marker|null}`; ids from the nexus
`roster/pins.yaml` — atomos does **not** bake the default set in, the roster
stays the authority) plus the post-op artifact (inline or path). Marker
defaults to the id token (literal match); explicit markers are Go regexps.
Output `{survived, pins[]{id,present}, missing[]}` — advisory, never blocking,
no writes, no re-injection.

### Track E — parity suite
MVP vectors (3): `defaults-only` (every optional field absent — exercises all
kernel defaults), `fully-populated` (every list populated, single-line
task_state), `narrative-open-vars` (narrative + open_vars + `contains_tool_origin:
true`). Edge-case vectors (oversize, multiline task_state, empty-string
entries) expand in v0.2 per ADR §7.
`scripts/regen-goldens.sh` (bash 3.2 safe): for each vector, runs the oracle in
a scratch dir with **no crystalium gating** (ingest skips, fail-open) and pinned
`EIDOLONS_VERSION`, via `eidolons context handoff <flags from input.json> --json`;
copies the produced brief/envelope/sha into the vector dir. Because the kernel
has **no `--ts` flag** (it wall-clocks `context_now_epoch_ts`), the script
**back-fills** the captured `ts` (from `message_id`), `iso_ts` (from
`trace.ts`), and `from_version` into `input.json` — the ts is *captured, not
chosen*. Stamps `fixtures/parity/PROVENANCE` with ECM_VERSION + the nexus
commit SHA. `internal/compose/parity_test.go` asserts T1 (brief bytes + SHA,
mandatory) and T2 (envelope bytes via the ordered emitter; documented fallback
= field-equal + `verify_envelope` pass, recorded in `fixtures/README.md`).
CI drift-guard: checks out `Rynaro/eidolons` at the PROVENANCE-pinned commit,
reruns the regen script, `git diff --exit-code fixtures/parity` — bumping the
pin is the deliberate act of absorbing a kernel change.

### Track F — fence suite
`TestToolSurfaceIsExactlyFenced` (registry == exactly the three names) and
`TestFenceNoForbiddenSurface` — walks non-test, non-fixture Go source and fails
on any deny-list identifier: meter/utilization/zone reads, policy or
decision-table verdicts, operation triggers (compact/handoff-fresh firing),
host-inject paths (`.claude/`, `.mcp.json`, `additionalContext`), crystalium
client verbs (ingest/commit/persist/recall). The deny-list lives in the test
with an explicit allowlist for itself; word-boundary identifier patterns keep
comments from false-positiving. Structural starvation is documented in README
(run shape `--cap-drop ALL --security-opt no-new-privileges`, no network
client linked, no crystalium mount). Handlers are pure: no `time.Now` outside
the single server seam — enforced by grep assertion.

### Track G — packaging
Dockerfile: multi-stage `golang:1.23` build (CGO_ENABLED=0, static) →
`gcr.io/distroless/static`, `ENTRYPOINT ["/atomos","serve"]` (`--version`
runnable via docker run args). `conformance.yml`: gofmt gate, `go test ./...`
(unit + parity + fence + tool-surface), multi-arch build check, golden
drift-guard step. `release.yml`: on tag — `docker buildx` for
`linux/amd64,linux/arm64`, push `ghcr.io/rynaro/atomos`, capture the **index
digest** into a release manifest (tonberry 0.3.1 pattern). `ATOMOS_VERSION`
file = `0.1.0`; README (fence + capability-starvation section, parity contract,
run shape), CHANGELOG, LICENSE (MIT, sibling-consistent), .gitignore,
.dockerignore.

**Build order for Vivi:** A → B(+E vectors as soon as compose exists) → C → D → F → G. Parity tests red-green against goldens from day one of Track B.

---

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

## Drift Fence

Declared in `BUILD-SPEC.state.json` via `ramza-drift --declare` and checked
against the executed diff at review time (`ramza-drift --state … --range`).

```yaml
files_allowed:            # the atomos repo, repo-relative
  - 'cmd/*'
  - 'internal/*'
  - 'fixtures/*'
  - 'scripts/*'
  - '.github/*'
  - 'docs/*'
  - 'Dockerfile'
  - '.dockerignore'
  - 'ATOMOS_VERSION'
  - 'go.mod'
  - 'go.sum'
  - 'README.md'
  - 'CHANGELOG.md'
  - 'LICENSE'
  - '.gitignore'
files_forbidden:          # the eidolons nexus kernel — READ-ONLY oracle
  - '<any path under the Rynaro/eidolons checkout>'
  -  # named explicitly because they are the oracle:
  - 'eidolons/cli/src/context_handoff.sh'
  - 'eidolons/cli/src/verify_envelope.sh'
  - 'eidolons/cli/src/lib_context.sh'
  - 'eidolons/cli/src/context_externalize.sh'
  - 'eidolons/roster/*'
```

atomos **NEVER** edits the nexus. If parity fails, the fix is in atomos (or, if
the kernel itself is wrong, a *separate* nexus ESL change proposed through the
normal lifecycle — never a drive-by edit from this campaign). Golden-pin bumps
happen in `fixtures/parity/PROVENANCE` inside the atomos repo only.

---

## Rejected Alternatives

1. **Python implementation** — rejected by FORGE ADR §1 (stateless compute matches tonberry's Go shape, not crystalium's store shape). Cited, not relitigated.
2. **`json.MarshalIndent` for the envelope** — sorts keys alphabetically; the kernel's `jq -n` emits insertion order. Breaks T2 by construction. Ordered emitter in `internal/ecl` instead.
3. **Fixing the envelope-version drift ("1.0"→"2.0")** — a nexus/ECL decision; atomos mirrors both sides verbatim (parity trap 2). Surfaced upstream, not resolved here.
4. **Stamping `ATOMOS_VERSION` into `from.version`** — diverges from the kernel envelope (parity trap 1). `from_version` is a caller input, default `"n/a"`.
5. **One-shot CLI op mode in the MVP** (tonberry has `tonberry <op>`) — not in the ADR §7 MVP list; adds surface without advancing the parity claim. Defer.
6. **Reading `context-policy.yaml` for the token target** — pulls stateful project config into a pure function and grazes the policy fence; the 1500 default is the kernel's own fallback and the check is WARN-only. Hardcode 1500.
7. **Baking the default pin set into the binary** — `roster/pins.yaml` is the authority and consumer projects extend it; a compiled copy drifts. Pins are caller input.
8. **Vendoring the bash kernel into the atomos repo as the oracle** (tonberry vendors its checker) — the handoff composer drags `lib.sh`/`lib_context.sh`/`lib_memory_probe.sh` with it; dual-maintenance risk exceeds the convenience. Pin the nexus commit in PROVENANCE and check it out for regen instead.
9. **`compose_externalize_manifest` in the MVP** — ADR §2.4 defers it to v0.2; the MVP proves parity on the load-bearing artifact first.

---

## Risks

- **R1 — jq byte-format reproduction is brittle (T2).** jq's 2-space indent + escaping quirks may resist exact reproduction. *Mitigation:* T1 (brief bytes + SHA) is the mandatory integrity anchor; T2 has the ADR-sanctioned documented fallback (semantic equivalence + verify pass) recorded in `fixtures/README.md`, byte-parity remains the standing goal.
- **R2 — kernel evolves, goldens rot.** *Mitigation:* PROVENANCE-pinned drift-guard in CI; a kernel change surfaces as a red build, absorbed by a deliberate pin bump + regen, never silently.
- **R3 — MCP SDK behavior (`go-sdk v1.0.0`) quirks over stdio.** *Mitigation:* same SDK + major as shipped tonberry 0.5.0; mirror its `internal/mcpserver` wiring; handshake AC-A01 pins behavior.
- **R4 — deny-list false positives/negatives (e.g. the word "zone" in a comment).** *Mitigation:* word-boundary identifier patterns, test-file allowlist, and the structural layer (capability starvation) backstops what grep misses.
- **R5 — brief edge-case divergence** (empty Identifiers section has no fallback; empty-string entries skipped; multiline task_state head). *Mitigation:* subtleties are spelled out in Approach Track B; `defaults-only` vector exercises the default path; v0.2 expands edge vectors.
- **R6 — regen environment leakage** (crystalium gating active, unpinned `EIDOLONS_VERSION`, macOS vs linux `date`). *Mitigation:* regen runs in a scratch dir, pins `EIDOLONS_VERSION`, back-fills captured `ts`; script is bash-3.2-safe and CI regen runs on linux.

---

## Confidence

Scored via `ramza-score --rubric confidence` (recorded in the state file's `gates[]`):
pattern_match 90 (tonberry is a shipped, near-isomorphic template) ·
requirement_clarity 92 (FORGE ADR is anchor-dense and settled) ·
decomposition_stability 85 (tracks are near-independent; B↔E couple by design) ·
constraint_compliance 90 (fence + parity traps are mechanically testable).
**Weighted: 89 → AUTO_PROCEED.**

---

## Executor Notes (Vivi)

- **Kernel is READ-ONLY ground truth.** You will read `context_handoff.sh`, `verify_envelope.sh`, `lib_context.sh` constantly; you will never edit them. Run the oracle from a nexus checkout with `EIDOLONS_NEXUS="<checkout>" bash <checkout>/cli/eidolons context handoff …`.
- **Parity-oracle workflow:** implement compose against the byte template in Approach Track B; capture goldens early with `scripts/regen-goldens.sh`; remember the kernel wall-clocks its `ts` — the script back-fills captured `ts`/`iso_ts`/`from_version` into `input.json`, you never choose them. Diff failures print byte offsets: compare with `cmp`/hexdump, not eyeballs.
- **Go idioms:** pure handlers `(input) → (artifact, error)`; table-driven tests; `crypto/sha256` + lowercase-hex encode (matches `shasum -a 256` output); the ordered envelope emitter writes fields in `context_handoff.sh:219-233` insertion order with 2-space indent, jq string escaping, `parent_id` as literal `null`, numeric `size_bytes`, trailing newline. No `map[string]interface{}` marshaling for the envelope — field order dies there.
- **Version discipline:** `Version` const in `cmd/atomos` mirrors tonberry; test-pinned to the `ATOMOS_VERSION` file (AC-G02). Never let it leak into envelopes (AC-B05).
- **Fence discipline:** every new identifier you introduce is one grep away from `TestFenceNoForbiddenSurface` — do not name things "meter", "policy", "inject", or "persist" even innocently. If a tool seems to want a fourth registry entry, stop: the set is closed, that is a spec revision.
- **Shell scripts are bash 3.2** (`regen-goldens.sh`): no associative arrays, no `${var,,}`, no readarray/mapfile, no `&>>` — same rule as the nexus CLI (its CI runs macos bash 3.2).
- **Fail-open everywhere:** verify tools report, never enforce; compose never blocks on oversize; sidecar mkdir is fail-soft. The only hard errors are malformed inputs.
- **Dependencies:** `github.com/modelcontextprotocol/go-sdk v1.0.0` only. No YAML lib (pins arrive as JSON input), no HTTP client, no crystalium SDK — their absence is part of the fence.

---

## Freeze Record

- **Criteria file:** `docs/BUILD-SPEC.criteria.md` (byte-extracted from the Acceptance Criteria section above)
- **criteria_sha256:** `678c084e3509c36a88ba4ea232256c8792662b09a86d85e45c616ac7d4688520` (frozen 2026-07-07T18:33Z)
- **Gates run:** rightsize full/7 · ramza-lint pass (full) · ramza-ears-lint pass (45 criteria) · confidence 89.25 AUTO_PROCEED · drift scope declared (15 globs) · freeze
- **State:** `docs/BUILD-SPEC.state.json` (schema `ramza/plan-state.v1`)
- Amendments only via `ramza-freeze --amend --reason` — hash-chained, never silent.
