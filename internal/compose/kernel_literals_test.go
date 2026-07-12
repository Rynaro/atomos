package compose

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestKernelLiteralsAreConstOnly is the structural guard that makes
// kernel_literals.go's allowlist entry (internal/tools/registry_test.go)
// safe rather than a hole: it parses the file with go/ast and fails the
// build the moment it contains anything but bare `const NAME = "string
// literal"` declarations — no imports, no funcs, no vars, no types, and no
// computed/concatenated/referenced values within a const either. An
// allowlisted file without this guard would be a standing invitation to
// smuggle real logic (an actual policy evaluator, a durable-memory client)
// past the fence inside a file the deny-list never looks at; this test is
// what keeps that door shut.
func TestKernelLiteralsAreConstOnly(t *testing.T) {
	const path = "kernel_literals.go"

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	if len(file.Imports) != 0 {
		t.Errorf("%s: must have zero imports, found %d", path, len(file.Imports))
	}

	if len(file.Decls) == 0 {
		t.Fatalf("%s: expected at least one const declaration, found none", path)
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			t.Fatalf("%s: contains a non-GenDecl top-level declaration (%T) — only `const` declarations are allowed", path, decl)
		}
		if genDecl.Tok != token.CONST {
			t.Fatalf("%s: contains a %q declaration — only `const` declarations are allowed (no var, no type, no func)", path, genDecl.Tok)
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				t.Fatalf("%s: const spec is not a ValueSpec (%T)", path, spec)
			}
			if valueSpec.Type != nil {
				t.Errorf("%s: const %v has an explicit type %v — only untyped string literals are allowed", path, valueSpec.Names, valueSpec.Type)
			}
			if len(valueSpec.Values) != len(valueSpec.Names) {
				t.Fatalf("%s: const spec has %d names but %d values", path, len(valueSpec.Names), len(valueSpec.Values))
			}
			for i, value := range valueSpec.Values {
				lit, ok := value.(*ast.BasicLit)
				if !ok {
					t.Errorf("%s: const %s's value is a %T, not a bare string literal — no concatenation, no function calls, no identifier references allowed in this file", path, valueSpec.Names[i].Name, value)
					continue
				}
				if lit.Kind != token.STRING {
					t.Errorf("%s: const %s is a %s literal, not a string literal", path, valueSpec.Names[i].Name, lit.Kind)
				}
			}
		}
	}
}
