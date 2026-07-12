// Package tools is the SINGLE source of truth for atomos's closed MCP tool
// surface (ADR §2, §4.1). internal/server enumerates and dispatches from this
// registry; nothing else in the codebase may declare a tool through another
// path. The set is now CLOSED AT FOUR: compose_handoff, verify_envelope,
// verify_pins, compose_externalize_manifest (added in the authorized v0.2
// revision, ADR §2's table and §4.1's pre-written assertion). Extending it
// further is a spec revision, never a drive-by addition — a fifth tool
// needs a new ADR, not a new BUILD-SPEC.
package tools

// Descriptor names one registered tool and its human-readable description.
type Descriptor struct {
	Name        string
	Description string
}

// Registry is the ordered, closed tool set. internal/server builds its
// tools/list and tools/call dispatch from exactly this slice.
var Registry = []Descriptor{
	{
		Name: "compose_handoff",
		Description: "Compose a session-handoff brief + ECL INFORM envelope " +
			"(ecm/handoff-brief@0.1), byte-identical to `eidolons context handoff` " +
			"for the same inputs. Writes the brief+envelope pair to out_dir " +
			"(default .eidolons/.context) when write_sidecar is true (the default).",
	},
	{
		Name: "verify_envelope",
		Description: "Recompute a payload's SHA-256 and compare it to an ECL " +
			"envelope's integrity tag, reproducing the kernel's full verdict " +
			"matrix (pass/tamper/inconsistent/unverifiable/missing_payload/" +
			"unsupported_algo/malformed). Advisory only — atomos never process-" +
			"exits; blocked is a reported flag, never enforced here.",
	},
	{
		Name: "verify_pins",
		Description: "Probe a post-operation artifact for pin-marker survival " +
			"(ECM spec §3.2). Advisory only: reports which pins are present/" +
			"missing; never re-injects, never writes.",
	},
	{
		Name: "compose_externalize_manifest",
		Description: "Build the identifier manifest that `eidolons context " +
			"externalize` builds — anchors, symbols, decisions, failed " +
			"approaches, open variables, contains_tool_origin, session, and " +
			"created_at — plus its SHA-256, and write the file-floor sidecar " +
			"under out_dir (default .eidolons/.context) when write_sidecar is " +
			"true (the default). Stops there: durable memory beyond that one " +
			"file is out of reach from this surface — a caller wanting it " +
			"uses the kernel verb directly.",
	},
}

// Names returns the ordered tool names in Registry.
func Names() []string {
	names := make([]string, len(Registry))
	for i, d := range Registry {
		names[i] = d.Name
	}
	return names
}

// Description looks up a tool's description by name ("" if not found).
func Description(name string) string {
	for _, d := range Registry {
		if d.Name == name {
			return d.Description
		}
	}
	return ""
}
