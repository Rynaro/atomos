# fixtures/

## fixtures/parity/ — the T1/T2 byte-parity contract

Each `fixtures/parity/<vector>/` directory is a matched golden set captured
by running the bash kernel (`eidolons context handoff --json`) as the
oracle via `scripts/regen-goldens.sh`:

```
fixtures/parity/<vector>/
  input.json      # compose_handoff input, incl. the CAPTURED ts/iso_ts/from_version
  brief.md        # the kernel's exact brief bytes
  envelope.json   # the kernel's exact envelope bytes (jq -n output)
  sha256          # sha256 of brief.md
```

Vectors (MVP, ADR §7 / BUILD-SPEC Track E, plus the v0.2 Track I edge-case
additions):

- `defaults-only` — every optional field absent; exercises every kernel
  default (task_state, thread_id chain, contains_tool_origin, from_version).
- `fully-populated` — every identifier/list field populated, single-line
  task_state, explicit thread_id.
- `narrative-open-vars` — narrative + open_vars + `contains_tool_origin: true`,
  thread_id resolved via session_id.
- `empty-sections` (v0.2) — anchors/symbols/decisions all absent, pinning
  that `## Identifiers` renders **empty with no `(none recorded)` fallback**
  — the one section with none.
- `oversize-brief` (v0.2) — a narrative long enough to push the brief past
  the 1500-token advisory target: `oversize: true`, `brief_md`
  **untruncated** and still byte-equal to the golden.
- `multiline-task-state` (v0.2) — a multi-line `task_state`: the envelope's
  `objective` uses only the **first non-blank line**, while the brief body
  keeps the full multiline text.
- `empty-list-entries` (v0.2) — empty-string list entries vanish, and a
  newline-bearing entry is embedded **raw** into a single brief bullet — the
  mirror image of the manifest's SPLIT behavior (M2 below): same kind of
  input, opposite handling in the two tools.

Every vector also carries an `advisory.json` — `{"tokens_est", "oversize"}`
captured from the kernel's own `--json` output (not computed by atomos) —
so `TestParityAdvisoryMatchesKernel` grounds the oversize claim in the
oracle's arithmetic and auto-guards the hardcoded 1500-token target: if the
nexus ever changes `limits.handoff_brief_target_tokens`, the kernel's
`oversize` flips and this golden drifts (CI catches it).

### T1 — mandatory, byte-exact

`compose_handoff`'s `brief_md` and `brief_sha256` MUST equal the vector's
`brief.md` / `sha256` byte-for-byte. The brief body embeds no timestamp
(`context_handoff.sh:107-168` is a pure function of the semantic inputs), so
this is fully reproducible and is the mandatory integrity anchor
(`internal/compose/parity_test.go: TestParityBriefBytes`,
`TestParityBriefSHA`).

### T2 — envelope byte parity (achieved, not just the fallback)

`internal/ecl.Envelope.Marshal()` is a hand-rolled ORDERED emitter (NOT
`json.MarshalIndent`, which sorts map keys and HTML-escapes `<`/`>`/`&`) that
reproduces the kernel's `jq -n` output exactly: 2-space indent, insertion
order (`context_handoff.sh:219-233`), jq-compatible string escaping (quote/
backslash/control-chars escaped as `\uXXXX`; forward slash and non-ASCII
UTF-8 passed through verbatim), and a single trailing newline. This was
verified empirically against a live `jq -n` capture (see git history / the
Vivi build session) before being hand-encoded. The escaper and its object/
array writers now live in `internal/jsonx` (extracted in v0.2 so the
manifest emitter below can share the exact same primitives instead of
growing a second, driftable copy) — `internal/ecl` calls into it rather
than owning a private copy.

`internal/ecl/parity_test.go: TestEnvelopeT2Parity` asserts
`compose.Handoff(...).EnvelopeBytes` — the SAME bytes the sidecar file
writer uses — equals each vector's `envelope.json` byte-for-byte. **The MVP
achieves full T2 byte parity; the ADR-sanctioned semantic-equivalence
fallback (field-equal + a `verify_envelope` pass) was not needed and is not
invoked**, but remains the documented contract (ADR §3.2, Risk R1) should a
future jq version or kernel change reintroduce a whitespace/escaping
divergence that isn't worth chasing byte-for-byte.

### Why `scripts/regen-goldens.sh` doesn't just re-diff blindly

The kernel has **no `--ts` flag** — every invocation wall-clocks a fresh
timestamp. `brief.md`/`sha256` are timestamp-free, so they're safe to
overwrite on every regen run and stay diff-clean whenever the kernel's
brief-building logic is unchanged. `envelope.json` (and `input.json`'s
`thread_id`-via-`ts` default chain) DO embed the timestamp, so the script
compares a freshly-captured envelope against the committed one with the
volatile fields (`message_id`, `artifact.path`, `trace.ts`, `thread_id`)
stripped; only a genuine field-level drift rewrites `envelope.json` and
back-fills `input.json` with the newly captured `ts`/`iso_ts`/`from_version`.
This is what makes `bash scripts/regen-goldens.sh && git diff --exit-code
fixtures/parity` (AC-E02) actually hold run over run against an unchanged
kernel commit, and what CI's drift-guard (AC-E03) checks against the
`PROVENANCE`-pinned nexus commit.

## fixtures/parity-manifest/ — the M0 single-document rule

`compose_externalize_manifest` (v0.2) mirrors ONLY the manifest-build
portion of `eidolons context externalize` (`context_externalize.sh:116-146`).
The kernel has no `--ts`/`--created-at` flag and prints no manifest on
stdout — its only manifest artifact is the **file-floor file**, written
when the durable-memory backend is absent (which a bare scratch dir always
triggers). That file-floor write IS the artifact atomos claims parity
with, driven through its `file_floor_path` side effect
(`eidolons context externalize --json`).

```
fixtures/parity-manifest/<vector>/
  input.json      # compose_externalize_manifest input, incl. the CAPTURED created_at + file_floor_reason
  manifest.json   # the kernel's exact file-floor bytes (11 keys — file_floor_reason always present, M7)
  sha256          # sha256 of manifest.json
  core.json       # manifest.json with file_floor_reason deleted, via `jq del(...)` — the kernel's OWN
                  #   emitter, so this reason-less 10-key form stays oracle-derived, never hand-authored
  core.sha256     # sha256 of core.json
```

Vectors (BUILD-SPEC-v0.2 Track I2):

- `manifest-defaults` — no inputs at all: default summary, five inline
  `[]`, `session_id: null`, `contains_tool_origin: false`.
- `manifest-populated` — every list populated, an explicit `session_id`,
  and a **multiline `summary`** (a scalar `--arg`, so it is NOT split — the
  discriminator against the array path below, and it exercises `\n`
  escaping in the emitter).
- `manifest-tool-origin` — `contains_tool_origin: true`, a **newline-bearing**
  entry (SPLITS into two array elements, M2) and an **empty-string** entry
  (VANISHES, M3).

### M0 — the single-document rule

Every `compose_externalize_manifest` call produces exactly **one** manifest
document: `manifest` (the response object) is its object form,
`manifest_sha256` is SHA-256 over its **canonical byte form**, and the
sidecar file (when written) contains those exact bytes. Nothing is hashed
that is not returned (`internal/compose: TestManifestResponseObjectDecodedFromHashedBytes`
decodes the response object FROM the hashed bytes rather than building it
by a second, independent path); nothing is written that is not hashed
(`TestManifestSidecarBytesHashToReportedSHA`). The digest is a pure
function of the tool inputs — it does not depend on whether the sidecar
was written (`TestManifestSHAIndependentOfSidecar`).

Unlike the brief, the manifest **embeds `created_at`** (M1,
`context_externalize.sh:137`) — there is no timestamp-free anchor for this
tool, so every vector freezes `created_at` and the parity tests
(`TestManifestParityBytes`/`TestManifestParitySHA`) assert byte/SHA
equality directly, no stripping needed.

### Two forms, two goldens — both byte-pinned, neither claim overstated

- **11-key form (`manifest.json`)**: the vector's `input.json` carries a
  caller-supplied `file_floor_reason` — the kernel's *sole* manifest writer
  (`_write_file_floor`) appends one unconditionally, so this is the only
  form that byte-matches a manifest the kernel ever wrote **to disk**
  (`TestManifestParityBytes`/`TestManifestParitySHA`).
- **10-key form (`core.json`)**: the same vector run with `file_floor_reason`
  omitted — the default path most callers get. It matches the kernel's
  **in-memory** `MANIFEST_JSON` object byte-for-byte (plus jq's trailing
  newline), derived from the 11-key golden by `jq` itself (the kernel's own
  emitter), never hand-authored (`TestManifestCoreParityBytes`/
  `TestManifestCoreParitySHA`). No kernel *file* has ever had this shape —
  the fixture doesn't claim otherwise.

## fixtures/conformant/ and fixtures/failing/ — the verify_envelope matrix

Each subdirectory is one `verify_envelope` scenario:

- an envelope JSON file (named `envelope.json`, or `<basename>.envelope.json`
  in `conformant/sibling-fallback/` to exercise the sibling-payload
  convention),
- a payload file (or none, for `missing-payload`),
- `expected.json`: `{"verdict", "mode", "blocked", ...}` — the verdict the
  kernel's `verify_envelope.sh` would produce for the same envelope+payload.

`internal/verify/envelope_test.go: TestVerdictMatrix` walks both trees and
asserts atomos's `verify_envelope` reproduces the expected verdict for
every fixture, covering the full matrix: `pass`, `tamper`, `inconsistent`,
`unverifiable`, `missing_payload`, `unsupported_algo`, `malformed`.
