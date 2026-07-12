# atomos v0.2.0 — Build Spec (close the closed set)

**Plan:** `atomos-v0-2-manifest` · **Tier:** full (mechanical right-size, score 5) · **Planner:** RAMZA · **Executor:** Vivi
**Upstream:** FORGE ADR `docs/ARCHITECTURE.md` §2.4, §7 Phase 2 (settled 2026-07-07 — spec'd TO, not relitigated)
**Predecessor:** `docs/BUILD-SPEC.md` (v0.1.0, shipped) — its patterns are the template, not a starting point to redesign
**Oracle (READ-ONLY):** the eidolons nexus bash kernel — `cli/src/context_externalize.sh`, `cli/src/lib_context.sh`, `cli/src/context_handoff.sh`
**State:** `docs/BUILD-SPEC-v0.2.state.json` · **Frozen criteria:** `docs/BUILD-SPEC-v0.2.criteria.md` (SHA in Freeze Record)

---

## Framing

v0.1.0 shipped three of the four tools in the ADR's **closed** set and proved the
parity claim on the load-bearing artifact (brief + envelope, T1 byte/SHA + T2
envelope bytes). v0.2.0 does exactly two things (ADR §7 Phase 2):

- **Track H** — ship `compose_externalize_manifest`, the **4th and final** tool.
  It mirrors ONLY the manifest-build portion of `context_externalize.sh`
  (`MANIFEST_JSON`, lines 116-138) plus the file-floor writer (lines 140-146).
  It **never** reaches the kernel's durable-memory chain (lines 154-196) — that
  is permanently out of fence.
- **Track I** — expand the parity vectors: handoff edge cases (empty sections,
  oversize WARN, multiline `task_state`, split/dropped list entries) and a new
  golden tree for the manifest.

A thin **Track V** carries the version bump and the doc/CHANGELOG hygiene that a
release requires. Nothing else is in this release.

After v0.2.0 the tool surface is **complete and closed**. The next atomos change
of any size is Phase 3 (nexus rostering) — a *separate ESL change in the nexus
repo*, not an atomos build.

### The one invariant this release adds

> **M0 — the single-document rule.** Every `compose_externalize_manifest` call
> produces exactly **one** manifest document. `manifest` (the response object) is
> its object form; `manifest_sha256` is SHA-256 over its **canonical byte form**;
> the sidecar file (when written) contains **those exact bytes**. Nothing is
> hashed that is not returned; nothing is written that is not hashed. The digest
> is a pure function of the tool inputs — it does **not** depend on whether the
> sidecar was written.

Everything in Track H is downstream of M0.

### Mirror, don't fix (parity traps — bind every track)

The v0.1 traps (`from_version` is an input; `envelope_version` stays `"1.0"`;
`trace` is hardcoded) still hold. These are the **new** ones, all verified
against a live kernel run (see "Empirical capture" below):

| # | Trap | Rule | Kernel anchor |
|---|------|------|---------------|
| M1 | `created_at` is **inside** the hashed document | Unlike the brief (timestamp-free — ADR §3.1), the manifest **embeds** `created_at`. So `manifest_sha256` is timestamp-**dependent**, and there is **no T1-style timestamp-free anchor** for this tool. Every byte/SHA assertion needs a frozen-`created_at` vector. ADR §3.1's simplification does **not** generalize to the manifest. | `context_externalize.sh:137` |
| M2 | Newline-bearing list entries are **SPLIT** | `context_json_array` is `jq -R -s 'split("\n") \| map(select(length > 0))'`. One entry containing `"a\nb"` becomes **two** array elements. The **brief** does the opposite — it embeds the newline raw into the markdown line. Same input, opposite handling in the two tools. Mirror **both**. | `lib_context.sh:135-141`; `context_handoff.sh:112-124` |
| M3 | Empty-string entries **VANISH** | The same `select(length > 0)` filter drops `""` from the manifest arrays; the brief drops them via `[ -n "$x" ]` guards. Nothing survives ⇒ `[]`. | `lib_context.sh:140`; `context_externalize.sh:110-114` |
| M4 | `session_id: ""` ⇒ JSON `null` | Never the empty string. jq if/then/else in the manifest build. | `context_externalize.sh:136` |
| M5 | `ecm_version` is a hardcoded `"0.1"` **literal** | It is **not** read from `roster/context-policy.yaml` (which happens to agree today). Reading the policy file would be both a fence graze and a latent parity bug — if the nexus bumps its policy, the kernel's manifest still says `"0.1"`. | `context_externalize.sh:117,128` |
| M6 | `file_floor_reason` lands **LAST** | The kernel appends it with `jq '. + {file_floor_reason: $reason}'`; jq object addition appends a new key **after** the existing ones. It is the 11th key, after `created_at`. | `context_externalize.sh:143` |
| M7 | The kernel's **only** on-disk manifest always carries a reason | `_write_file_floor` is the kernel's sole manifest writer and it *unconditionally* appends `file_floor_reason` (`context_externalize.sh:143`). So a 10-key manifest matches **no kernel artifact on disk** — it matches only the kernel's in-memory `MANIFEST_JSON` object. Byte-parity with a kernel *file* is therefore reachable **only** if the reason is a caller input (Q3). Corroborating, not load-bearing: hard-coding the kernel's literal `"crystalium absent"` in non-test Go source would also trip `TestFenceNoForbiddenSurface`'s `\bcrystalium\b` deny-list — the fence forbids the constant independently. | `context_externalize.sh:140-146,199`; `internal/tools/registry_test.go:49,73,81` |

| M8 | **The regen script dies on bash 3.2 when a vector has no flags** | Under `set -u`, bash **3.2** aborts on `"${FLAGS[@]}"` when `FLAGS` is empty (`unbound variable`); bash 4+ does not. `defaults-only/input.json` carries *only* `ts`/`iso_ts`/`from_version` — every flag is conditional, so the array **is** empty, and the new `manifest-defaults` vector reproduces it in the new arm. This is a **live defect in shipped v0.1.0**: the drift-guard runs only on `ubuntu-latest` (bash 5), so CI is green while a macOS maintainer's regen aborts. `shellcheck` and a bad-construct grep provably cannot see it. Fix: **seed** the array (`FLAGS=(--json)`) so it is never expanded empty, and run the regen on `macos-latest` in CI (AC-I12/I13). | `scripts/regen-goldens.sh:52,81`; kernel idiom `context_handoff.sh:113` |

### Empirical capture (the design is not inferred, it was observed)

The kernel was driven in a scratch directory (no `.mcp.json` / `eidolons.mcp.lock`
⇒ `memory_probe_gated_in` false ⇒ warn + file-floor write ⇒ exit 0). Verbatim
output of `eidolons context externalize --anchor internal/compose/externalize.go:1
--anchor '' --symbol BuildManifest --decision 'multi\nline decision' --open-var 'q?'
--contains-tool-origin --json`:

```json
{
  "ecm_version": "0.1",
  "summary": "Eidolons context externalize checkpoint: identifier manifest recorded while cheap (ECM P1 policy operation).",
  "anchors": [
    "internal/compose/externalize.go:1"
  ],
  "symbols": [
    "BuildManifest"
  ],
  "decisions": [
    "multi",
    "line decision"
  ],
  "failed_approaches": [],
  "open_vars": [
    "q?"
  ],
  "contains_tool_origin": true,
  "session_id": null,
  "created_at": "2026-07-12T02:06:12Z",
  "file_floor_reason": "crystalium absent"
}
```

Read it closely: the empty `--anchor ''` is **gone** (M3); the multiline
`--decision` became **two** entries (M2); empty arrays are inline `[]`; populated
arrays are multi-line with elements at 4 spaces; `session_id` is `null` (M4);
`file_floor_reason` is last (M6); the file ends `"}\n"`.

---

## Right-sizing

Mechanical gate (`ramza-rightsize`, observable signals only):

| Signal | Value | Points |
|---|---|---|
| files-est | ~20 hand-authored/edited (fixtures generated on top) | 2 |
| new-dep | **none** — no YAML lib, no HTTP client, no crystalium SDK (their absence is the fence) | 0 |
| public-api | the closed MCP tool surface gains its 4th member — a published API | 1 |
| migration | none | 0 |
| security | the SHA-256 integrity semantics (M0) + a change to the structural fence guard | 1 |
| novel | no — every mechanism is a v0.1 pattern applied again (ordered emitter, sidecar write, golden regen, fence tests) | 0 |
| stakes | med (additive; the kernel stays canonical; fail-open) | 1 |
| **Score** | | **5 → full** |

Full tier ⇒ Stories, Confidence, Rejected Alternatives, Risks, EARS criteria with
a freeze, and a declared drift fence. Full tier also requires an **independent
critic** (`ramza-gate critic`, author ≠ checker) before phase A — see the
Freeze Record for the honest status of that gate.

---

## The six open questions — SETTLED

The ADR fixes the output core as `{ manifest: {…}, manifest_sha256: "hex" }` and
leaves the rest open. Each answer below is binding; each loser is in Rejected
Alternatives with its Explore score.

### Q1 — `manifest_sha256` covers WHICH bytes? → **The canonical document as written and returned** (M0)

`manifest_sha256` = SHA-256 over the **canonical byte form of the manifest
document** — the ordered, jq-exact rendering of exactly the object the response
returns, **including** `file_floor_reason` when the caller supplied one, **with**
the single trailing newline. Those same bytes are what the sidecar file receives.

The kernel computes no manifest SHA, so there is no oracle to copy. The choice is
governed by what makes the digest *useful and non-lying*: atomos's entire brand is
"the SHA-256 tool," and its sibling `verify_envelope` recomputes digests over
files. A digest that does not verify the file next to it is an integrity footgun.
So: **hash what you write; write what you return.** The digest is still a pure
function of the tool inputs — no wall clock (Q4), no filesystem state, and
independent of `write_sidecar` (AC-H06).

Consequence of M1: because `created_at` is inside the document, the digest is
timestamp-dependent. That is the kernel's shape, not a choice.

### Q2 — Does it write a sidecar? → **Yes. Default on.**

`write_sidecar: true` by default; `out_dir` default `.eidolons/.context`; filename
**`externalized-<ts>.json`** (kernel-exact, `context_externalize.sh:142`, `<ts>` =
`context_now_epoch_ts` format `YYYYMMDDTHHMMSSZ`). The response carries
`manifest_path` (`null` when `write_sidecar: false`).

The ADR's one-line output sketch (§2.4) is a **minimum** shape, not a prohibition:
§2.1 sets the precedent that a compose tool's response carries its path fields, and
§5.2 makes sidecar-on-disk the standing rule for composed artifacts. The kernel is
blunter still — `context_externalize.sh:26-27`: *"belt-and-suspenders: the
identifier manifest must survive somewhere."* atomos structurally cannot reach the
other landing place, so if it does not write the file, the manifest exists
**in-context only** — precisely the state ECM's P0 forbids. The file is not
optional in spirit; `write_sidecar: false` remains available as a dry preview,
exactly as it is for `compose_handoff`.

`ts` affects **only** the filename. It never enters the document, so
`manifest_sha256` does not depend on it.

### Q3 — `file_floor_reason`? → **Caller-supplied, optional, absent by default.**

**The argument is parity, not the fence.** The kernel's *only* manifest writer is
`_write_file_floor`, and it appends `file_floor_reason` **unconditionally**
(`context_externalize.sh:140-146`). Every manifest the kernel has ever written to disk
carries a reason. So if atomos could not accept one, it could **never byte-match any
kernel-written manifest** — the parity claim would be dead on arrival for this tool.
The reason must therefore be an **input** (M7). Given that, it is appended **last**
(M6) and, per M0, becomes part of the hashed document (Q1).

It is a *caller* input rather than an atomos constant for two reasons, in order:
1. **atomos cannot observe the reason.** It has no probe into durable memory and must
   never grow one. A tool that asserts an unobservable cause is lying with a straight
   face; a kernel-parity caller passes the kernel's own `"crystalium absent"`, and a
   caller with a different cause passes that instead.
2. **Corroboration:** hard-coding the kernel's literal string in `internal/compose/*.go`
   would trip `TestFenceNoForbiddenSurface`'s `\bcrystalium\b` deny-list anyway. The
   fence independently forbids the constant.

**The reason is free-form, not an enum.** The kernel itself has **two** literals —
`"crystalium absent"` and `"crystalium commit unreachable or timed out (…s budget)"`
(`context_externalize.sh:198-200`; the second fires when durable memory *is* reachable but
the commit fails or times out). atomos passes whatever string it is given through verbatim
(AC-H28). Nothing in this spec, the README, or the tool description may imply a closed set
of two — and **neither literal may appear in non-test Go source** (both carry the
deny-listed `\bcrystalium\b` token; they live in fixtures and tests only).

**When omitted, the document has exactly ten keys.** That is the shape most callers get,
so it is *not* left unpinned: it matches the kernel's in-memory `MANIFEST_JSON` object
(plus jq's trailing newline), and it is byte-covered by `core.json` — a golden derived
from the kernel's own output *by jq, the kernel's own emitter* (AC-H22/H23). What the
10-key form does **not** do is match a kernel file on disk; no such file exists. The spec
says so where it matters (Story S6) rather than implying a byte-identity that is false.

Both the README and the tool description must state the structural truth plainly:
*atomos emits the file-floor manifest and stops; durable memory is reached only through
the kernel verb or the memory MCP directly.*

### Q4 — `created_at` / `ts`? → **Caller inputs; wall-clock only at the server seam.**

Exactly the `compose_handoff` contract (`internal/server/server.go:71-86`,
`resolveTimestamps`), and for exactly the same reason: frozen-timestamp golden
vectors are impossible otherwise. `internal/compose` never reads a clock; the
server fills an omitted `created_at` (`2006-01-02T15:04:05Z`) and `ts`
(`20060102T150405Z`) from **one** `time.Now().UTC()` reading. AC-F04's grep is
extended to the new package set (AC-H18) so this stays mechanically enforced.

### Q5 — The fence tests must change. → **Yes. This spec IS the authorized revision. Tighten to exactly four; never loosen.**

`internal/tools/registry.go`'s comment ("if a fourth tool ever seems necessary,
STOP") and `TestToolSurfaceIsExactlyFenced`'s 3-name assertion are the guard-rail
working as designed. They are **not** a blocker here, because the ADR pre-authorized
this exact revision — §2's table lists `compose_externalize_manifest` with
"MVP? no (v0.2)", and §4.1 spells out the assertion's future shape verbatim:

> *"asserts the advertised list == `{compose_handoff, verify_envelope, verify_pins}`
> (+`compose_externalize_manifest` at v0.2) — no more, no less."*

So: update the registry comment to name the closed **four**, and tighten the test to
assert **exactly four**, element-wise. Do **not** relax it to a superset ("at least
these"), which would silently retire the closed-set guarantee. After this release
the honest comment is: *the set is closed at four; there is no fifth tool, and a
fifth would need a new ADR, not a new spec.*

### Q6 — Does `regen-goldens.sh` grow a manifest arm? → **Yes, and the oracle is driven through its file-floor side effect.**

The kernel has no `--ts`/`--created-at` flag and prints no manifest on stdout — its
only manifest artifact is the **file-floor file**, written when the memory backend is
absent or the commit fails (`context_externalize.sh:198-200`). That is not a
limitation to work around; it is the exact artifact atomos claims parity with.

Mechanics (per manifest vector):
1. `scratch=$(mktemp -d)`, `cd "$scratch"` — a bare dir has no `.mcp.json` /
   `eidolons.mcp.lock`, so `memory_probe_gated_in` fails ⇒ the kernel warns, writes
   the file-floor manifest, and exits 0. No docker, no network, no durable-memory
   call. (This is the same scratch-dir trick the v0.1 handoff arm already uses.)
2. `eidolons context externalize <flags from input.json> --json` ⇒ read
   `.file_floor_path` from the JSON verdict.
3. Copy that file **verbatim** to `fixtures/parity-manifest/<vector>/manifest.json`.
   Write `sha256` = digest of those bytes. Write `core.json` =
   `jq 'del(.file_floor_reason)' manifest.json` (jq — the kernel's own emitter — so
   the reason-less form is still an oracle-derived golden, not a hand-authored one)
   and `core.sha256` = its digest.
4. Back-fill the **captured** `created_at` (from `.created_at`), `ts` (from the
   filename `externalized-<ts>.json`) and `file_floor_reason` (from
   `.file_floor_reason`) into `input.json` — captured, never chosen, exactly as the
   handoff arm back-fills `ts`/`iso_ts`/`from_version`.
5. Diff-clean rule (mirrors the envelope arm): compare a freshly captured manifest
   against the committed golden with `.created_at` **stripped**. Identical ⇒ leave
   `manifest.json` / `core.json` / both `sha256` files / `input.json` untouched.
   Different ⇒ genuine semantic drift: rewrite the goldens and back-fill the new
   timestamp. This is what keeps
   `bash scripts/regen-goldens.sh && git diff --exit-code` honest run over run.

---

## Scope

**In scope:**

- **H1** — extract the jq-exact ordered-emitter primitives from
  `internal/ecl/envelope.go` into a new `internal/jsonx` package, and add an
  **array writer** (v0.1's emitter has no arrays — the manifest has five).
- **H2** — the manifest document type + its canonical `Marshal()` (`internal/compose/manifest.go`).
- **H3** — the tool handler `internal/compose/externalize.go`: defaults, kernel
  array normalization (M2/M3), SHA, sidecar write, `manifest_path`.
- **H4** — the authorized fence revision: registry entry #4, server registration +
  timestamp seam, `TestToolSurfaceIsExactlyFenced` tightened to exactly four.
- **I1** — 4 new handoff parity vectors + a per-vector `advisory.json` (the kernel's
  own `tokens_est`/`oversize`).
- **I2** — 3 new manifest parity vectors (`fixtures/parity-manifest/`).
- **I3** — `scripts/regen-goldens.sh`: manifest arm, index-driven flag building,
  advisory capture, extended PROVENANCE.
- **I4** — CI drift-guard diff extended to cover the new fixture tree.
- **V** — `ATOMOS_VERSION` → `0.2.0` (both compiled constants pinned to it),
  CHANGELOG, README, `fixtures/README.md`.

**Already satisfied — verify, do NOT respec.** ADR §7 Phase 2's third bullet
("multi-arch index + drift-guard CI, tonberry pattern") **shipped in v0.1.0**.
Verified in the tree, not assumed:

- `.github/workflows/release.yml` builds `linux/amd64` on `ubuntu-latest` and
  `linux/arm64` on the native `ubuntu-24.04-arm` runner, pushes by digest, merges a
  manifest list with `docker buildx imagetools create`, and captures the **index**
  digest into `dist/release-manifest.json` (`manifest_sha256`), attested.
- `.github/workflows/conformance.yml` has the `drift-guard` job: read the
  `NEXUS_COMMIT` pin from PROVENANCE → check out `Rynaro/eidolons` at that commit →
  re-run `regen-goldens.sh` → `git diff --exit-code fixtures/parity` → fail with an
  explicit "bump the pin deliberately" error. A weekly cron runs it unprompted.

The **only** carry-over from that bullet is that the drift-guard's diff path does
not yet cover the new manifest tree — that is AC-I09, one line.

**Non-goals (OUT — refuse if they drift in):**

- **Anything in the kernel's persist chain** (`context_externalize.sh:154-196`):
  no `--envelope` input, no durable-memory commit, no plan-checkpoint, no ingest.
  Permanently out of fence (ADR §0.1, §2.4).
- **The kernel's `--scope-project` flag** — it exists only to slug the durable-memory
  scope. It has no meaning inside the fence and is not an atomos input.
- **The budget ledger.** The kernel appends a line to
  `.eidolons/.context/budget-ledger.jsonl` on every externalize
  (`context_externalize.sh:202-223`) and reads `meter.json` to fill it. That is
  metering. atomos writes **one** file and nothing else (AC-H17). A caller that needs
  ledger continuity uses the kernel verb — this is a deliberate, documented subset,
  the same shape as v0.1's "brief+envelope, then stop".
- Any nexus edit whatsoever, including "fixing" the kernel.
- Nexus rostering (Phase 3, a separate ESL change in the nexus repo).
- A 5th tool. Any new dependency. A one-shot CLI op mode.
- Meter/policy/trigger/inject: unchanged from v0.1's fence.

---

## Stories

- **S6 — cheap checkpoint.** As the orchestrating host crossing into amber, I call `mcp__atomos__compose_externalize_manifest` and get the identifier manifest, its SHA, and a sidecar on disk — so the identifiers survive on either surface. **Byte-identity has a precondition:** the sidecar is byte-identical to what `eidolons context externalize` writes **when I pass the kernel's `file_floor_reason`** (its sole writer always appends one — M7). With the reason omitted I get the 10-key core document, which matches the kernel's in-memory `MANIFEST_JSON` object, not any kernel file on disk. Both forms are byte-pinned against kernel-derived goldens (AC-H03/H04, AC-H22/H23); neither claim is stronger than the artifact behind it.
- **S7 — one document, one hash.** As a receiver handed `manifest_path` + `manifest_sha256`, I recompute the digest of the file and it matches. Always. No annotation, no dry-run flag, and no code path changes that.
- **S8 — closed set, enforced.** As a fence auditor, `tools/list` shows exactly four tools, and the registry test goes red the moment a fifth appears — the closed-set claim is a test, not a comment.
- **S9 — edge-case parity.** As the parity maintainer, the golden set now pins the kernel's *weird* behavior (empty sections, oversize WARN, multiline `task_state`, entries that split, entries that vanish), so a kernel edge-case change surfaces as a red drift-guard instead of a silent divergence.

---

## Approach

### Track H1 — `internal/jsonx` (the shared jq-exact emitter)

`internal/ecl/envelope.go` already contains a hand-rolled emitter that reproduces
`jq -n` bytes exactly (2-space indent, insertion order, jq escaping — quote,
backslash, `\b \f \n \r \t`, `\u00XX` for other control chars and DEL; `/`, `<`,
`>`, `&` and non-ASCII UTF-8 passed through verbatim; single trailing newline).
The manifest needs the same escaper. **Two escapers would drift**, and a drifted
escaper is a silent parity break — so move the primitives (`jsonString`, `indent`,
`writeStr`, `writeRaw`, `writeObjOpen`, `writeObjClose`) **verbatim** into
`internal/jsonx` and have `internal/ecl` call them.

Add one primitive: **`writeArr(b, level, key, items []string, comma bool)`** — jq's
array formatting, confirmed empirically:

```
  "failed_approaches": [],                 <- empty: inline, no newline inside
  "anchors": [
    "internal/compose/externalize.go:1"    <- populated: one element per line, indent level+1
  ],                                       <- closing bracket at the key's indent
```

This refactor is **behavior-preserving by definition**: `TestEnvelopeT2Parity`
(v0.1, byte-equality against the kernel envelope goldens) is its guard and must stay
green (AC-H20). Move the code; do not rewrite it.

**`internal/jsonx` is an additive deviation from ADR §6's layout sketch — declared, not
smuggled.** §6 is a *layout* (it lists `compose/externalize.go` for this very tool), not
a fence; adding a package under `internal/` relitigates no decision. The alternative —
**exporting the primitives from `internal/ecl` in place** (no code moves, so T2 cannot
regress by construction) — was scored and lost, **71.5 vs 79.0**: it makes package `ecl`,
whose doc declares it an *ECL envelope* emitter, the home of an *ECM manifest* emitter,
and a package comment that lies is exactly the failure mode the fence tests exist to
prevent. The move's risk is real but bounded, and it is bounded *by a test that already
exists*.

### Track H2 — the manifest document (`internal/compose/manifest.go`)

Canonical byte form, key order fixed to `context_externalize.sh:127-138`'s `jq -n`
insertion order (`?` = present only when the caller supplied a reason):

```
{
  "ecm_version": "0.1",
  "summary": <string>,
  "anchors": <array>,
  "symbols": <array>,
  "decisions": <array>,
  "failed_approaches": <array>,
  "open_vars": <array>,
  "contains_tool_origin": <bool>,
  "session_id": <string|null>,
  "created_at": "<iso8601>",
  "file_floor_reason": <string>            ? last, only when supplied (M6)
}
```
…plus a single trailing newline. `Marshal()` returns these bytes; the result struct
carries them as `ManifestBytes []byte json:"-"` — the *same* bytes that are hashed
and written, exactly as `HandoffResult.EnvelopeBytes` does for the envelope (never a
re-serialization, or the parity test stops testing the production path).

**Array normalization (the kernel's `context_json_array`, M2/M3).** For each input
list: join the elements with `"\n"`, append `"\n"`, split on `"\n"`, drop every empty
string. Equivalently: for each element, split it on `"\n"` and append the non-empty
parts. Only `"\n"` separates — `"\r"` is not special (a `"\r\n"` entry keeps a
trailing `"\r"`). Nothing survives ⇒ `[]`.

**Defaults.** `summary == ""` ⇒ the kernel's sentence, verbatim
(`context_externalize.sh:100`):
`"Eidolons context externalize checkpoint: identifier manifest recorded while cheap (ECM P1 policy operation)."`
`session_id == ""` ⇒ literal `null` (M4). `ecm_version` ⇒ the literal `"0.1"`, read
from no file (M5). `contains_tool_origin` ⇒ `false`.

### Track H3 — the tool (`internal/compose/externalize.go`)

Pure function `(ManifestInput) → (ManifestResult, error)`. No clock (Q4), no probe,
no config read.

```jsonc
// input
{
  "summary": "string",                 // "" => the kernel default sentence
  "anchors": ["path:line", ...], "symbols": [...], "decisions": [...],
  "failed_approaches": [...], "open_vars": [...],
  "contains_tool_origin": false,
  "session_id": "string",              // "" => JSON null in the document
  "created_at": "2026-07-12T02:06:12Z",// PARITY-CRITICAL caller input; server seam fills when omitted
  "ts": "20260712T020612Z",            // sidecar FILENAME ONLY — never enters the document
  "file_floor_reason": "string",       // optional; omitted => absent (Q3)
  "write_sidecar": true,               // default true
  "out_dir": ".eidolons/.context"      // default
}
// output
{
  "manifest": { /* the object form of the ONE document */ },
  "manifest_sha256": "hex",            // SHA-256 over the canonical bytes (M0)
  "manifest_path": ".eidolons/.context/externalized-<ts>.json"  // null when write_sidecar=false
}
```

Sidecar: `os.MkdirAll(out_dir, 0o755)` fail-soft (mirrors `context_sidecar_dir`),
then one `os.WriteFile` of `ManifestBytes`. **One file. Stop.** No ledger, no meter
read, no policy log, no durable-memory call — the kernel does all four and every one
of them is out of fence.

### Track H4 — the authorized fence revision

- `internal/tools/registry.go`: add the 4th descriptor. **Rewrite the package
  comment** — it currently says a fourth tool means STOP; it must now say the set is
  closed at **four** and that a fifth would require a new ADR. The description string
  must state the two structural truths: file-floor manifest only, and durable
  persistence is not reachable from this surface. (Mind the deny-list while writing
  it — see Executor Notes.)
- `internal/server/server.go`: register the tool; extend the timestamp seam to fill
  `created_at`/`ts` from one clock read; bump `Version` to `0.2.0`.
- `internal/tools/registry_test.go`: `want` becomes the 4-name set; the length
  assertion stays **exact equality**; add `internal/jsonx` to the `time.Now` package
  scan (AC-H18).

### Track I — parity vectors

**New handoff vectors** (`fixtures/parity/`), added to `parityVectors` in
`internal/compose/parity_test.go`:

| Vector | What it pins |
|---|---|
| `empty-sections` | `--anchor`/`--symbol`/`--decision` all absent ⇒ `## Identifiers` renders **empty with no `(none recorded)` fallback** (the one section that has none) |
| `oversize-brief` | a brief over the 1500-token advisory ⇒ `oversize: true`, `brief_md` **untruncated** and byte-equal to the golden |
| `multiline-task-state` | `objective` = prefix + **first non-blank line**; the body keeps the full multiline task state |
| `empty-list-entries` | empty-string entries dropped **and** a newline-bearing entry embedded **raw** in the brief (the mirror image of M2 — the same input that *splits* in the manifest) |

`contains_tool_origin: true` is **already covered** by v0.1's `narrative-open-vars`
vector (verified in `fixtures/parity/narrative-open-vars/input.json`) — the ADR §7
bullet is satisfied for the handoff; the manifest gets its own coverage below.

**`advisory.json`** (all handoff vectors): `{"tokens_est": N, "oversize": bool}`
captured from the kernel's own `--json`. It is timestamp-free, so it is diff-clean,
and it does two jobs at once: it grounds the oversize assertion in the **oracle**
rather than in atomos's own arithmetic, and it **auto-guards the hardcoded 1500**
(v0.1 Rejected Alternative #6) — if the nexus ever changes
`limits.handoff_brief_target_tokens` (today `1500` in `roster/context-policy.yaml`),
the kernel's `oversize` flips, the golden drifts, and CI goes red. One mechanism,
two properties.

**New manifest vectors** (`fixtures/parity-manifest/<vector>/`), files
`input.json`, `manifest.json`, `sha256`, `core.json`, `core.sha256`:

| Vector | What it pins |
|---|---|
| `manifest-defaults` | no inputs at all ⇒ default summary, five inline `[]`, `session_id: null`, `contains_tool_origin: false` |
| `manifest-populated` | every list populated, explicit `session_id`, and a **multiline `summary`** (`--arg`, so it is *not* split — the discriminator against the array path, and it exercises `\n` escaping in the emitter) |
| `manifest-tool-origin` | `contains_tool_origin: true`, a **newline-bearing** entry (must SPLIT, M2) and an **empty-string** entry (must VANISH, M3) |

**`scripts/regen-goldens.sh`** (stays bash 3.2): grows the manifest arm (Q6), and its
array-flag building changes from a line-reading loop to **index-driven** extraction:

```bash
n="$(jq -r '.anchors | length // 0' "$input")"
i=0
while [ "$i" -lt "$n" ]; do
  v="$(jq -r --argjson i "$i" '.anchors[$i]' "$input")"
  FLAGS+=(--anchor "$v")            # NO [ -n "$v" ] guard — "" must reach the kernel
  i=$((i + 1))
done
```

This is not cosmetic, and the reason is subtler than "the script is sloppy" — it is the
**most dangerous thing in this build**. The current loop (`while IFS= read -r v` +
`[ -n "$v" ] &&`) cannot feed the oracle an empty entry (the guard drops it) or a
newline-bearing entry (`read` splits it into **two** flags). For the **manifest** arm that
mangling happens to be harmless — `context_json_array` re-splits on `\n`, so one
`--decision $'a\nb'` and two `--decision` flags produce the *same* array. For the **brief**
arm it is **not**, and the kernel was driven to prove it:

| kernel invocation | brief bytes |
|---|---|
| one `--decision $'multi\nline decision'` | `- decision: multi` ⏎ `line decision`  — **ONE** bullet, newline embedded raw |
| two flags: `--decision multi --decision 'line decision'` | `- decision: multi` ⏎ `- decision: line decision` — **TWO** bullets |

Different bytes. So the `empty-list-entries` vector — whose entire purpose is to pin
*"newline embedded RAW in the brief, the mirror image of M2"* — would be captured through
the mangling and its golden would show the brief **SPLIT**. That golden is not merely
stale, it is **actively wrong**: the parity test goes red, and the obvious "fix" is to make
atomos's brief split newlines — which is precisely backwards, and would break byte-parity
with the real kernel while turning the suite green. Index-driven extraction round-trips
both cases. The three existing vectors contain neither an empty nor a multiline entry, so
their goldens stay byte-identical (the diff-clean contract holds).

**Seed the flag array (M8).** `FLAGS=(--json)` before any conditional flag is appended, in
**both** arms. Under `set -u`, bash 3.2 aborts on `"${FLAGS[@]}"` when the array is empty
— which `defaults-only` and `manifest-defaults` both produce — and the kernel accepts
`--json` in any position, so seeding costs nothing and removes the expansion entirely.
(The kernel's own idiom, `[ "${#ARR[@]}" -gt 0 ]` at `context_handoff.sh:113`, is the
equivalent guard for any *other* array the script grows.) CI must **execute** this, not
lint it: the drift-guard job runs on a `[ubuntu-latest, macos-latest]` matrix, and macOS's
system bash **is** 3.2 (the same job shape the nexus uses to catch exactly this class —
CLAUDE.md §"Bash 3.2 compatibility"). AC-I12/I13.

*Fixture-authoring rule, mechanically enforced (AC-I14):* no list element may **end**
with a newline — `$( )` strips trailing newlines, so such an element cannot be captured
faithfully and would yield a **silently wrong golden** (the exact failure class AC-I08
exists to prevent). Prose is not a guard, so the script must refuse the fixture:

```bash
jq -e '[.anchors[]?,.symbols[]?,.decisions[]?,.failed_approaches[]?,.open_vars[]?]
       | map(select(endswith("\n"))) | length == 0' "$input" >/dev/null || {
  echo "regen-goldens: [$vector] fixture list element ends with a newline — cannot be captured faithfully" >&2
  exit 1
}
```

atomos's handler still handles such an element correctly per the split rule; a **unit**
test covers that, never a golden.

PROVENANCE gains `MANIFEST_VECTORS=` and a `CAPTURED_VIA` line naming both kernel
verbs; it stays a single stamp for a single regen run (`ECM_VERSION` + `NEXUS_COMMIT`
unchanged in shape, so `TestGoldenProvenanceStamp` still holds).

CI: the drift-guard's check becomes `git status --porcelain --untracked-files=all
fixtures/parity fixtures/parity-manifest` (non-empty output ⇒ fail), **not**
`git diff --exit-code` (AC-I09). `git diff` cannot see **untracked** files, so a golden the
executor forgot to `git add` leaves CI green over a file that is not in the repo — and v0.2
creates an entirely **new** tree (`fixtures/parity-manifest/`), which is exactly where that
failure lands.

### Track V — release hygiene

`ATOMOS_VERSION` → `0.2.0`. Both compiled constants (`cmd/atomos.Version` and
`internal/server.Version`) must equal it — v0.1 pins only the former to the file, so
add the twin test (AC-V02); a version that disagrees with itself is exactly the class
of drift this repo exists to prevent. CHANGELOG `[0.2.0]`; README's tool table gains
the 4th tool and the file-floor-only truth; `fixtures/README.md` documents the
manifest tree, M0, and the `core.json` derivation.

**Build order for Vivi:** H1 (jsonx, T2 stays green) → H2 → H3 → I2 manifest vectors
(capture goldens the moment the emitter exists — red/green against the kernel from
hour one) → H4 (registry/server/fence) → I1/I3 (handoff edge vectors + regen) →
I4 → V.

---

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

## Drift Fence

Declared in `BUILD-SPEC-v0.2.state.json` via `ramza-drift --declare`, checked against
the executed diff at review time (`ramza-drift --state … --range`).

```yaml
files_allowed:            # the atomos repo, repo-relative
  - 'cmd/*'
  - 'internal/*'
  - 'fixtures/*'
  - 'scripts/*'
  - '.github/*'
  - 'docs/*'
  - 'ATOMOS_VERSION'
  - 'README.md'
  - 'CHANGELOG.md'
files_NOT_declared:       # deliberate: a change here is DRIFT, and that is the point
  - 'go.mod'              # v0.2 adds NO dependency. A go.mod diff must fail the fence.
  - 'go.sum'              # same.
  - 'Dockerfile'          # the build does not change in v0.2.
  - 'LICENSE' '.gitignore' '.dockerignore'
files_forbidden:          # the eidolons nexus kernel — READ-ONLY oracle
  - '<any path under the Rynaro/eidolons checkout>'
  -  # named explicitly because they are the oracle:
  - 'eidolons/cli/src/context_externalize.sh'
  - 'eidolons/cli/src/lib_context.sh'
  - 'eidolons/cli/src/context_handoff.sh'
  - 'eidolons/cli/src/verify_envelope.sh'
  - 'eidolons/roster/*'
```

`go.mod`/`go.sum` are **deliberately outside** the declared scope: the
no-new-dependency constraint is thereby enforced *mechanically* by `ramza-drift`
rather than by good intentions. If the executor genuinely needs to touch them, that
is an amendment (`ramza-freeze --amend --reason`), not a quiet edit.

atomos **NEVER** edits the nexus. If parity fails, atomos is wrong. If the *kernel*
is wrong, that is a separate nexus ESL change proposed through the normal lifecycle —
never a drive-by edit from this campaign. Golden-pin bumps happen in
`fixtures/parity/PROVENANCE` inside the atomos repo only.

---

## Rejected Alternatives

Scored with `ramza-score --rubric explore` (totals recorded in the state file's
`gates[]`); the winner was **H-A, 83.0** — the single-document rule (M0).

1. **H-B — hash the 10-field core, write the 11-field file (55.5).** The reported
   `manifest_sha256` would not verify the file next to it. In a tool whose sibling
   verb *recomputes SHA-256 over files*, that is an integrity footgun that will one
   day be read as `tamper`. Rejected on correctness.
2. **H-C — response-only, no sidecar; the literal ADR §2.4 shape (68.0).** Simplest
   and cheapest, and it is exactly what the ADR sentence says. Rejected because ECM's
   P0 forbids in-context-only artifacts, ADR §5.2 makes sidecar-on-disk the standing
   rule for composed output, and the kernel says it outright: *"the identifier
   manifest must survive somewhere"* (`context_externalize.sh:26-27`). The ADR shape
   is a minimum, and §2.1's precedent (a compose tool returns its path fields)
   governs the rest.
3. **H-D — atomos authors a fixed `file_floor_reason` (57.0).** It invents a fact
   atomos cannot observe (it has no probe and must never have one), and the natural
   wording of that fact trips the fence deny-list (M7). Rejected twice over.
4. **H-E — wrap the manifest in an ECL envelope for symmetry with `compose_handoff` (46.0).**
   Tempting, and wrong: the kernel emits **no** envelope for the manifest. Minting
   `ecm/externalize-manifest@0.1` would have atomos *defining* ECL surface, which ADR
   §5.3 forbids — atomos is a faithful emitter of shapes the kernel already writes,
   never an author of new ones.
5. **Hashing the bash `MANIFEST_JSON` variable's exact bytes** (jq output minus the
   trailing newline that `$( )` strips). That byte string is an artifact of shell
   command substitution, not a document. The document is what lands on disk.
6. **Reading `roster/context-policy.yaml` for `ecm_version`.** The kernel hardcodes
   `"0.1"` (M5); reading the policy is a fence graze *and* a latent parity bug. Same
   reasoning as v0.1's Rejected Alternative #6 (the 1500-token target).
7. **Porting any part of the persist chain** — commit, plan-checkpoint, ingest,
   budget ledger, meter read. Permanently out of fence (ADR §0.1, §2.4). The file-floor
   manifest is a strict, deliberate subset; that is the whole design.
8. **Loosening `TestToolSurfaceIsExactlyFenced` to a superset assertion** ("at least
   these four") to make future additions painless. That retires the closed-set
   guarantee — the pain *is* the feature. Tighten to exactly four (Q5).
9. **Duplicating the jq string escaper into a second emitter.** Two escapers drift,
   and a drifted escaper is a silent byte-parity break. Extract `internal/jsonx`;
   share one (H1).
9b. **Exporting the emitter primitives from `internal/ecl` in place** instead of
   extracting `internal/jsonx` — scored **71.5** vs jsonx's **79.0**. Genuinely lower
   risk (nothing moves, so T2 cannot regress by construction) and it avoids the ADR §6
   layout deviation. Rejected because it makes package `ecl` — whose doc declares it an
   *ECL envelope* emitter that "does not define, validate against, or extend" anything
   else — the home of an *ECM manifest* emitter. A package comment that lies is the
   failure mode the fence tests exist to catch, and the move's risk is already bounded
   by a test that exists today (AC-H20).
9c. **Omitting `file_floor_reason` entirely** (no input, no field, ever) — scored
   **60.5**. Simplest and cleanest-looking. Fatal: the kernel's *only* manifest writer
   appends the reason unconditionally (M7), so an atomos that cannot emit it can never
   byte-match **any** kernel-written manifest; the parity claim would collapse to
   *matches a document the kernel never wrote*.
10. **Teaching the kernel a `--ts` flag or a manifest `--json` emit** so the oracle is
    easier to drive. That is a *nexus edit*. Absolutely not. Drive the oracle through
    the file-floor side effect it already has (Q6).
11. **Hand-authoring the manifest goldens** from the byte template in this spec. The
    goldens' entire value is that the *kernel* produced them; a hand-written golden
    proves only that the executor read the spec.

---

## Risks

- **R1 — jq array formatting is new emitter surface.** v0.1's envelope has no arrays;
  the manifest has five. *Mitigation:* the exact shape is captured empirically in
  Framing (inline `[]`, elements at indent+1, bracket at the key's indent) and pinned
  by kernel-captured goldens the moment H2 exists.
- **R2 — the `internal/jsonx` extraction could regress T2 envelope parity.**
  *Mitigation:* move the primitives **verbatim**; `TestEnvelopeT2Parity` is the
  refactor guard and is an explicit AC (AC-H20). Do H1 first, on green.
- **R3 — the `empty-list-entries` *brief* assertion is vacuously satisfiable** (both
  kernel and atomos drop empties, so the output cannot distinguish "dropped" from
  "never passed"). *Mitigation:* accepted honestly — that vector is regression
  insurance, not oracle discrimination. The **manifest**'s SPLIT behavior (M2) *is*
  discriminating and is the real test, and AC-I08 makes the script feed the oracle
  faithfully regardless.
- **R4 — regen could capture goldens from the wrong inputs.** The current line-based
  flag loop silently drops `""` and splits newline-bearing entries. *Mitigation:*
  index-driven extraction (Track I), AC-I08, plus the fixture rule (no element ends
  with a newline).
- **R5 — `created_at` in the hash means every regen run mints a new SHA** (M1).
  *Mitigation:* the diff-clean rule compares with `.created_at` stripped and only
  rewrites goldens on genuine semantic drift — the same trick the envelope arm already
  uses for `message_id`/`trace.ts`.
- **R6 — deny-list false-trips on innocent naming.** The new code is *about* the thing
  the fence forbids, so the words are on the tip of the tongue. *Mitigation:* Executor
  Notes name the trap explicitly and point at `internal/ecl/envelope.go:11-13`, which
  already solves it (it writes "no durable-storage call, no session-budget read, no
  rule-table evaluation" precisely to stay clean).
- **R7 — the nexus could change `handoff_brief_target_tokens` away from 1500.**
  *Mitigation:* `advisory.json` turns that into a red drift-guard instead of a silent
  divergence.
- **R9 — the regen script aborts on bash 3.2 when a vector has no flags** (M8) — a
  **live defect in shipped v0.1.0**, not a v0.2 speculation: `defaults-only` already
  produces an empty flag array, and `manifest-defaults` reproduces it in the new arm.
  CI never saw it because the drift-guard runs only on `ubuntu-latest`. *Mitigation:*
  seed the array (AC-I13) **and** execute the regen on `macos-latest` (AC-I12) — a static
  lint provably cannot detect this, so the gate must run the script, not read it.
- **R10 — a VERIFY that names a not-yet-written Go test passes vacuously.**
  `go test ./pkg -run TestNope` prints `[no tests to run]` and **exits 0**. Every AC in
  this spec that cites a new test would otherwise be green before a line of code exists.
  *Mitigation:* every such VERIFY is now `grep -rq 'func TestX(' <pkg> && go test …` — the
  criteria file is the executable contract, and a gate that cannot fail is worse than no
  gate at all.
- **R11 — a golden that was never `git add`-ed.** `git diff --exit-code` is blind to
  untracked files, so the drift-guard would pass over a fixture that does not exist in the
  repo — worst exactly in v0.2, which creates a new tree. *Mitigation:* AC-I09 switches the
  check to `git status --porcelain --untracked-files=all`.
- **R8 — callers may expect ledger/meter continuity** because the kernel verb writes
  more than atomos does. *Mitigation:* documented in README + the tool description as
  a deliberate subset; the kernel verb remains the always-canonical path.

---

## Confidence

Scored via `ramza-score --rubric confidence` (recorded in the state file's `gates[]`):
pattern_match 92 (v0.1 is a shipped, near-isomorphic template *in this repo*, and the
kernel behavior was captured empirically, not inferred) ·
requirement_clarity 88 (the ADR is anchor-dense; the six open questions are now settled
with stated rationale; residual judgment lives in the `core.json` derivation) ·
decomposition_stability 85 (H and I are near-independent, but the `jsonx` extraction
touches a parity-critical package and H3↔I2 couple through the goldens by design) ·
constraint_compliance 94 (fence, parity, no-new-dep, `time.Now`, bash 3.2 are each
carried by a mechanical AC — including the deny-list-vs-reason-string trap).
**Weighted: 89.75 → AUTO_PROCEED.**

---

## Executor Notes (Vivi)

- **READ THIS FIRST — the registry guard-rail is not stopping you.**
  `internal/tools/registry.go` says: *"The set is closed: compose_handoff,
  verify_envelope, verify_pins. Extending it is a spec revision, never a drive-by
  addition — if a fourth tool ever seems necessary, STOP."* **This spec IS that
  authorized spec revision.** The ADR planned it from day one: §2's table lists
  `compose_externalize_manifest` as "MVP? no (**v0.2**)", and §4.1 pre-writes the
  assertion — *"== {compose_handoff, verify_envelope, verify_pins}
  (+compose_externalize_manifest at v0.2) — no more, no less."* Add the fourth tool,
  **rewrite the comment** so it no longer lies (the set is now closed at four), and
  **tighten** `TestToolSurfaceIsExactlyFenced` to exactly four. Do **not** loosen it to
  "at least" — a superset assertion silently retires the closed-set guarantee, which is
  the one thing this repo's fence is for.
- **The fence deny-list will bite you while writing *about* the fence.**
  `TestFenceNoForbiddenSurface` greps non-test Go source for `\bcrystalium\b`,
  `\bpersist\b`, `\bingest\b`, `\brecall\b`, `\bpolicy\b`, `\bmeter\b`, `\bzone\b`,
  `\butilization\b`, `\bcompact\(`. Your new package doc, your registry description, and
  your comments all *want* to say "never calls crystalium / never persists". They cannot.
  The precedent is already in the tree — `internal/ecl/envelope.go:11-13` says
  *"no durable-storage call, no session-budget read, no rule-table evaluation, no host
  prompt-surface write"* and passes. Copy that register. The kernel's literal
  `"crystalium absent"` string may appear **only** in `fixtures/**` and `*_test.go`
  (both skipped by the fence walk) — never as a Go constant.
- **Kernel is READ-ONLY ground truth.** Drive it, never edit it:
  `cd "$(mktemp -d)" && EIDOLONS_NEXUS=<checkout> bash <checkout>/cli/eidolons context externalize … --json`.
  A bare scratch dir is what makes the kernel take the file-floor branch — that is the
  artifact you are mirroring, not a degraded mode.
- **Hash what you write; write what you return (M0).** One document per call. The
  result struct carries `ManifestBytes []byte json:"-"` — the *same* bytes that are
  hashed and written, exactly as v0.1 does with `EnvelopeBytes`. If you ever
  re-serialize the object to compute the digest, the parity test stops testing the
  production path and starts testing itself.
- **`json.MarshalIndent` is still forbidden.** It sorts keys and HTML-escapes; the
  kernel's `jq -n` emits insertion order. Everything goes through the ordered emitter.
  Extract `internal/jsonx` **first**, on green, and let `TestEnvelopeT2Parity` prove the
  move was behavior-preserving before you build anything on top of it.
- **The two tools disagree about newlines, and that is the single most dangerous thing
  in this build** (M2). A list entry containing `"a\nb"` becomes **two array elements**
  in the manifest and **one raw markdown line containing a newline** in the brief. Same
  input, opposite handling. Both are the kernel. Mirror both; fix neither. If a brief
  parity test goes red showing a *split* bullet, the bug is in `regen-goldens.sh`'s flag
  building (it fed the kernel two flags), **never** in atomos's brief — "fixing" atomos to
  split newlines would turn the suite green while breaking parity with the real kernel.
  That is the trap; AC-I08 exists to keep you out of it.
- **You cannot paste this spec's prose into a Go comment.** `TestFenceNoForbiddenSurface`
  greps for `\bmeter\b`, `\bpolicy\b`, `\bzone\b`, `\butilization\b`, `\bpersist\b`,
  `\bingest\b`, `\brecall\b`, `\bcrystalium\b`. AC-H17's own THEN ("no budget ledger,
  no **meter** read, no **policy** log") and the path `roster/context-policy.yaml` are both
  deny-listed strings. Cite kernel line numbers rather than a kernel filename when the
  filename itself carries a forbidden token, and describe the fence the way
  `internal/ecl/envelope.go:11-13` already does.
- **Timestamps are inputs** (Q4). `internal/compose` never calls `time.Now` — the grep
  in AC-H18 covers `internal/jsonx` too, so keep the new package clean.
- **`scripts/regen-goldens.sh` is bash 3.2** (macOS system bash): no associative arrays,
  no `${var,,}`, no `readarray`/`mapfile`, no `&>>`. The index-driven flag loop is
  deliberate — the line-reading loop *cannot* feed the oracle an empty or multiline
  entry, which would make the new vectors lie.
- **`file_floor_reason` is free-form; do not model it as an enum.** The kernel has two
  literals (`context_externalize.sh:198-200`) and a caller may legitimately pass a third.
  Pass it through verbatim (AC-H28). Neither kernel literal may appear in non-test Go — both
  carry the deny-listed token.
- **No new dependency.** `github.com/modelcontextprotocol/go-sdk` only. `go.mod` and
  `go.sum` are deliberately **outside** the declared drift scope — if you touch them,
  `ramza-drift` will flag it, and it is meant to.
- **Fail-open everywhere.** `mkdir -p` is fail-soft; oversize never truncates; the only
  hard errors are malformed inputs.
- **Do not respec the CI.** Multi-arch index digest + drift-guard already shipped in
  v0.1.0 (verified). The only CI change in v0.2 is one line: the drift-guard's diff must
  also cover `fixtures/parity-manifest` (AC-I09).

---

## Critique record (refine cycle 1)

Independent critic **`ramza-critic-opus48`** (author ≠ checker, recorded via
`ramza-gate critic`): **APPROVE-WITH-FIXES** — 3 MUST, 8 SHOULD, 3 NICE. The critic
re-drove the kernel and confirmed M0/M1/M2/M3/M7, AC-I08 and the "§7 Phase-2 CI already
shipped" finding. One refine pass (`ramza-gate refine`, cycle 1/3) landed the fixes:

| Finding | Disposition | Where |
|---|---|---|
| **F1** (MUST) — regen dies on bash 3.2 when the flag array is empty; a **live defect in shipped v0.1.0**; `shellcheck` cannot see it | **ACCEPTED** | Trap **M8**; Track I (seed `FLAGS=(--json)`, macOS drift-guard matrix); **AC-I12** (executed on bash 3.2), **AC-I13** (never expanded empty); **R9** |
| **F2** (MUST) — the default 10-key document had zero byte/SHA coverage | **ACCEPTED** | **AC-H22** (bytes vs `core.json`), **AC-H23** (SHA vs `core.sha256`) |
| **F3** (MUST) — Story S6's byte-identity claim is false under default inputs | **ACCEPTED** | **S6** rewritten with the precondition stated; Q3 |
| **F5** (SHOULD) — a VERIFY naming a not-yet-written test is *vacuously green* (`go test -run TestNope` exits 0) | **ACCEPTED, class-wide** | every test-citing VERIFY is now `grep -rq 'func TestX(' <pkg> && go test …`; **R10** |
| **F6** (SHOULD) — Q3's fence argument is a rationalization; lead with parity | **ACCEPTED** | **Q3** rewritten (parity first, fence demoted to corroboration); **M7** retitled; **RA 9c** (omit-entirely) scored **60.5** and rejected |
| **F9** (SHOULD) — AC-H18's `! grep -rn` on a missing dir exits 2 ⇒ vacuous pass | **ACCEPTED** | **AC-H18** now binds to `TestNoTimeNowInHandlerPackages` + asserts `jsonx` is in its scan list |
| **F10** (SHOULD) — deny-listed words (`meter`, `policy`) appear in this spec's own prose and in a kernel path | **ACCEPTED** | Executor Notes: "you cannot paste this spec's prose into a Go comment" |
| **F11** (SHOULD) — `internal/jsonx` is an unscored ADR §6 deviation | **ACCEPTED** | Track H1 declares the deviation; the in-place alternative scored **71.5** vs **79.0** (**RA 9b**) |
| **AC-I08 framing** — the regen mangling is behaviour-preserving for the *manifest* arm but **not** the *brief* arm | **ACCEPTED** | Track I now carries the kernel-driven one-flag-vs-two-flag byte table; Executor Notes call it the most dangerous trap in the build |
| **F4** (SHOULD) — M0's "nothing is hashed that is not returned" was **untested**; a second serialization path could silently diverge | **ACCEPTED** | **AC-H24** (the response object is *decoded from* the hashed bytes and deep-equals them) |
| **F7** (SHOULD) — the `\r` rule (`"a\r\nb"` ⇒ `["a\r","b"]`) had zero coverage | **ACCEPTED** (re-confirmed against the kernel by the author) | **AC-H25** |
| **F8** (SHOULD) — the fixture-authoring rule was prose only | **ACCEPTED** | regen now **refuses** the fixture (`jq -e … endswith("\n")`); **AC-I14** |
| **F12** (NICE) — grep-only VERIFYs that cannot fail meaningfully | **ACCEPTED** | **AC-I07** now claims only what its grep proves (AC-I06 named as the real guard); **AC-I09** strips YAML comments before grepping; **AC-V03/V04/V05** VERIFYs now cover every conjunct in their THEN |
| **F13** (NICE, promoted to SHOULD) — `git diff --exit-code` is blind to **untracked** files, so a forgotten `git add` in the brand-new `fixtures/parity-manifest/` tree leaves CI green over files that are not in the repo | **ACCEPTED** | **AC-I09** switches the drift-guard to `git status --porcelain --untracked-files=all`; **R11** |
| **F14** (NICE) — coverage holes: non-default `out_dir`, the malformed-input/write-error path, and the kernel's **second** reason literal | **ACCEPTED** | **AC-H26** (custom `out_dir`), **AC-H27** (write error surfaces; `MkdirAll` fail-soft, the write is not), **AC-H28** (`file_floor_reason` is free-form, never a closed set of two) |

**Refine cycle 2 was DENIED by the gate** — `ramza-gate refine` is enterable from **T**
only, and the plan had already advanced to **A** (`DENY: refine is entered from T only
(current: A)`). That is the state machine working, not a workaround: the sanctioned
post-Assemble mechanism for a criteria change is the **hash-chained amendment**
(`ramza-freeze --amend --reason`, RAMZA "Drift watch"), which is what landed cycle 2's
fixes. `refine_cycles` therefore stays at **1**, and the DENY is recorded in the state
file rather than bypassed.

## Freeze Record

- **Criteria file:** `docs/BUILD-SPEC-v0.2.criteria.md` (byte-extracted from the Acceptance Criteria section above)
- **criteria_sha256:** recorded in `docs/BUILD-SPEC-v0.2.state.json` (`criteria_sha256`) — **twice amended**, hash-chained: `2453220…` (freeze) → `bb1b81b…` (critic fixes, batch 1) → current (critic fixes, batch 2). Every link carries its reason in `amendments[]`.
- **Gates run:** rightsize full/5 · complexity 7 (extended) · explore ×8 (winner H-A 83.0; H-F jsonx 79.0 over H-G 71.5; H-H omit-reason 60.5 rejected) · **critic recorded (ramza-critic-opus48 ≠ ramza)** · refine cycle 1/3 · ramza-lint pass (full) · ramza-ears-lint pass (47 criteria) · confidence 91.75 AUTO_PROCEED · drift scope declared (9 globs) · freeze + 2 amendments
- **Open:** none from the critique — all 14 findings are dispositioned above.
- **State:** `docs/BUILD-SPEC-v0.2.state.json` (schema `ramza/plan-state.v1`)
- Amendments only via `ramza-freeze --amend --reason` — hash-chained, never silent.
