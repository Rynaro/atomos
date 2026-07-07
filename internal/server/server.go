// Package server wires atomos's closed 3-tool set (internal/tools.Registry)
// onto an official Go MCP SDK stdio server (mirrors tonberry's
// internal/mcpserver). This is the ONE place in the codebase allowed to call
// time.Now — the single server-layer seam that defaults compose_handoff's
// ts/iso_ts when the caller omits them (Track B; AC-F04 keeps every handler
// package pure).
package server

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Rynaro/atomos/internal/compose"
	"github.com/Rynaro/atomos/internal/tools"
	"github.com/Rynaro/atomos/internal/verify"
)

// ServerName is the MCP server name and its host registration key.
const ServerName = "atomos"

// Version is the atomos build version reported to MCP clients. Kept in sync
// with ATOMOS_VERSION and cmd/atomos.Version by convention (mirrors
// tonberry's dual-declared Version constants).
const Version = "0.1.0"

// New constructs the MCP server with exactly the tools.Registry set
// registered — the only tool-registration call site in the codebase.
func New() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    ServerName,
		Version: Version,
		Title:   "atomos — ECM compose/verify executor MCP",
	}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "compose_handoff",
		Description: tools.Description("compose_handoff"),
	}, func(_ context.Context, _ *mcp.CallToolRequest, in compose.HandoffInput) (*mcp.CallToolResult, compose.HandoffResult, error) {
		resolveTimestamps(&in)
		out, err := compose.Handoff(in)
		return errResult(err), out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "verify_envelope",
		Description: tools.Description("verify_envelope"),
	}, func(_ context.Context, _ *mcp.CallToolRequest, in verify.EnvelopeInput) (*mcp.CallToolResult, verify.EnvelopeResult, error) {
		out, err := verify.Envelope(in)
		return errResult(err), out, err
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "verify_pins",
		Description: tools.Description("verify_pins"),
	}, func(_ context.Context, _ *mcp.CallToolRequest, in verify.PinsInput) (*mcp.CallToolResult, verify.PinsResult, error) {
		out, err := verify.Pins(in)
		return errResult(err), out, err
	})

	return s
}

// Serve runs the stdio MCP server until the client disconnects or ctx is
// cancelled.
func Serve(ctx context.Context) error {
	return New().Run(ctx, &mcp.StdioTransport{})
}

// resolveTimestamps is the single wall-clock seam (Track B / AC-F04):
// compose.Handoff itself never calls time.Now. ts/iso_ts are frozen inputs
// for parity fixtures; in live use, an omitted pair is filled here from one
// shared `now` reading.
func resolveTimestamps(in *compose.HandoffInput) {
	if in.TS != "" && in.ISOTS != "" {
		return
	}
	now := time.Now().UTC()
	if in.TS == "" {
		in.TS = now.Format("20060102T150405Z")
	}
	if in.ISOTS == "" {
		in.ISOTS = now.Format("2006-01-02T15:04:05Z")
	}
}

// errResult builds the unstructured CallToolResult on a tool error, matching
// the SDK guidance that tool-origin errors ride the result, not the
// protocol (mirrors tonberry's mcpserver.result).
func errResult(err error) *mcp.CallToolResult {
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		}
	}
	return nil // SDK fills Content from the structured Out when result is nil
}
