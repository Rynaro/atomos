# atomos — Architecture Decision Record

**Status:** settled (foundational). No implementation in this document.
**Author:** FORGE (reasoner). **Feeds:** RAMZA (implementation spec) → Vivi (build).
**Date:** 2026-07-07.
**Version target:** `ATOMOS_VERSION` = `0.1.0`.

---

## 0. What atomos is (decided upstream — not relitigated here)

atomos is the **compose/verify executor MCP** for the ECM (Eidolons Context
Management) context lifecycle — the tonberry-analog for ECM the way tonberry is
the executor-analog for ESL. It is a committed **P3** build (maintainer decision
2026-07-07, superseding an earlier FORGE NO-GO; recorded in
`.spectra/changes/ecm-p2-host-adapters/amendment-atomos-go.md` and
`docs/specs/ecm/decisions/atomos-go-no-go.md` §0 in the nexus repo). It was
chosen for **MCP-surface parity + ecosystem consistency**, not feasibility.

### 0.1 The binding scope FENCE (immovable)

atomos does **compose/verify ONLY**:

- **Composes** the ECM session-handoff brief + its ECL envelope, and
  (post-MVP) the identifier/externalize manifest.
- **Verifies** ECL envelope integrity (SHA-256) and pin-set survival.
- **NEVER** meters context, decides policy, triggers operations, or performs
  context injection. Metering / policy / triggering stay in the **harness
  kernel**; injection is a **host-surface** property no MCP can manufacture;
  persistence is **DEFERRED to CRYSTALIUM**; transport/envelope schema is
  **DEFERRED to ECL**.
- It is **ADDITIVE**. The `eidolons context …` bash kernel verbs remain the
  canonical, always-available path. atomos is an *alternate surface*, never the
  sole path.

The kernel is the **source of truth**. atomos conforms to it, never the reverse
— exactly as tonberry's verify is "byte-identical to the canonical bash
conformance checker" (nexus `roster/mcps.yaml:214`).

---

## 1. Language / stack — **Go**

**Decision: Go**, mirroring tonberry and junction (`roster/mcps.yaml`: tonberry
`kind: oci-image`, `ghcr.io/rynaro/tonberry`; junction `kind: binary`, Go).

**Rationale.**
1. **Ecosystem executor precedent is Go.** tonberry (the ESL compose/verify
   executor) and junction (the harness) are Go; crystalium is Python *because it
   is a stateful store* (SQLite/FTS, embeddings, Dream). atomos is a **stateless
   compute executor** — pure functions over inputs — which is exactly tonberry's
   shape, not crystalium's.
2. **Byte-parity favors Go's deterministic stdlib.** `crypto/sha256` +
   ordered JSON emission give dependency-free, reproducible hashing and
   serialization — the load-bearing requirement of the parity contract (§3).
3. **Deployment shape matches the sibling.** atomos ships as a distroless
   single-static-binary OCI image pulled by digest and run per-project via
   `docker run --rm -i … serve` (the tonberry template, `cli/templates/mcp/tonberry.mcp.json.tmpl`).
   Go gives a tiny image and a fast cold start, which matters for a
   spawned-per-invocation stdio MCP.

**Rejected alternative — Python (crystalium-consistent).** Rejected because
atomos is stateless compute, not a store; a Python image is heavier and slower to
cold-start for a `--rm` stdio server; and bash byte-parity is equally achievable
in Go with far less runtime surface. The ecosystem's *executor* precedent
(tonberry), not its *store* precedent (crystalium), is the correct anchor.

---

## 2. MCP tool surface — the CLOSED set

Every tool is compose or verify. The set is **closed**: extensions require a
spec revision (same discipline as ECL's closed 10-performative set).

| Tool | Class | MVP? | Kernel parity target |
|---|---|---|---|
| `compose_handoff` | compose | **yes** | `cli/src/context_handoff.sh` |
| `verify_envelope` | verify | **yes** | `cli/src/verify_envelope.sh` |
| `verify_pins` | verify | **yes** | ECM spec §3.2 pin probe + `roster/pins.yaml` |
| `compose_externalize_manifest` | compose | no (v0.2) | `cli/src/context_externalize.sh` (manifest build only) |

**Deliberately EXCLUDED (the fence, §4):** no `meter`/`status` (never reads
`meter.json`, `utilization`, `zone`); no `policy`/`decide` (never evaluates the
P1–P7 decision table or emits a verdict); no `trigger`/`compact`/`handoff-fresh`
(never *fires* an operation); no `inject`/`recall` (never writes a host prompt
surface); no `ingest`/`commit`/`persist` (never calls CRYSTALIUM).

### 2.1 `compose_handoff` (compose)

Mirrors `eidolons context handoff` (`cli/src/context_handoff.sh`). Produces a
byte-identical brief and a hash-identical envelope for the same inputs.

**Purpose.** Build the session-handoff brief markdown (survival-priority section
order: Identifiers → Failed approaches → Next steps → Narrative → Open variables
→ `contains_tool_origin`; `context_handoff.sh:107-168`) and the ECL sidecar
envelope (`INFORM`, `ecm/handoff-brief@0.1`, SHA-256 over the brief;
`context_handoff.sh:195-235`).

**Input schema (design-level).**
```jsonc
{
  "task_state": "string",                 // default "(no task-state summary provided)" (context_handoff.sh:105)
  "narrative": "string | null",
  "anchors": ["path:line", ...],
  "symbols": ["name", ...],
  "decisions": ["text", ...],
  "failed_approaches": ["text", ...],
  "open_vars": ["text", ...],
  "next_steps": ["text", ...],
  "thread_id": "string | null",           // default session_id, else "handoff-<ts>" (context_handoff.sh:104)
  "session_id": "string | null",
  "from_version": "string",               // PARITY: echoes caller's $EIDOLONS_VERSION (context_handoff.sh:212). NOT ATOMOS_VERSION.
  "contains_tool_origin": false,
  "ts": "20260707T130000Z",               // PARITY-CRITICAL: injected epoch token; if omitted, atomos wall-clocks it
  "iso_ts": "2026-07-07T13:00:00Z",       // PARITY-CRITICAL: injected ISO ts (used in trace.ts)
  "write_sidecar": true,                  // default true; writes the file pair to .eidolons/.context/
  "out_dir": ".eidolons/.context"         // workspace-relative
}
```

**Output schema.**
```jsonc
{
  "brief_md": "string",                   // exact bytes of handoff-<ts>.md (no trailing newline; context_handoff.sh:171)
  "brief_sha256": "hex",                  // sha256 over brief_md (context_handoff.sh:196)
  "envelope": { /* the ECL-shaped object, §5.3 */ },
  "brief_path": ".eidolons/.context/handoff-<ts>.md | null",       // null when write_sidecar=false
  "envelope_path": ".eidolons/.context/handoff-<ts>.envelope.json | null",
  "tokens_est": 0, "oversize": false      // advisory 1500-token target (context_handoff.sh:173-193); WARN-only, never truncates
}
```

**Fence note.** atomos writes the brief+envelope pair (sidecar-on-disk P0) and
**stops**. It does **not** run the crystalium ingest that the kernel does next
(`context_handoff.sh:242-250`) — persistence is CRYSTALIUM's, reached via the
kernel's `eidolons context externalize --envelope <path>` or crystalium directly.

### 2.2 `verify_envelope` (verify)

Mirrors `eidolons verify-envelope` (`cli/src/verify_envelope.sh`) — the
deterministic, non-LLM SHA-256 gate.

**Purpose.** Recompute the payload SHA-256 and compare to the envelope's
integrity tag; cross-check `artifact.sha256 == integrity.value`; guard
placeholders.

**Input schema.**
```jsonc
{
  "envelope": { ... } ,                   // OR "envelope_path": "…"
  "payload": "string",                    // OR "payload_path": "…" (resolved from artifact.path, sibling fallback; verify_envelope.sh:177-187)
  "mode": "warn | block"                  // advisory only in atomos (§5.1) — sets the `blocked` flag, never process-exits the host
}
```

**Output schema** — the full kernel verdict matrix
(`verify_envelope.sh:45-47`):
```jsonc
{
  "verdict": "pass | tamper | inconsistent | unverifiable | missing_payload | unsupported_algo | malformed",
  "expected_sha": "hex", "actual_sha": "hex",
  "blocked": false,                       // true only when verdict is a genuine integrity failure AND mode=block (verify_envelope.sh:81-85)
  "message": "string",
  "from": "eidolon", "to": "eidolon", "performative": "…"
}
```

### 2.3 `verify_pins` (verify)

Mirrors the ECM §3.2 **post-op pin-survival probe** (spec §3.2;
`roster/pins.yaml`). This is the "pin set survives every lossy operation" P0
check (spec §6.4) as a callable, advisory verifier.

**Purpose.** Given the default+extended pin set and a post-operation artifact
(the injected/re-injected context), report which pin markers survived. Advisory,
**never blocking** (spec §3.2 "Advisory, never blocking").

**Input schema.**
```jsonc
{
  "pins": [ { "id": "cortex_routing_digest", "marker": "regex | null" }, ... ],  // ids from pins.yaml; marker defaults to the id token
  "artifact": "string"                    // OR "artifact_path": "…" — the post-op injected context to grep
}
```

**Output schema.**
```jsonc
{
  "survived": true,                       // all pins present
  "pins": [ { "id": "cortex_routing_digest", "present": true }, ... ],
  "missing": [ "id", ... ]                // repair targets for the kernel's re-inject hook (atomos does NOT re-inject)
}
```

**Fence note.** atomos *reports* missing pins; it does **not** re-inject them
(that is the host's post-compaction hook surface, spec §3.2). Verify only.

### 2.4 `compose_externalize_manifest` (compose) — v0.2, in-fence, not MVP

Mirrors only the **manifest-build** portion of `cli/src/context_externalize.sh`
(the `MANIFEST_JSON` object, `context_externalize.sh:116-138`). It emits the
identifier manifest JSON (anchors/symbols/decisions/failed-approaches/open-vars/
`contains_tool_origin`/session/created_at) and its SHA.

**Why in the closed set but NOT the MVP.** The manifest object is a pure build —
trivially mirrored — but its *value* in the kernel is the CRYSTALIUM persist
chain (`context_externalize.sh:154-196`), which atomos must **never** touch. So
atomos would emit only the **file-floor manifest** — a strict subset. That is
legitimate compose work (in-fence), but the MVP's job is to prove the parity
claim on the **load-bearing artifact first** (the handoff brief + envelope), so
the manifest ships in v0.2. When shipped it NEVER calls crystalium; output is
`{ manifest: {…}, manifest_sha256: "hex" }`.

---

## 3. Parity contract — enforcement mechanism

**Decision: shared golden fixtures + a parity conformance suite**, with the bash
kernel as the canonical oracle. atomos same-input ⇒ same brief-bytes and same
brief-SHA as `eidolons context handoff`; if it drifts, CI fails.

### 3.1 The load-bearing simplification

The **brief body contains no timestamp** — it is a pure function of the semantic
inputs (`context_handoff.sh:109-168`, plain string concatenation). Therefore
`brief_sha256` (`context_handoff.sh:196`, sha over the brief file) is
**timestamp-independent** and is the **primary parity assertion**. Only the
**envelope** embeds the timestamp (`message_id`, `artifact.path`, `trace.ts`;
`context_handoff.sh:210-233`), so envelope byte-parity requires the input vector
to pin `ts`/`iso_ts`.

### 3.2 Two-tier parity

- **T1 — mandatory (byte-exact):** `compose_handoff.brief_md` bytes ==
  kernel `handoff-<ts>.md` bytes, and `brief_sha256` == kernel SHA. This is the
  integrity anchor — if a receiver runs `verify_envelope` on atomos's output, it
  MUST pass against the kernel-composed brief.
- **T2 — target (envelope canonical bytes):** atomos's envelope bytes ==
  kernel envelope bytes. The kernel emits via `jq -n` (2-space indent, **insertion
  order**, jq string-escaping, trailing newline — `context_handoff.sh:209-235`).
  atomos MUST reproduce that exact formatting through an **ordered emitter**
  (`internal/ecl/envelope.go`), NOT Go's `json.MarshalIndent` (which sorts keys).
  **Documented fallback contract** (if jq-exact whitespace proves brittle):
  *semantic-envelope-equivalence* — identical field values and a `verify_envelope`
  pass — is the accepted contract, recorded in the fixture README, with T2
  byte-parity as the standing goal.

### 3.3 Mechanics

- `fixtures/parity/<vector>/input.json` — a `compose_handoff` input set with a
  **frozen `ts`/`iso_ts`**.
- `fixtures/parity/<vector>/{brief.md, envelope.json, sha256}` — goldens
  captured by **running the bash kernel** (`scripts/regen-goldens.sh` invokes
  `eidolons context handoff … --json` against each vector).
- `internal/compose/parity_test.go` asserts T1 (byte+SHA) for every vector and
  T2 (or the documented fallback) for the envelope.
- **Drift guard (CI, mirrors tonberry's vendored-oracle drift guard,
  `roster/mcps.yaml:257`):** goldens carry a provenance stamp
  (`ECM_VERSION` + nexus commit SHA). A conformance step re-derives goldens from
  the kernel and diffs; a mismatch fails the build so kernel changes cannot
  silently break the alternate-surface claim.

### 3.4 Two parity-critical inputs (flag for RAMZA)

1. **`from_version` must be an input, not `ATOMOS_VERSION`.** The kernel stamps
   `from.version` = `$EIDOLONS_VERSION` (`context_handoff.sh:212,224`). If atomos
   stamped its own version the envelope would diverge. `compose_handoff` echoes
   the caller-supplied `from_version`.
2. **Known kernel cross-verb inconsistency atomos MUST mirror (not "fix").**
   `context_handoff.sh:220` emits `envelope_version: "1.0"`, while
   `verify_envelope.sh:151-155` expects `"2.0"` and *warns* on anything else.
   atomos's `compose_handoff` reproduces `"1.0"` (parity), and its
   `verify_envelope` reproduces the same acceptance (accept `2.0`, warn
   otherwise). A parity fixture will therefore show a benign version WARN. This is
   faithful mirroring; reconciling the drift is a **nexus/ECL decision, not
   atomos's** — surfaced here, not resolved here.

---

## 4. Fence enforcement — mechanical, structural

The compose/verify-only guarantee is enforced four ways, all mechanical:

1. **Single closed registry.** `internal/tools/registry.go` is the *only* place
   tools are declared; the server enumerates from it. A conformance test
   `TestToolSurfaceIsExactlyFenced` asserts the advertised list ==
   `{compose_handoff, verify_envelope, verify_pins}` (+`compose_externalize_manifest`
   at v0.2) — no more, no less. This mirrors the roster's `exposes_tools.list`
   audit (`roster/mcps.yaml:225-236`) and becomes the nexus catalogue's
   `exposes_tools.list` at Phase 3.
2. **Source deny-list test.** `TestFenceNoForbiddenSurface` greps the atomos
   source for forbidden concerns and fails on any hit: no `meter`/`utilization`/
   `zone` read, no `policy`/decision-table verdict, no operation *trigger*, no
   host-inject path (no writes to `.claude/`, `.mcp.json`, `additionalContext`),
   no crystalium client / `ingest` / `commit`.
3. **Capability starvation (structural).** The container runs `--cap-drop ALL
   --security-opt no-new-privileges` (tonberry template) with **no network
   client and no crystalium mount** — atomos *cannot* persist or transport even
   if code tried. Compose/verify are pure functions; there is simply no wire to
   metering, policy, persistence, or the host prompt surface.
4. **Pure handlers.** Every tool handler is `(input) → (artifact, error)` with no
   ambient state, no clock except an injected `ts`, and at most one explicit,
   opt-in sidecar write. No handler reads a meter or writes a policy verdict
   because no such code path is linked.

---

## 5. State & boundaries

### 5.1 Stateless

Every tool is pure inputs→artifacts. No DB, no session memory, no cross-call
state. The image is `--rm` ephemeral like tonberry. `mode: block` on
`verify_envelope` sets an advisory `blocked` flag in the response; atomos **never
process-exits or blocks the host** — enforcement (exit 3, ECL §6.2.2) stays with
the kernel/orchestrator `eidolons verify-envelope --block`. atomos reports;
the harness decides. This preserves ECM's fail-open P0 (spec §6.9).

### 5.2 Sidecar-on-disk AND in-response (both)

Sidecar-on-disk is an ECM P0 (spec §6.3: meter/log/**handoff brief** never
in-context-only). `compose_handoff` therefore:
- **returns** `brief_md` + `brief_sha256` + `envelope` in the tool response
  (in-band for the caller), and
- **writes** the byte-identical `handoff-<ts>.md` + `handoff-<ts>.envelope.json`
  pair to `.eidolons/.context/` when `write_sidecar: true` (**default**, matching
  the kernel).

Default is write-sidecar so end-to-end parity with the kernel path holds;
`write_sidecar: false` gives a response-only dry preview.

### 5.3 Staying clear of persistence and transport while emitting an ECL envelope

- **Persistence (CRYSTALIUM):** atomos writes only the **file-floor** pair and
  stops. It never calls `crystalium_ingest`/`commit`/`plan_checkpoint`. The
  kernel's post-compose ingest (`context_handoff.sh:242-250`) is *out of fence*;
  a caller wanting durable memory feeds atomos's envelope to
  `eidolons context externalize --envelope <path>` (`context_externalize.sh:186-192`).
- **Transport / schema (ECL):** atomos emits an ECL-**shaped** envelope by
  **reproducing** the structure the kernel already writes
  (`context_handoff.sh:219-233`) — it hard-codes the one performative the kernel
  uses (`INFORM`) and the one artifact kind (`ecm/handoff-brief@0.1`). It does
  **not** define, validate against, extend, or own the ECL performative set or
  envelope schema. The ECL spec stays the authority; atomos is a faithful emitter
  exactly as the bash kernel is. Envelope shape reproduced verbatim:
  ```jsonc
  {
    "envelope_version": "1.0",
    "message_id": "msg-context-handoff-<ts>",
    "thread_id": "<thread_id>", "parent_id": null,
    "from": { "eidolon": "eidolons-context-kernel", "version": "<from_version>" },
    "to":   { "eidolon": "session_successor", "version": "n/a" },
    "objective": "Session handoff brief for context-lifecycle succession (ECM P1): <task_state_head>",
    "performative": "INFORM",
    "artifact": { "kind": "ecm/handoff-brief@0.1", "schema_version": "0.1",
                  "path": "handoff-<ts>.md", "sha256": "<sha>", "size_bytes": <n> },
    "integrity": { "method": "sha256", "value": "<sha>" },
    "trace": { "ts": "<iso_ts>", "host": "claude-code", "model": "n/a", "tier": "standard" },
    "topic_key": "session_handoff",
    "contains_tool_origin": <bool>
  }
  ```
  `task_state_head` = first non-blank line of `task_state`
  (`context_handoff.sh:206-207`).

---

## 6. Repo layout + packaging (mirror tonberry)

```
Rynaro/atomos/
  cmd/atomos/main.go                 # stdio MCP entrypoint: `atomos serve` / `atomos --version`
  internal/
    server/server.go                 # MCP protocol handler; dispatches from the registry
    tools/registry.go                # THE CLOSED tool set (single source of truth)
    compose/handoff.go               # compose_handoff — brief + envelope
    compose/handoff_test.go
    compose/externalize.go           # compose_externalize_manifest (v0.2)
    verify/envelope.go               # verify_envelope (full verdict matrix)
    verify/envelope_test.go
    verify/pins.go                   # verify_pins (§3.2 probe)
    verify/pins_test.go
    ecl/envelope.go                  # ECL-shaped ordered emitter (jq-exact, §3.2 T2)
    hashing/sha256.go                # sha256 helpers (parity with lib_context.sh)
  fixtures/
    parity/<vector>/{input.json, brief.md, envelope.json, sha256}   # kernel goldens (§3.3)
    conformant/                      # valid envelopes → verify pass
    failing/                         # tamper / inconsistent / missing / unsupported → verdicts
  scripts/regen-goldens.sh           # runs the bash kernel to (re)derive parity goldens
  Dockerfile                         # multi-stage: golang build → distroless/static; ENTRYPOINT serve
  ATOMOS_VERSION                     # "0.1.0"
  go.mod  go.sum
  .github/workflows/conformance.yml  # go test + parity + fence + tool-surface + gofmt + multi-arch build
  .github/workflows/release.yml      # buildx multi-arch → push ghcr.io/rynaro/atomos → capture index digest
  README.md  CHANGELOG.md  LICENSE
```

**Versioning.** `ATOMOS_VERSION` file starts `0.1.0` (mirrors tonberry's
`ESL_VERSION` file). It names the **image/MCP version**, surfaced by
`atomos --version` and captured in the nexus catalogue's `versions.latest`. It is
**not** stamped into composed envelopes — envelope `from.version` echoes the
caller's `from_version` (§3.4).

**Image / run shape.** `ghcr.io/rynaro/atomos`, pulled by **digest**, `serve`
entrypoint. The nexus template (Phase 3) mirrors `tonberry.mcp.json.tmpl`:
`docker run --rm -i --name atomos-<slug> --label eidolons.project=<slug> -v
<root>:/workspace -w /workspace --cap-drop ALL --security-opt no-new-privileges
ghcr.io/rynaro/atomos@<digest> serve`.

**CI.**
- `conformance.yml` — `go test ./...` (unit + **parity** + **fence** +
  **tool-surface** suites), `gofmt` gate, multi-arch build check. Golden
  drift-guard step (§3.3).
- `release.yml` — `docker buildx` multi-arch (`linux/amd64` + `linux/arm64`,
  matching tonberry 0.3.0), push to ghcr, **capture the index digest** into a
  release manifest (tonberry 0.3.1 pattern, `roster/mcps.yaml:257`).

**Health probes (Phase-3 catalogue entry):** `docker_cli`, `docker_daemon`,
`image_local`, `registry_reachable` (identical to tonberry,
`roster/mcps.yaml:274-279`).

---

## 7. Build phasing

### MVP — v0.1.0 (smallest shippable atomos that proves the alternate-surface claim)

Proves compose/verify parity on the **load-bearing artifact** (handoff brief +
envelope) end-to-end:

- `cmd/atomos` stdio MCP server + `internal/server` + `internal/tools/registry`.
- **`compose_handoff`** — brief + `INFORM` envelope, **T1 byte/SHA parity** to
  `context_handoff.sh`, with sidecar-on-disk write (default on).
- **`verify_envelope`** — full verdict matrix parity to `verify_envelope.sh`.
- **`verify_pins`** — §3.2 marker-survival probe.
- **Parity suite** + golden fixtures (frozen-`ts` vectors) + `scripts/regen-goldens.sh`.
- **Fence suite** — `TestToolSurfaceIsExactlyFenced` + `TestFenceNoForbiddenSurface`.
- Dockerfile → distroless static; `conformance.yml` + `release.yml` →
  `ghcr.io/rynaro/atomos:0.1.0` pulled by digest.
- **NOT in MVP:** `compose_externalize_manifest`; nexus integration.

### Phase 2 — v0.2.0 (complete the closed set + harden)

- **`compose_externalize_manifest`** — file-floor manifest only, no crystalium.
- Expand parity vectors (edge cases: empty sections, oversize brief WARN,
  `contains_tool_origin: true`, multiline `task_state`).
- Multi-arch index + the vendored-oracle **drift-guard** CI (tonberry pattern).

### Phase 3 — nexus integration (a SEPARATE ESL change in the nexus repo)

atomos ships to ghcr **first**, then the nexus rosters it — the same order as
every other MCP (build+tag upstream → roster the digest here). This phase is a
**nexus-side ESL change**, distinct from the atomos build campaign:

- Add the **5th catalogue entry** to `roster/mcps.yaml`: `name: atomos`,
  `kind: oci-image`, `source.image: ghcr.io/rynaro/atomos`,
  `exposes_tools.glob: "mcp__atomos__*"` + the enumerated `list`,
  `versions.{latest,pins.stable,releases.<v>.digest}`,
  `install.{template, hosts_wired}`, tonberry health probes.
- Add `cli/templates/mcp/atomos.mcp.json.tmpl` (tonberry-shaped `docker run`).
- Update `eidolons.mcp.lock` via `eidolons mcp install/upgrade`; CHANGELOG.
- **Open decision for the nexus change (recommendation, not settled here):**
  `wiring_mode` + `grants_to_eidolons`. Compose is a session-boundary
  orchestration op and `verify_envelope` is already the symmetric per-Eidolon
  gate the kernel provides, so `wiring_mode: transport` (registered in
  `.mcp.json`, dispatched by the orchestration layer — junction's posture,
  `roster/mcps.yaml:114-116`) is the leaner default; `allowlist` + a small grant
  set is the alternative. Defer the final call to the Phase-3 ESL right-sizing.

---

## 8. Anchored kernel facts (for RAMZA's spec)

| Fact | Anchor |
|---|---|
| Brief section order + string build (no timestamp in body) | `cli/src/context_handoff.sh:107-168` |
| Brief written with `printf '%s'` (no trailing newline) | `cli/src/context_handoff.sh:171` |
| Brief SHA-256 = parity anchor | `cli/src/context_handoff.sh:196` |
| Envelope shape / field order / `INFORM` / `ecm/handoff-brief@0.1` | `cli/src/context_handoff.sh:209-233` |
| Envelope `envelope_version: "1.0"` | `cli/src/context_handoff.sh:220` |
| `from.version` = `$EIDOLONS_VERSION` | `cli/src/context_handoff.sh:212,224` |
| `task_state_head` = first non-blank line | `cli/src/context_handoff.sh:206-207` |
| Advisory 1500-token target, WARN-only, never truncates | `cli/src/context_handoff.sh:173-193` |
| Post-compose crystalium ingest (OUT of atomos fence) | `cli/src/context_handoff.sh:242-250` |
| Verify verdict matrix + warn/block exit codes | `cli/src/verify_envelope.sh:45-47,81-85,114-118` |
| Verifier expects `envelope_version "2.0"` (drift vs composer's 1.0) | `cli/src/verify_envelope.sh:151-155` |
| Payload resolution + sibling fallback | `cli/src/verify_envelope.sh:177-187` |
| sha256 helpers (portable shasum/sha256sum) | `cli/src/lib_context.sh:91-113` |
| Externalize manifest object (for v0.2) | `cli/src/context_externalize.sh:116-138` |
| Pin set + advisory survival probe | `roster/pins.yaml`; ECM spec §3.2 |
| MCP catalogue entry shape (Phase 3) | `roster/mcps.yaml:210-279` (tonberry); `schemas/mcp-catalogue.schema.json` |
| MCP run template shape (Phase 3) | `cli/templates/mcp/tonberry.mcp.json.tmpl` |
