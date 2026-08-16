# atomos

**The compose/verify executor MCP for ECM** (the [Eidolons Context
Management](https://github.com/Rynaro/eidolons-ecm) context lifecycle) — the
tonberry-analog for ECM the way [tonberry](https://github.com/Rynaro/tonberry)
is the executor-analog for ESL. A thin, single-binary Go stdio MCP server
exposing a **closed, now complete** four-tool surface: `compose_handoff`,
`verify_envelope`, `verify_pins`, `compose_externalize_manifest`.

`ATOMOS_VERSION` = `0.2.0` · 5th sibling executor (EIIS install / ECL wire /
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

## The 4 tools (`mcp__atomos__*`)

| Tool | Purpose |
|------|---------|
| `compose_handoff` | Compose a session-handoff brief + ECL `INFORM` envelope (`ecm/handoff-brief@0.1`), byte-identical to `eidolons context handoff` for the same inputs. Writes the brief+envelope pair to `out_dir` (default `.eidolons/.context`) when `write_sidecar` is true (the default). |
| `verify_envelope` | Recompute a payload's SHA-256 and compare it to an ECL envelope's integrity tag, reproducing the kernel's full verdict matrix: `pass` · `tamper` · `inconsistent` · `unverifiable` · `missing_payload` · `unsupported_algo` · `malformed`. Advisory only — atomos never process-exits; `blocked` is a reported flag, never enforced here. |
| `verify_pins` | Probe a post-operation artifact for pin-marker survival (ECM spec §3.2). Advisory only: reports which pins are present/missing; never re-injects, never writes. |
| `compose_externalize_manifest` | Build the identifier manifest that `eidolons context externalize` builds (anchors, symbols, decisions, failed approaches, open variables, `contains_tool_origin`, session, `created_at`) plus its SHA-256, and write the **file-floor** sidecar under `out_dir` (default `.eidolons/.context`) when `write_sidecar` is true (the default). Stops there: durable memory beyond that one file is out of reach from this surface — a caller wanting it uses the kernel verb directly. |

The set is **closed at four** — `internal/tools/registry.go` is the single
declaration site. A fifth tool is a new ADR, never a drive-by addition
(`internal/tools/registry_test.go: TestToolSurfaceIsExactlyFenced` asserts
exact equality, never a superset check).

Non-goals: nexus rostering (`roster/mcps.yaml`, a separate ESL change in the
nexus repo — Phase 3), and anything metering/policy/trigger/inject/persist.

---

## Parity contract

- **T1 (mandatory, byte-exact):** `compose_handoff`'s `brief_md` and
  `brief_sha256` equal the kernel's `handoff-<ts>.md` bytes and SHA-256
  exactly. The brief body embeds no timestamp — it is a pure function of the
  semantic inputs — so this is the integrity anchor.
- **T2 (envelope byte parity, achieved):** the envelope is emitted by a
  hand-rolled **ordered** JSON writer (`internal/jsonx`, shared with the
  manifest emitter) that reproduces the kernel's `jq -n` output exactly —
  insertion order, 2-space indent, jq-compatible string escaping, trailing
  newline — NOT Go's `json.MarshalIndent` (which sorts map keys and
  HTML-escapes `<`/`>`/`&`).
- **M0 (manifest single-document rule):** `compose_externalize_manifest`
  produces exactly one manifest document per call — `manifest_sha256` is
  SHA-256 over the canonical bytes returned, and the sidecar file (when
  written) carries those exact bytes. Unlike the brief, the manifest embeds
  `created_at` (M1) — so its byte-parity fixtures are frozen-`created_at`
  vectors, not timestamp-free ones.

See `fixtures/README.md` for the full parity-vector layout and the
documented (unused) semantic-equivalence fallback contract.

```
go test ./internal/compose -run TestParityBriefBytes
go test ./internal/compose -run TestParityBriefSHA
go test ./internal/ecl     -run TestEnvelopeT2Parity
go test ./internal/compose -run TestManifestParityBytes
go test ./internal/compose -run TestManifestParitySHA
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
   with no ambient state and no clock except an injected `ts`/`created_at`.
   `TestNoTimeNowInHandlerPackages` asserts `internal/compose`,
   `internal/verify`, `internal/ecl`, `internal/hashing`, and `internal/jsonx`
   never call `time.Now` — wall-clock defaulting happens at the single
   server-layer seam (`internal/server/server.go`) only when a caller omits
   `ts`/`iso_ts`/`created_at`.

Recommended run shape — mirrors `cli/templates/mcp/atomos.mcp.json.tmpl` in
[`Rynaro/eidolons`](https://github.com/Rynaro/eidolons), which is the
authoritative source. `eidolons mcp install atomos` renders exactly this:

```
docker run --rm -i \
  --user "$(id -u):$(id -g)" \
  --label eidolons.project=<slug> \
  -v <project-root>:<project-root>:z -w <project-root> \
  --cap-drop ALL --security-opt no-new-privileges \
  ghcr.io/rynaro/atomos@<digest> serve
```

Three details are load-bearing, and earlier versions of this README got each
of them wrong — see [Known issues](#known-issues) for the symptoms:

- **`--user "$(id -u):$(id -g)"`** — the image is distroless-nonroot (UID
  65532). Without this, every write to a workspace owned by your host user
  fails.
- **Identity mount `<project-root>:<project-root>`, not `:/workspace`** —
  tool arguments that carry a path (`out_dir`) are host-absolute, and the
  server resolves them inside its own mount namespace. The container must
  expose the project at the same absolute path the caller uses.
- **No `--name`** — a static name makes a reconnect after a dropped stdio
  pipe collide with the still-running `--rm` container. Per-project identity
  rides the `--label` instead.

---

## Known issues

| Symptom | Cause | Fixed in | Repair |
|---|---|---|---|
| `open …: no such file or directory` (ENOENT) on any tool taking a path argument ([#4](https://github.com/Rynaro/atomos/issues/4)) | Wiring mounted the project at `/workspace`, so a host-absolute `out_dir` did not exist in the container's namespace | eidolons **v2.12.0** (identity mount) | `eidolons mcp install atomos@<ver> --force --no-pull` |
| `mkdir: Permission denied` / EACCES on every write ([#1](https://github.com/Rynaro/atomos/issues/1)) | Container ran as UID 65532 against a workspace owned by the host user | eidolons **v2.14.0** (`--user` pin) | `eidolons mcp install atomos@<ver> --force --no-pull` |
| MCP `-32000` server error after a dropped connection | Static `docker --name` collided with the still-running container on reconnect | eidolons **v2.4.0** (`--name` removed) | `eidolons mcp install atomos@<ver> --force --no-pull` |

All three are **wiring** defects fixed in the nexus templates, not in atomos
itself — but a project installed before the fix keeps its old `.mcp.json`
until it is re-rendered, because no upgrade path re-renders an install whose
version has not changed. `eidolons mcp verify` reports this as a
`V-ARGV-DRIFT` finding and prints the repair command.

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
