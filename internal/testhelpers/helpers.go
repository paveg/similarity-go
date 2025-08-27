package testhelpers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	astpkg "github.com/paveg/similarity-go/internal/ast"
)

// CreateTempGoFile creates a temporary Go file with the given content for testing.
func CreateTempGoFile(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.go")

	err := os.WriteFile(tempFile, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return tempFile
}

// ParseGoSource parses Go source code and returns the AST.
func ParseGoSource(t *testing.T, source string) (*token.FileSet, *ast.File) {
	t.Helper()

	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "test.go", source, 0)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	return fileSet, file
}

// ExtractFunctionFromSource parses source and extracts the first function with the given name.
func ExtractFunctionFromSource(t *testing.T, source, funcName string) *ast.FuncDecl {
	t.Helper()

	_, file := ParseGoSource(t, source)

	var funcDecl *ast.FuncDecl

	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Name.Name == funcName {
			funcDecl = fd

			return false
		}

		return true
	})

	if funcDecl == nil {
		t.Fatalf("Function %s not found in source", funcName)
	}

	return funcDecl
}

// CreateFunctionFromSource creates a Function instance from Go source code.
// Returns nil if the function is not found instead of failing the test.
func CreateFunctionFromSource(t *testing.T, source, funcName string) *astpkg.Function {
	t.Helper()

	fileSet, file := ParseGoSource(t, source)

	var funcDecl *ast.FuncDecl

	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Name.Name == funcName {
			funcDecl = fd
			return false
		}
		return true
	})

	if funcDecl == nil {
		return nil // Return nil instead of failing the test
	}

	return &astpkg.Function{
		Name:      funcName,
		File:      "test.go",
		AST:       funcDecl,
		StartLine: fileSet.Position(funcDecl.Pos()).Line,
		EndLine:   fileSet.Position(funcDecl.End()).Line,
		LineCount: fileSet.Position(funcDecl.End()).Line - fileSet.Position(funcDecl.Pos()).Line + 1,
	}
}

// AssertNoError is a helper to assert that an error is nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError is a helper to assert that an error is not nil.
func AssertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// AssertEqual is a generic helper to assert equality.
func AssertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()

	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEmpty is a helper to assert that a string is not empty.
func AssertNotEmpty(t *testing.T, value string) {
	t.Helper()

	if value == "" {
		t.Error("Expected non-empty string")
	}
}

// AssertContains is a helper to assert that a string contains a substring.
func AssertContains(t *testing.T, haystack, needle string) {
	t.Helper()

	if !contains(haystack, needle) {
		t.Errorf("Expected %q to contain %q", haystack, needle)
	}
}

// contains checks if a string contains a substring (helper for AssertContains).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr || containsAt(s, substr, 1)))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}

	if start+len(substr) > len(s) {
		return containsAt(s, substr, start+1)
	}

	if s[start:start+len(substr)] == substr {
		return true
	}

	return containsAt(s, substr, start+1)
}

// AbsFloat returns the absolute value of a float64.
func AbsFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
