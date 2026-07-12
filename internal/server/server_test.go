package server

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Rynaro/atomos/internal/compose"
	"github.com/Rynaro/atomos/internal/tools"
)

func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()
	ct, st := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := New().Connect(ctx, st, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "atomos-test-client", Version: "0.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// AC-A01: the server completes the MCP handshake identifying itself as
// "atomos" with the ATOMOS_VERSION version string.
func TestInitializeHandshake(t *testing.T) {
	cs := connect(t)
	res := cs.InitializeResult()
	if res == nil || res.ServerInfo == nil {
		t.Fatalf("no InitializeResult/ServerInfo returned")
	}
	if res.ServerInfo.Name != ServerName {
		t.Errorf("ServerInfo.Name = %q, want %q", res.ServerInfo.Name, ServerName)
	}
	if res.ServerInfo.Version != Version {
		t.Errorf("ServerInfo.Version = %q, want %q", res.ServerInfo.Version, Version)
	}
}

// AC-A02: tools/list advertises exactly compose_handoff, verify_envelope,
// verify_pins — no other tool, matching internal/tools.Registry.
func TestToolsListMatchesRegistry(t *testing.T) {
	cs := connect(t)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	var got []string
	for _, tl := range res.Tools {
		got = append(got, tl.Name)
	}
	want := append([]string(nil), tools.Names()...)
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("tools/list returned %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("tools/list returned %v, want %v", got, want)
		}
	}
}

// AC-A04: tools/call naming a tool outside the closed set returns a
// tool-not-found error without executing anything.
func TestUnknownToolRejected(t *testing.T) {
	cs := connect(t)
	_, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "context_meter",
		Arguments: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected an error calling an unregistered tool, got nil")
	}
}

// AC-H19: a compose_externalize_manifest call omitting both created_at and
// ts has both filled at the server seam from a SINGLE UTC clock reading —
// compose.ExternalizeManifest itself never calls time.Now (AC-H18).
func TestManifestTimestampSeam(t *testing.T) {
	in := compose.ManifestInput{}
	resolveManifestTimestamps(&in)
	if in.CreatedAt == "" || in.TS == "" {
		t.Fatalf("expected both CreatedAt and TS to be filled, got CreatedAt=%q TS=%q", in.CreatedAt, in.TS)
	}
	createdAt, err := time.Parse("2006-01-02T15:04:05Z", in.CreatedAt)
	if err != nil {
		t.Fatalf("parse CreatedAt: %v", err)
	}
	ts, err := time.Parse("20060102T150405Z", in.TS)
	if err != nil {
		t.Fatalf("parse TS: %v", err)
	}
	if !createdAt.Equal(ts) {
		t.Errorf("CreatedAt and TS come from different clock readings: %v vs %v", createdAt, ts)
	}

	// A caller that already supplied both is left untouched (parity
	// fixtures depend on this: frozen created_at/ts must never be
	// overwritten).
	in2 := compose.ManifestInput{CreatedAt: "2020-01-01T00:00:00Z", TS: "20200101T000000Z"}
	resolveManifestTimestamps(&in2)
	if in2.CreatedAt != "2020-01-01T00:00:00Z" || in2.TS != "20200101T000000Z" {
		t.Errorf("resolveManifestTimestamps overwrote caller-supplied values: %+v", in2)
	}
}
