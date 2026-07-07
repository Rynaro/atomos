// Package tools is the SINGLE source of truth for atomos's closed MCP tool
// surface (ADR §2, §4.1). internal/server enumerates and dispatches from this
// registry; nothing else in the codebase may declare a tool through another
// path. The set is closed: compose_handoff, verify_envelope, verify_pins.
// Extending it is a spec revision, never a drive-by addition — if a fourth
// tool ever seems necessary, STOP (BUILD-SPEC.md "Executor Notes").
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
