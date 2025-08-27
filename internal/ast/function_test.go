package ast_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	astpkg "github.com/paveg/similarity-go/internal/ast"
)

func TestFunction_GetSignature(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name: "simple function",
			source: `package main
func add(a, b int) int {
	return a + b
}`,
			expected: "func(a, b int) int",
		},
		{
			name: "function with no parameters",
			source: `package main
func hello() {
	fmt.Println("hello")
}`,
			expected: "func()",
		},
		{
			name: "function with multiple return values",
			source: `package main
func divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}`,
			expected: "func(a, b int) (int, error)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileSet := token.NewFileSet()

			file, err := parser.ParseFile(fileSet, "test.go", tt.source, 0)
			if err != nil {
				t.Fatalf("Failed to parse source: %v", err)
			}

			var funcDecl *ast.FuncDecl

			ast.Inspect(file, func(n ast.Node) bool {
				if fd, ok := n.(*ast.FuncDecl); ok && fd.Name.Name != "main" {
					funcDecl = fd

					return false
				}

				return true
			})

			if funcDecl == nil {
				t.Fatal("No function declaration found")
			}

			fn := &astpkg.Function{
				Name:      funcDecl.Name.Name,
				File:      "test.go",
				AST:       funcDecl,
				StartLine: fileSet.Position(funcDecl.Pos()).Line,
				EndLine:   fileSet.Position(funcDecl.End()).Line,
			}

			got := fn.GetSignature()
			if got != tt.expected {
				t.Errorf("Expected signature %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestFunction_GetSource(t *testing.T) {
	source := `package main
func add(a, b int) int {
	return a + b
}`

	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "test.go", source, 0)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	var funcDecl *ast.FuncDecl

	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Name.Name == "add" {
			funcDecl = fd

			return false
		}

		return true
	})

	fn := &astpkg.Function{
		Name: "add",
		File: "test.go",
		AST:  funcDecl,
	}

	got, err := fn.GetSource()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if got == "" {
		t.Error("Expected non-empty source code")
	}

	// Source should contain the function signature and body
	if !contains(got, "func add(a, b int) int") {
		t.Errorf("Expected source to contain function signature, got: %s", got)
	}
}

func TestFunction_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		function *astpkg.Function
		minLines int
		expected bool
	}{
		{
			name: "valid function above minimum lines",
			function: &astpkg.Function{
				Name:      "test",
				File:      "test.go",
				StartLine: 1,
				EndLine:   10,
				LineCount: 10,
				AST:       &ast.FuncDecl{Body: &ast.BlockStmt{}}, // Non-nil body
			},
			minLines: 5,
			expected: true,
		},
		{
			name: "function below minimum lines",
			function: &astpkg.Function{
				Name:      "test",
				File:      "test.go",
				StartLine: 1,
				EndLine:   3,
				LineCount: 3,
				AST:       &ast.FuncDecl{Body: &ast.BlockStmt{}},
			},
			minLines: 5,
			expected: false,
		},
		{
			name: "function with nil body (interface method)",
			function: &astpkg.Function{
				Name:      "test",
				File:      "test.go",
				StartLine: 1,
				EndLine:   10,
				LineCount: 10,
				AST:       &ast.FuncDecl{Body: nil}, // Nil body
			},
			minLines: 5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.function.IsValid(tt.minLines)
			if got != tt.expected {
				t.Errorf("Expected IsValid() to be %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestFunction_Hash(t *testing.T) {
	source1 := `package main
func add(a, b int) int {
	return a + b
}`

	source2 := `package main
func add(x, y int) int {
	return x + y
}`

	fn1 := createFunctionFromSource(t, source1, "add")
	fn2 := createFunctionFromSource(t, source2, "add")

	hash1 := fn1.Hash()
	hash2 := fn2.Hash()

	if hash1 == "" {
		t.Error("Expected non-empty hash for function 1")
	}

	if hash2 == "" {
		t.Error("Expected non-empty hash for function 2")
	}
	// Same structure with different variable names should have same hash after normalization
	// This will be implemented when we add the Hash method
}

func TestFunction_Normalize(t *testing.T) {
	source := `package main
func add(a, b int) int {
	return a + b
}`

	fn := createFunctionFromSource(t, source, "add")

	// Test first call to Normalize
	normalized1 := fn.Normalize()
	if normalized1 == nil {
		t.Error("Expected non-nil normalized function")
		return
	}

	if normalized1.Name != fn.Name {
		t.Errorf("Expected name %s, got %s", fn.Name, normalized1.Name)
	}

	// Test second call returns cached version
	fn.Normalized = fn.AST // Simulate cached normalization

	normalized2 := fn.Normalize()
	if normalized2 == nil {
		t.Error("Expected non-nil normalized function from cache")
		return
	}

	if normalized2.Name != fn.Name {
		t.Errorf("Expected cached name %s, got %s", fn.Name, normalized2.Name)
	}
}

func TestFunction_GetSignature_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		function *astpkg.Function
		expected string
	}{
		{
			name: "function with nil AST",
			function: &astpkg.Function{
				Name: "test",
				File: "test.go",
				AST:  nil,
			},
			expected: "func()",
		},
		{
			name: "function with AST but nil Type",
			function: &astpkg.Function{
				Name: "test",
				File: "test.go",
				AST:  &ast.FuncDecl{Type: nil},
			},
			expected: "func()",
		},
		{
			name: "function with nil AST and Type",
			function: &astpkg.Function{
				Name: "test",
				File: "test.go",
				AST:  &ast.FuncDecl{Type: nil},
			},
			expected: "func()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.function.GetSignature()
			if got != tt.expected {
				t.Errorf("Expected signature %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestFunction_GetSource_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		function       *astpkg.Function
		expectedSource string
		expectError    bool
	}{
		{
			name: "function with nil AST",
			function: &astpkg.Function{
				Name: "test",
				File: "test.go",
				AST:  nil,
			},
			expectedSource: "",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := tt.function.GetSource()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if source != tt.expectedSource {
				t.Errorf("Expected source %q, got %q", tt.expectedSource, source)
			}
		})
	}
}

func TestFunction_Hash_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		function *astpkg.Function
		expected string
	}{
		{
			name: "function with nil AST returns nil_ast_hash",
			function: &astpkg.Function{
				Name: "test",
				File: "test.go",
				AST:  nil,
			},
			expected: "nil_ast_hash",
		},
		{
			name: "function with nil AST returns nil_ast_hash",
			function: &astpkg.Function{
				Name: "test2",
				File: "test.go",
				AST:  nil,
			},
			expected: "nil_ast_hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.function.Hash()
			if got != tt.expected {
				t.Errorf("Expected hash %q, got %q", tt.expected, got)
			}
		})
	}
}

// Helper functions.
func createFunctionFromSource(t *testing.T, source, funcName string) *astpkg.Function {
	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "test.go", source, 0)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	var funcDecl *ast.FuncDecl

	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Name.Name == funcName {
			funcDecl = fd

			return false
		}

		return true
	})

	if funcDecl == nil {
		t.Fatalf("astpkg.Function %s not found", funcName)
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

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr, 1)))
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
