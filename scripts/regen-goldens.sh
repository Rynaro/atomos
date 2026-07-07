#!/usr/bin/env bash
# scripts/regen-goldens.sh — regenerate fixtures/parity/<vector>/{brief.md,
# envelope.json,sha256} by running the bash kernel (`eidolons context
# handoff`) as the oracle, exactly as the ADR/BUILD-SPEC mandate.
#
# The kernel has NO --ts flag (it wall-clocks context_now_epoch_ts): every
# invocation mints a genuinely fresh timestamp. So this script's diff-clean
# contract (AC-E02) is:
#   - brief.md + sha256 are TIMESTAMP-FREE (context_handoff.sh:107-168, a
#     pure function of the semantic inputs) — they are rewritten every run
#     and MUST stay byte-identical to the committed goldens when the
#     kernel's brief-building logic hasn't changed.
#   - envelope.json embeds the timestamp (message_id, artifact.path,
#     trace.ts). This script compares a freshly-captured envelope against
#     the committed one with ONLY those three volatile fields stripped; if
#     everything else is identical, the committed envelope.json and
#     input.json's ts/iso_ts/from_version are left untouched (so `git diff`
#     stays clean run over run). Only a genuine field-level drift rewrites
#     envelope.json + backfills input.json with the newly captured ts.
#
# Requires: EIDOLONS_NEXUS pointing at a Rynaro/eidolons checkout (the
# READ-ONLY oracle — this script never edits it), jq, git, bash, mktemp.
#
# Bash 3.2 safe (same discipline as the nexus CLI, whose CI runs macOS's
# system bash): no bash-4-only associative arrays, no lower/upper-case
# parameter expansion, no line-array-slurp builtins, no combined append
# stdout+stderr redirection operator. Only plain indexed arrays, `[ ]`
# tests, command substitution, and process substitution are used below.

set -euo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SELF_DIR/.." && pwd)"
FIXTURES_DIR="$REPO_ROOT/fixtures/parity"

: "${EIDOLONS_NEXUS:?Set EIDOLONS_NEXUS to a Rynaro/eidolons checkout (the read-only oracle).}"
KERNEL_ENTRY="$EIDOLONS_NEXUS/cli/eidolons"
[ -f "$KERNEL_ENTRY" ] || { echo "regen-goldens: kernel entrypoint not found: $KERNEL_ENTRY" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "regen-goldens: jq is required" >&2; exit 1; }
command -v git >/dev/null 2>&1 || { echo "regen-goldens: git is required" >&2; exit 1; }

VECTORS="defaults-only fully-populated narrative-open-vars"
ANY_DRIFT=false

for vector in $VECTORS; do
  vdir="$FIXTURES_DIR/$vector"
  input="$vdir/input.json"
  [ -f "$input" ] || { echo "regen-goldens: missing $input" >&2; exit 1; }

  # ── Build the CLI flags from input.json's SEMANTIC fields only. ts/
  # iso_ts/from_version are never fed in — they are captured FROM the run.
  FLAGS=()

  task_state="$(jq -r '.task_state // empty' "$input")"
  [ -n "$task_state" ] && FLAGS+=(--task-state "$task_state")

  narrative="$(jq -r '.narrative // empty' "$input")"
  [ -n "$narrative" ] && FLAGS+=(--narrative "$narrative")

  while IFS= read -r v; do [ -n "$v" ] && FLAGS+=(--anchor "$v"); done < <(jq -r '.anchors[]? // empty' "$input")
  while IFS= read -r v; do [ -n "$v" ] && FLAGS+=(--symbol "$v"); done < <(jq -r '.symbols[]? // empty' "$input")
  while IFS= read -r v; do [ -n "$v" ] && FLAGS+=(--decision "$v"); done < <(jq -r '.decisions[]? // empty' "$input")
  while IFS= read -r v; do [ -n "$v" ] && FLAGS+=(--failed-approach "$v"); done < <(jq -r '.failed_approaches[]? // empty' "$input")
  while IFS= read -r v; do [ -n "$v" ] && FLAGS+=(--open-var "$v"); done < <(jq -r '.open_vars[]? // empty' "$input")
  while IFS= read -r v; do [ -n "$v" ] && FLAGS+=(--next-step "$v"); done < <(jq -r '.next_steps[]? // empty' "$input")

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
  ( cd "$scratch" && bash "$KERNEL_ENTRY" context handoff "${FLAGS[@]}" --json ) > "$scratch/out.json"

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
  sha256_file() {
    if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{print $1}';
    else sha256sum "$1" | awk '{print $1}'; fi
  }
  sha256_file "$new_brief" > "$vdir/sha256.tmp"
  mv "$vdir/sha256.tmp" "$vdir/sha256"

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

# ── Provenance stamp: ECM_VERSION (roster/context-policy.yaml) + the nexus
# commit the goldens were captured from. Deterministic (no wall-clock field)
# so it never contributes to spurious diffs across repeat regen runs.
ecm_version="$(grep '^ecm_version:' "$EIDOLONS_NEXUS/roster/context-policy.yaml" 2>/dev/null | sed -E 's/^ecm_version: *"?([^"]*)"?.*/\1/')"
[ -n "$ecm_version" ] || ecm_version="unknown"
nexus_commit="$(git -C "$EIDOLONS_NEXUS" rev-parse HEAD 2>/dev/null || echo unknown)"

cat > "$FIXTURES_DIR/PROVENANCE" <<EOF
ECM_VERSION=$ecm_version
NEXUS_COMMIT=$nexus_commit
CAPTURED_VIA=eidolons context handoff --json (Rynaro/eidolons checkout, cli/src/context_handoff.sh)
VECTORS=$VECTORS
NOTE=ts/iso_ts/from_version are CAPTURED from the kernel run (it wall-clocks internally and stamps its own EIDOLONS_VERSION), not chosen; scripts/regen-goldens.sh back-fills them into each vector's input.json. Because every regen run mints a genuinely fresh wall-clock timestamp, the script treats brief.md/sha256 (timestamp-free) as the byte-diff-clean contract and only rewrites envelope.json/input.json's ts fields when a SEMANTIC (non-timestamp) field actually drifted.
EOF

if [ "$ANY_DRIFT" = "true" ]; then
  echo "regen-goldens: drift detected and goldens updated — review before committing." >&2
else
  echo "regen-goldens: no drift — all vectors already match the kernel oracle." >&2
fi

exit 0
