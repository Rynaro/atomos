# Session Handoff Brief

## Identifiers
- anchor: internal/compose/handoff.go:120
- anchor: internal/ecl/envelope.go:1
- symbol: BuildBrief
- symbol: ecl.Envelope
- decision: Hand-roll the ordered JSON emitter instead of json.MarshalIndent

## Failed approaches
- Tried map[string]interface{} + json.MarshalIndent (alphabetized keys)
- Tried struct field reordering without escaping parity check

## Next steps
- Write parity_test.go asserting brief + envelope byte equality
- Wire scripts/regen-goldens.sh

## Narrative

Task state: Building atomos Track B compose_handoff parity suite

## contains_tool_origin
false
