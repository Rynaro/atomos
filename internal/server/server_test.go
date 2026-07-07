package server

import (
	"context"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

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
