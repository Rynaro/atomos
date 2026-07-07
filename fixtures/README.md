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

Vectors (MVP, ADR §7 / BUILD-SPEC Track E):

- `defaults-only` — every optional field absent; exercises every kernel
  default (task_state, thread_id chain, contains_tool_origin, from_version).
- `fully-populated` — every identifier/list field populated, single-line
  task_state, explicit thread_id.
- `narrative-open-vars` — narrative + open_vars + `contains_tool_origin: true`,
  thread_id resolved via session_id.

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
Vivi build session) before being hand-encoded into `internal/ecl/envelope.go`.

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
