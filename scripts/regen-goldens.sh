#!/usr/bin/env bash
# scripts/regen-goldens.sh — regenerate the parity goldens by running the
# bash kernel as the oracle (ADR/BUILD-SPEC mandate). Two arms:
#
#   - HANDOFF arm (fixtures/parity/<vector>/): `eidolons context handoff`
#     (cli/src/context_handoff.sh). Produces brief.md, envelope.json,
#     sha256, advisory.json.
#   - MANIFEST arm (fixtures/parity-manifest/<vector>/): `eidolons context
#     externalize` (cli/src/context_externalize.sh). The kernel has no
#     --ts/--created-at flag and prints no manifest on stdout — its only
#     manifest artifact is the FILE-FLOOR file, written when the memory
#     backend is absent (which a bare scratch dir always triggers). That
#     file-floor write IS the artifact atomos claims parity with, driven via
#     its file_floor_path side effect (AC-I07). Produces manifest.json,
#     sha256, core.json (= manifest.json with file_floor_reason deleted —
#     via jq, the kernel's own emitter, so it stays oracle-derived rather
#     than hand-authored), core.sha256.
#
# Diff-clean contract (AC-I06, mirrors v0.1's envelope-arm trick):
#   - HANDOFF: brief.md/sha256/advisory.json are TIMESTAMP-FREE (pure
#     functions of the semantic inputs) — rewritten every run, byte-stable
#     whenever the kernel's brief-building logic is unchanged. envelope.json
#     embeds the timestamp (message_id/artifact.path/trace.ts/thread_id via
#     the "handoff-<ts>" default chain); compared with those fields
#     stripped, and only rewritten (+ input.json ts/iso_ts/from_version
#     back-filled) on a genuine field-level drift.
#   - MANIFEST: created_at is INSIDE the hashed document (M1) — there is no
#     timestamp-free anchor for this tool. Compared with .created_at
#     stripped; only rewritten (+ input.json created_at/file_floor_reason
#     back-filled) on genuine semantic drift.
#
# Requires: EIDOLONS_NEXUS pointing at a Rynaro/eidolons checkout (the
# READ-ONLY oracle — this script never edits it), jq, git, bash, mktemp.
#
# Bash 3.2 safe (same discipline as the nexus CLI, whose CI runs macOS's
# system bash): no bash-4-only associative arrays, no lower/upper-case
# parameter expansion, no line-array-slurp builtins, no combined append
# stdout+stderr redirection operator. Only plain indexed arrays, `[ ]`
# tests, command substitution, and process substitution are used below.
#
# The kernel flag array is ALWAYS seeded with --json before any conditional
# flag is appended (both arms) so "${FLAGS[@]}" is never expanded empty —
# under `set -u`, bash 3.2 (unlike bash 4+) aborts with "unbound variable"
# on an empty array expansion, and the defaults-only/manifest-defaults
# vectors (every semantic field absent) would otherwise reproduce exactly
# that abort on a maintainer's macOS regen run, silently, since CI's
# drift-guard historically ran on ubuntu-latest (bash 5) only.

set -euo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SELF_DIR/.." && pwd)"
HANDOFF_DIR="$REPO_ROOT/fixtures/parity"
MANIFEST_DIR="$REPO_ROOT/fixtures/parity-manifest"

: "${EIDOLONS_NEXUS:?Set EIDOLONS_NEXUS to a Rynaro/eidolons checkout (the read-only oracle).}"
KERNEL_ENTRY="$EIDOLONS_NEXUS/cli/eidolons"
[ -f "$KERNEL_ENTRY" ] || { echo "regen-goldens: kernel entrypoint not found: $KERNEL_ENTRY" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "regen-goldens: jq is required" >&2; exit 1; }
command -v git >/dev/null 2>&1 || { echo "regen-goldens: git is required" >&2; exit 1; }

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{print $1}';
  else sha256sum "$1" | awk '{print $1}'; fi
}

HANDOFF_VECTORS="defaults-only fully-populated narrative-open-vars empty-sections oversize-brief multiline-task-state empty-list-entries"
MANIFEST_VECTORS="manifest-defaults manifest-populated manifest-tool-origin"
ANY_DRIFT=false

# refuse_trailing_newline_entries_handoff / _manifest INPUT_JSON — AC-I14: a
# list element ending in a newline would be silently stripped by $( )
# command substitution when captured back into a golden, yielding a WRONG
# golden with no signal. Refuse to process such a fixture at all
# (mechanical, not prose — AC-I08 exists precisely because prose-only
# guards get bypassed).
refuse_trailing_newline_entries_handoff() {
  jq -e '[.anchors[]?,.symbols[]?,.decisions[]?,.failed_approaches[]?,.open_vars[]?,.next_steps[]?]
         | map(select(endswith("\n"))) | length == 0' "$1" >/dev/null || {
    echo "regen-goldens: [$1] fixture list element ends with a newline — cannot be captured faithfully" >&2
    exit 1
  }
}

refuse_trailing_newline_entries_manifest() {
  jq -e '[.anchors[]?,.symbols[]?,.decisions[]?,.failed_approaches[]?,.open_vars[]?]
         | map(select(endswith("\n"))) | length == 0' "$1" >/dev/null || {
    echo "regen-goldens: [$1] fixture list element ends with a newline — cannot be captured faithfully" >&2
    exit 1
  }
}

# ═══════════════════════════════════════════════════════════════════════
# HANDOFF arm — eidolons context handoff (cli/src/context_handoff.sh)
# ═══════════════════════════════════════════════════════════════════════
for vector in $HANDOFF_VECTORS; do
  vdir="$HANDOFF_DIR/$vector"
  input="$vdir/input.json"
  [ -f "$input" ] || { echo "regen-goldens: missing $input" >&2; exit 1; }

  refuse_trailing_newline_entries_handoff "$input"

  # ── Build the CLI flags from input.json's SEMANTIC fields only. ts/
  # iso_ts/from_version are never fed in — they are captured FROM the run.
  # Seeded (AC-I13): never expanded empty under set -u on bash 3.2.
  FLAGS=(--json)

  task_state="$(jq -r '.task_state // empty' "$input")"
  [ -n "$task_state" ] && FLAGS+=(--task-state "$task_state")

  narrative="$(jq -r '.narrative // empty' "$input")"
  [ -n "$narrative" ] && FLAGS+=(--narrative "$narrative")

  # Index-driven extraction (AC-I08): NO non-empty guard, NO read-loop
  # splitting. An empty-string element reaches the kernel as an empty
  # argument (it VANISHES there, on the kernel's own terms); a
  # newline-bearing element reaches the kernel as ONE argument (the brief
  # embeds the newline RAW — a `read`-split loop would instead hand the
  # kernel TWO flags and capture a golden showing the brief SPLIT, the
  # exact inverse of what these vectors exist to pin).
  n="$(jq -r '.anchors | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.anchors[$i]' "$input")"; FLAGS+=(--anchor "$v"); i=$((i + 1)); done
  n="$(jq -r '.symbols | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.symbols[$i]' "$input")"; FLAGS+=(--symbol "$v"); i=$((i + 1)); done
  n="$(jq -r '.decisions | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.decisions[$i]' "$input")"; FLAGS+=(--decision "$v"); i=$((i + 1)); done
  n="$(jq -r '.failed_approaches | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.failed_approaches[$i]' "$input")"; FLAGS+=(--failed-approach "$v"); i=$((i + 1)); done
  n="$(jq -r '.open_vars | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.open_vars[$i]' "$input")"; FLAGS+=(--open-var "$v"); i=$((i + 1)); done
  n="$(jq -r '.next_steps | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.next_steps[$i]' "$input")"; FLAGS+=(--next-step "$v"); i=$((i + 1)); done

  thread_id="$(jq -r '.thread_id // empty' "$input")"
  [ -n "$thread_id" ] && FLAGS+=(--thread-id "$thread_id")

  session_id="$(jq -r '.session_id // empty' "$input")"
  [ -n "$session_id" ] && FLAGS+=(--session-id "$session_id")

  contains_tool_origin="$(jq -r '.contains_tool_origin // false' "$input")"
  [ "$contains_tool_origin" = "true" ] && FLAGS+=(--contains-tool-origin)

  # ── Run the oracle in a scratch dir (no crystalium gating: a bare scratch
  # dir has no docker/config wiring, so the kernel's memory_probe_gated_in
  # check fails closed and the ingest step is skipped — fail-open, no
  # side effects beyond the scratch dir itself).
  scratch="$(mktemp -d)"
  ( cd "$scratch" && bash "$KERNEL_ENTRY" context handoff "${FLAGS[@]}" ) > "$scratch/out.json"

  new_brief="$(jq -r '.brief_path' "$scratch/out.json")"
  new_envelope="$(jq -r '.envelope_path' "$scratch/out.json")"

  # ── T1 (brief.md + sha256): timestamp-free, always safe to rewrite in
  # place — byte-identical to the previous golden whenever the kernel's
  # brief-building logic is unchanged.
  if [ -f "$vdir/brief.md" ] && cmp -s "$new_brief" "$vdir/brief.md"; then
    : # unchanged — leave sha256 alone too
  else
    ANY_DRIFT=true
    echo "regen-goldens: [$vector] brief.md drifted — updating golden" >&2
  fi
  cp "$new_brief" "$vdir/brief.md"
  sha256_file "$new_brief" > "$vdir/sha256.tmp"
  mv "$vdir/sha256.tmp" "$vdir/sha256"

  # ── advisory.json: {tokens_est, oversize} — timestamp-free (pure
  # function of the brief's byte length), so always safe to rewrite in
  # place. Grounds the oversize assertion in the ORACLE's own arithmetic
  # rather than atomos's, and auto-guards the hardcoded 1500-token target:
  # if the nexus ever changes limits.handoff_brief_target_tokens, the
  # kernel's oversize flips here and the golden drifts (CI catches it).
  jq '{tokens_est, oversize}' "$scratch/out.json" > "$vdir/advisory.json.tmp"
  if [ -f "$vdir/advisory.json" ] && diff -q "$vdir/advisory.json.tmp" "$vdir/advisory.json" >/dev/null 2>&1; then
    rm -f "$vdir/advisory.json.tmp"
  else
    ANY_DRIFT=true
    echo "regen-goldens: [$vector] advisory.json drifted — updating golden" >&2
    mv "$vdir/advisory.json.tmp" "$vdir/advisory.json"
  fi

  # ── T2 (envelope.json): compare with the timestamp-derived fields
  # stripped (message_id/artifact.path/trace.ts always embed ts; thread_id
  # ALSO embeds ts via the "handoff-<ts>" default chain whenever a vector
  # gives neither --thread-id nor --session-id, e.g. defaults-only).
  # Identical => leave the committed envelope.json + input.json's
  # ts/iso_ts/from_version untouched (diff-clean). Different => a genuine
  # field-level drift; rewrite the envelope and back-fill the newly
  # captured ts/iso_ts/from_version into input.json.
  strip_volatile() { jq 'del(.message_id, .artifact.path, .trace.ts, .thread_id)' "$1"; }

  envelope_drifted=true
  if [ -f "$vdir/envelope.json" ]; then
    if diff -q <(strip_volatile "$new_envelope") <(strip_volatile "$vdir/envelope.json") >/dev/null 2>&1; then
      envelope_drifted=false
    fi
  fi

  if [ "$envelope_drifted" = "true" ]; then
    ANY_DRIFT=true
    echo "regen-goldens: [$vector] envelope.json drifted — updating golden + input.json" >&2
    cp "$new_envelope" "$vdir/envelope.json"

    base="$(basename "$new_brief" .md)"          # handoff-<ts>
    captured_ts="${base#handoff-}"
    captured_iso_ts="$(jq -r '.trace.ts' "$new_envelope")"
    captured_from_version="$(jq -r '.from.version' "$new_envelope")"

    jq --arg ts "$captured_ts" --arg iso_ts "$captured_iso_ts" --arg fromv "$captured_from_version" \
      '.ts = $ts | .iso_ts = $iso_ts | .from_version = $fromv' \
      "$input" > "$input.tmp"
    mv "$input.tmp" "$input"
  fi

  rm -rf "$scratch"
done

# ═══════════════════════════════════════════════════════════════════════
# MANIFEST arm — eidolons context externalize (cli/src/context_externalize.sh)
# ═══════════════════════════════════════════════════════════════════════
for vector in $MANIFEST_VECTORS; do
  vdir="$MANIFEST_DIR/$vector"
  input="$vdir/input.json"
  [ -f "$input" ] || { echo "regen-goldens: missing $input" >&2; exit 1; }

  refuse_trailing_newline_entries_manifest "$input"

  # created_at/ts/file_floor_reason are OUTPUT-only (captured FROM the run
  # below, backfilled after) — the kernel has no flag for any of them, so
  # they are never read here. Seeded (AC-I13): never expanded empty.
  FLAGS=(--json)

  summary="$(jq -r '.summary // empty' "$input")"
  [ -n "$summary" ] && FLAGS+=(--summary "$summary")

  n="$(jq -r '.anchors | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.anchors[$i]' "$input")"; FLAGS+=(--anchor "$v"); i=$((i + 1)); done
  n="$(jq -r '.symbols | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.symbols[$i]' "$input")"; FLAGS+=(--symbol "$v"); i=$((i + 1)); done
  n="$(jq -r '.decisions | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.decisions[$i]' "$input")"; FLAGS+=(--decision "$v"); i=$((i + 1)); done
  n="$(jq -r '.failed_approaches | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.failed_approaches[$i]' "$input")"; FLAGS+=(--failed-approach "$v"); i=$((i + 1)); done
  n="$(jq -r '.open_vars | length // 0' "$input")"; i=0
  while [ "$i" -lt "$n" ]; do v="$(jq -r --argjson i "$i" '.open_vars[$i]' "$input")"; FLAGS+=(--open-var "$v"); i=$((i + 1)); done

  session_id="$(jq -r '.session_id // empty' "$input")"
  [ -n "$session_id" ] && FLAGS+=(--session-id "$session_id")

  contains_tool_origin="$(jq -r '.contains_tool_origin // false' "$input")"
  [ "$contains_tool_origin" = "true" ] && FLAGS+=(--contains-tool-origin)

  # ── Run the oracle in a scratch dir. A bare scratch dir has no
  # .mcp.json/eidolons.mcp.lock, so memory_probe_gated_in fails closed and
  # the kernel takes its file-floor branch (warn + write + exit 0) — the
  # exact artifact atomos claims parity with (Q6), never a degraded mode.
  scratch="$(mktemp -d)"
  ( cd "$scratch" && bash "$KERNEL_ENTRY" context externalize "${FLAGS[@]}" ) > "$scratch/out.json" 2>/dev/null

  # AC-I07: locate the artifact through the kernel's own file_floor_path,
  # never by constructing the path ourselves.
  new_manifest="$(jq -r '.file_floor_path' "$scratch/out.json")"
  [ -n "$new_manifest" ] && [ "$new_manifest" != "null" ] || {
    echo "regen-goldens: [$vector] kernel did not report a file_floor_path — cannot capture" >&2
    exit 1
  }

  # M1: created_at is INSIDE the hashed document, so there is no
  # timestamp-free anchor here — compare with .created_at stripped.
  strip_created_at() { jq 'del(.created_at)' "$1"; }

  manifest_drifted=true
  if [ -f "$vdir/manifest.json" ]; then
    if diff -q <(strip_created_at "$new_manifest") <(strip_created_at "$vdir/manifest.json") >/dev/null 2>&1; then
      manifest_drifted=false
    fi
  fi

  if [ "$manifest_drifted" = "true" ]; then
    ANY_DRIFT=true
    echo "regen-goldens: [$vector] manifest.json drifted — updating golden + input.json" >&2
    cp "$new_manifest" "$vdir/manifest.json"
    sha256_file "$vdir/manifest.json" > "$vdir/sha256"
    jq 'del(.file_floor_reason)' "$vdir/manifest.json" > "$vdir/core.json"
    sha256_file "$vdir/core.json" > "$vdir/core.sha256"

    captured_created_at="$(jq -r '.created_at' "$vdir/manifest.json")"
    captured_reason="$(jq -r '.file_floor_reason // empty' "$vdir/manifest.json")"

    jq --arg created_at "$captured_created_at" --arg reason "$captured_reason" \
      '.created_at = $created_at | .file_floor_reason = $reason' \
      "$input" > "$input.tmp"
    mv "$input.tmp" "$input"
  fi

  rm -rf "$scratch"
done

# ── Provenance stamp: ECM_VERSION (roster/context-policy.yaml) + the nexus
# commit both trees were captured from, and both vector lists. Deterministic
# (no wall-clock field) so it never contributes to spurious diffs across
# repeat regen runs.
ecm_version="$(grep '^ecm_version:' "$EIDOLONS_NEXUS/roster/context-policy.yaml" 2>/dev/null | sed -E 's/^ecm_version: *"?([^"]*)"?.*/\1/')"
[ -n "$ecm_version" ] || ecm_version="unknown"
nexus_commit="$(git -C "$EIDOLONS_NEXUS" rev-parse HEAD 2>/dev/null || echo unknown)"

cat > "$HANDOFF_DIR/PROVENANCE" <<EOF
ECM_VERSION=$ecm_version
NEXUS_COMMIT=$nexus_commit
CAPTURED_VIA=eidolons context handoff --json + eidolons context externalize --json (Rynaro/eidolons checkout, cli/src/context_handoff.sh + cli/src/context_externalize.sh)
VECTORS=$HANDOFF_VECTORS
MANIFEST_VECTORS=$MANIFEST_VECTORS
NOTE=ts/iso_ts/from_version (handoff) and created_at/file_floor_reason (manifest) are CAPTURED from the kernel run (it wall-clocks internally and stamps its own EIDOLONS_VERSION), not chosen; scripts/regen-goldens.sh back-fills them into each vector's input.json. Because every regen run mints a genuinely fresh wall-clock timestamp, the script treats brief.md/sha256/advisory.json (handoff, timestamp-free) and byte comparisons with .created_at stripped (manifest, M1 — created_at is INSIDE the hashed document so there is no timestamp-free anchor for that tool) as the diff-clean contract, rewriting a golden only when a SEMANTIC (non-timestamp) field actually drifted.
EOF

if [ "$ANY_DRIFT" = "true" ]; then
  echo "regen-goldens: drift detected and goldens updated — review before committing." >&2
else
  echo "regen-goldens: no drift — all vectors already match the kernel oracle." >&2
fi

exit 0
