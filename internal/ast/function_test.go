package ast

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
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
			expected: "func(a int, b int) int",
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
			expected: "func(a int, b int) (int, error)",
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

			fn := &Function{
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

	fn := &Function{
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
		function *Function
		minLines int
		expected bool
	}{
		{
			name: "valid function above minimum lines",
			function: &Function{
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
			function: &Function{
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
			function: &Function{
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

// Helper functions
func createFunctionFromSource(t *testing.T, source, funcName string) *Function {
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
		t.Fatalf("Function %s not found", funcName)
	}

	return &Function{
		Name:      funcName,
		File:      "test.go",
		AST:       funcDecl,
		StartLine: fileSet.Position(funcDecl.Pos()).Line,
		EndLine:   fileSet.Position(funcDecl.End()).Line,
		LineCount: fileSet.Position(funcDecl.End()).Line - fileSet.Position(funcDecl.Pos()).Line + 1,
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr, 1)))
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
