// Command atomos is the stdio MCP entrypoint for the ECM compose/verify
// executor (FORGE ADR, settled 2026-07-07):
//
//	atomos serve                  -> stdio MCP server (the closed 4-tool set)
//	atomos version | --version | -h | --help
//
// atomos is an ALTERNATE surface to the always-canonical
// `eidolons context …` bash kernel: same inputs => same bytes. It is
// compose/verify ONLY — no session-budget tracking, no rule-table
// evaluation, no operation firing, no durable-storage calls — see
// internal/tools.Registry for the closed set and README.md for the fence.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Rynaro/atomos/internal/server"
)

// Version is the atomos build/release version, test-pinned to the
// ATOMOS_VERSION file (AC-V01) and to internal/server.Version (AC-V02).
// Never stamped into a composed envelope's from.version (AC-B05) — that
// field echoes the caller's from_version input.
const Version = "0.2.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		if err := server.Serve(context.Background()); err != nil {
			fmt.Fprintln(os.Stderr, "atomos serve:", err)
			os.Exit(1)
		}
	case "version", "--version":
		fmt.Println(Version)
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "atomos: unknown command %q\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `atomos — ECM (Eidolons Context Management) compose/verify executor MCP

Usage:
  atomos serve
      Run the stdio MCP server (compose_handoff, verify_envelope, verify_pins,
      compose_externalize_manifest).

  atomos version | --version | -h | --help
      Print the ATOMOS_VERSION string.
`)
}
