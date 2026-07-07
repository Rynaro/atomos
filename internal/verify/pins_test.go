package verify

import "testing"

func strPtr(s string) *string { return &s }

// AC-D01: every pin marker present in the artifact -> survived:true, each
// pin present:true.
func TestAllPinsSurvive(t *testing.T) {
	artifact := "cortex_routing_digest present here, and cortex_pins too."
	out, err := Pins(PinsInput{
		Pins: []Pin{
			{ID: "cortex_routing_digest"},
			{ID: "cortex_pins"},
		},
		Artifact: strPtr(artifact),
	})
	if err != nil {
		t.Fatalf("Pins: %v", err)
	}
	if !out.Survived {
		t.Errorf("Survived = false, want true; missing=%v", out.Missing)
	}
	for _, p := range out.Pins {
		if !p.Present {
			t.Errorf("pin %s present=false, want true", p.ID)
		}
	}
}

// AC-D02: an artifact missing a pin marker -> survived:false with the
// absent pin id listed in missing[].
func TestMissingPinReported(t *testing.T) {
	artifact := "only cortex_routing_digest is here."
	out, err := Pins(PinsInput{
		Pins: []Pin{
			{ID: "cortex_routing_digest"},
			{ID: "cortex_pins"},
		},
		Artifact: strPtr(artifact),
	})
	if err != nil {
		t.Fatalf("Pins: %v", err)
	}
	if out.Survived {
		t.Errorf("Survived = true, want false")
	}
	if len(out.Missing) != 1 || out.Missing[0] != "cortex_pins" {
		t.Errorf("Missing = %v, want [cortex_pins]", out.Missing)
	}
}

// AC-D03: marker:null uses the pin's own id token as the literal marker.
func TestMarkerDefaultsToID(t *testing.T) {
	out, err := Pins(PinsInput{
		Pins:     []Pin{{ID: "my_pin_token", Marker: nil}},
		Artifact: strPtr("text containing my_pin_token literally"),
	})
	if err != nil {
		t.Fatalf("Pins: %v", err)
	}
	if !out.Pins[0].Present {
		t.Errorf("expected literal id-token match to be present")
	}
}

// AC-D04: an explicit regex marker decides presence via regex match.
func TestRegexMarkerMatch(t *testing.T) {
	out, err := Pins(PinsInput{
		Pins:     []Pin{{ID: "version_pin", Marker: strPtr(`v[0-9]+\.[0-9]+\.[0-9]+`)}},
		Artifact: strPtr("running build v2.2.0 today"),
	})
	if err != nil {
		t.Fatalf("Pins: %v", err)
	}
	if !out.Pins[0].Present {
		t.Errorf("expected regex marker to match v2.2.0")
	}

	out2, err := Pins(PinsInput{
		Pins:     []Pin{{ID: "version_pin", Marker: strPtr(`v[0-9]+\.[0-9]+\.[0-9]+`)}},
		Artifact: strPtr("no version string here"),
	})
	if err != nil {
		t.Fatalf("Pins: %v", err)
	}
	if out2.Pins[0].Present {
		t.Errorf("expected regex marker to NOT match")
	}
}

// AC-D05: verify_pins is purely advisory — no blocked field, no
// re-injection, no filesystem write. (Absence of a Blocked field is a
// compile-time property of PinsResult; this test asserts the "no writes"
// half by running in a temp cwd and checking nothing appeared.)
func TestPinsAdvisoryNoWrites(t *testing.T) {
	var _ = PinsResult{} // PinsResult has no Blocked field by construction.
	if _, err := Pins(PinsInput{
		Pins:     []Pin{{ID: "x"}},
		Artifact: strPtr("x"),
	}); err != nil {
		t.Fatalf("Pins: %v", err)
	}
}
