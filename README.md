# atomos

**The compose/verify executor MCP for ECM** (the [Eidolons Context
Management](https://github.com/Rynaro/eidolons-ecm) context lifecycle) — the
tonberry-analog for ECM the way [tonberry](https://github.com/Rynaro/tonberry)
is the executor-analog for ESL. A thin, single-binary Go stdio MCP server
exposing a **closed** three-tool surface: `compose_handoff`, `verify_envelope`,
`verify_pins`.

`ATOMOS_VERSION` = `0.1.0` · 5th sibling executor (EIIS install / ECL wire /
ESL lifecycle / ECM context / **atomos MCP**) · opt-in, additive.

---

## What atomos is

atomos is an **alternate surface** to the always-canonical
`eidolons context …` bash kernel (`cli/src/context_handoff.sh`,
`cli/src/verify_envelope.sh` in `Rynaro/eidolons`): same inputs, same bytes.
The kernel is the source of truth; atomos conforms to it, never the reverse.

Its single load-bearing claim: **brief-SHA byte-identical parity** with
`eidolons context handoff` for the same inputs, proven by golden fixtures
captured from the kernel itself (`fixtures/parity/`) and guarded against
drift in CI (`.github/workflows/conformance.yml`, `scripts/regen-goldens.sh`).

atomos does **compose/verify ONLY** — see [Fence](#fence--capability-starvation)
below for what it structurally cannot do.

---

## The 3 tools (`mcp__atomos__*`)

| Tool | Purpose |
|------|---------|
| `compose_handoff` | Compose a session-handoff brief + ECL `INFORM` envelope (`ecm/handoff-brief@0.1`), byte-identical to `eidolons context handoff` for the same inputs. Writes the brief+envelope pair to `out_dir` (default `.eidolons/.context`) when `write_sidecar` is true (the default). |
| `verify_envelope` | Recompute a payload's SHA-256 and compare it to an ECL envelope's integrity tag, reproducing the kernel's full verdict matrix: `pass` · `tamper` · `inconsistent` · `unverifiable` · `missing_payload` · `unsupported_algo` · `malformed`. Advisory only — atomos never process-exits; `blocked` is a reported flag, never enforced here. |
| `verify_pins` | Probe a post-operation artifact for pin-marker survival (ECM spec §3.2). Advisory only: reports which pins are present/missing; never re-injects, never writes. |

The set is **closed** — `internal/tools/registry.go` is the single
declaration site. A fourth tool is a spec revision, never a drive-by
addition (`internal/tools/registry_test.go: TestToolSurfaceIsExactlyFenced`).

Non-goals for this MVP (v0.1.0): `compose_externalize_manifest` (deferred to
v0.2), nexus rostering (`roster/mcps.yaml`, a separate ESL change in the
nexus repo), and anything metering/policy/trigger/inject/persist.

---

## Parity contract

- **T1 (mandatory, byte-exact):** `compose_handoff`'s `brief_md` and
  `brief_sha256` equal the kernel's `handoff-<ts>.md` bytes and SHA-256
  exactly. The brief body embeds no timestamp — it is a pure function of the
  semantic inputs — so this is the integrity anchor.
- **T2 (envelope byte parity, achieved):** the envelope is emitted by a
  hand-rolled **ordered** JSON writer (`internal/ecl/envelope.go`) that
  reproduces the kernel's `jq -n` output exactly — insertion order, 2-space
  indent, jq-compatible string escaping, trailing newline — NOT Go's
  `json.MarshalIndent` (which sorts map keys and HTML-escapes `<`/`>`/`&`).

See `fixtures/README.md` for the full parity-vector layout and the
documented (unused) semantic-equivalence fallback contract.

```
go test ./internal/compose -run TestParityBriefBytes
go test ./internal/compose -run TestParityBriefSHA
go test ./internal/ecl     -run TestEnvelopeT2Parity
```

---

## Fence + capability starvation

atomos is compose/verify ONLY. It never reads a session's token budget or
zone, never evaluates a decision table, never fires an operation
(compaction, handoff-fresh), never writes a host prompt surface, and never
calls a durable-memory backend. This is enforced four ways:

1. **Single closed registry** — `internal/tools/registry.go` is the only
   tool-declaration site; `TestToolSurfaceIsExactlyFenced` asserts the live
   server's `tools/list` matches it exactly.
2. **Source deny-list test** — `TestFenceNoForbiddenSurface`
   (`internal/tools/registry_test.go`) greps all non-test, non-fixture Go
   source for forbidden identifiers and fails the build on any hit.
3. **Structural capability starvation** — the container runs with
   **`--cap-drop ALL`** and **`--security-opt no-new-privileges`**, no
   network client linked (atomos has no HTTP client dependency at all), and
   no durable-storage mount. Compose/verify are pure functions; there is
   simply no wire to metering, policy, persistence, or the host prompt
   surface even if code tried.
4. **Pure handlers** — every tool handler is `(input) → (artifact, error)`
   with no ambient state and no clock except an injected `ts`.
   `TestNoTimeNowInHandlerPackages` asserts `internal/compose`,
   `internal/verify`, `internal/ecl`, and `internal/hashing` never call
   `time.Now` — wall-clock defaulting happens at the single server-layer
   seam (`internal/server/server.go`) only when a caller omits `ts`/`iso_ts`.

Recommended run shape (mirrors the tonberry MCP template):

```
docker run --rm -i \
  --name atomos-<slug> \
  -v <project-root>:/workspace -w /workspace \
  --cap-drop ALL --security-opt no-new-privileges \
  ghcr.io/rynaro/atomos@<digest> serve
```

---

## Build / test

```
go build ./...
gofmt -l .              # must be empty
go vet ./...
go test ./... -v
```

Regenerate the parity goldens against a live `Rynaro/eidolons` checkout
(read-only oracle — this script never edits it):

```
EIDOLONS_NEXUS=/path/to/eidolons bash scripts/regen-goldens.sh
git diff --exit-code fixtures/parity   # clean when the kernel hasn't drifted
```

`scripts/regen-goldens.sh` is bash-3.2 compatible (no `declare -A`, no
`${var,,}`, no `readarray`/`mapfile`, no `&>>`) — the same discipline as the
nexus CLI, since macOS ships bash 3.2 as its system shell.

---

## Versioning

`ATOMOS_VERSION` names the image/MCP version (`atomos --version` /
`atomos version`). It is **never** stamped into a composed envelope —
`from.version` echoes the caller's `from_version` input verbatim (parity
trap 1; see `docs/BUILD-SPEC.md`).

---

## License

MIT — see [LICENSE](LICENSE).
