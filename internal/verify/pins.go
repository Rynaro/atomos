package verify

import (
	"os"
	"regexp"
	"strings"
)

// Pin is one entry from the caller-supplied pin set (ids come from the
// nexus's roster/pins.yaml — atomos does NOT bake in a default set; the
// roster stays the authority, per ADR §2.3 and Rejected Alternative #7).
type Pin struct {
	ID     string  `json:"id"`
	Marker *string `json:"marker,omitempty"` // nil -> literal id token; else a Go regexp
}

// PinsInput is the verify_pins MCP tool input (ECM spec §3.2 probe).
type PinsInput struct {
	Pins         []Pin   `json:"pins"`
	Artifact     *string `json:"artifact,omitempty"`
	ArtifactPath string  `json:"artifact_path,omitempty"`
}

// PinStatus reports one pin's survival.
type PinStatus struct {
	ID      string `json:"id"`
	Present bool   `json:"present"`
}

// PinsResult is the verify_pins MCP tool output. Purely advisory: no
// `blocked` field, no re-injection, no filesystem write (AC-D05).
type PinsResult struct {
	Survived bool        `json:"survived"`
	Pins     []PinStatus `json:"pins"`
	Missing  []string    `json:"missing"`
}

// Pins runs the verify_pins tool over in.
func Pins(in PinsInput) (PinsResult, error) {
	artifact, err := resolveArtifact(in)
	if err != nil {
		return PinsResult{}, err
	}

	result := PinsResult{Survived: true, Pins: []PinStatus{}, Missing: []string{}}
	for _, p := range in.Pins {
		present, err := pinPresent(p, artifact)
		if err != nil {
			return PinsResult{}, err
		}
		result.Pins = append(result.Pins, PinStatus{ID: p.ID, Present: present})
		if !present {
			result.Survived = false
			result.Missing = append(result.Missing, p.ID)
		}
	}
	return result, nil
}

// pinPresent decides survival: a nil marker defaults to a literal match on
// the pin's own id token (AC-D03); an explicit marker is a Go regexp
// (AC-D04).
func pinPresent(p Pin, artifact string) (bool, error) {
	if p.Marker == nil || *p.Marker == "" {
		return strings.Contains(artifact, p.ID), nil
	}
	re, err := regexp.Compile(*p.Marker)
	if err != nil {
		return false, err
	}
	return re.MatchString(artifact), nil
}

func resolveArtifact(in PinsInput) (string, error) {
	if in.Artifact != nil {
		return *in.Artifact, nil
	}
	if in.ArtifactPath != "" {
		data, err := os.ReadFile(in.ArtifactPath)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return "", nil
}
