package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AC-G02: the ATOMOS_VERSION file contains exactly 0.1.0, matching the
// binary's compiled Version constant.
func TestVersionMatchesVersionFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "ATOMOS_VERSION"))
	if err != nil {
		t.Fatalf("read ATOMOS_VERSION: %v", err)
	}
	fileVersion := strings.TrimSpace(string(data))
	if fileVersion != Version {
		t.Errorf("ATOMOS_VERSION file = %q, want %q (main.Version)", fileVersion, Version)
	}
}
