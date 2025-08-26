package ast

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
)

// Function represents a Go function with its metadata and AST representation.
type Function struct {
	Name       string        // Function name
	File       string        // Source file path
	StartLine  int           // Starting line number
	EndLine    int           // Ending line number
	AST        *ast.FuncDecl // Original AST node
	Normalized *ast.FuncDecl // Normalized AST for comparison
	hash       string        // Cached structure hash
	signature  string        // Cached function signature
	LineCount  int           // Number of lines in the function
}

// GetSignature returns the function signature as a string.
// The signature is cached after first computation.
func (f *Function) GetSignature() string {
	if f.signature != "" {
		return f.signature
	}

	if f.AST == nil || f.AST.Type == nil {
		f.signature = "func()"
		return f.signature
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, token.NewFileSet(), f.AST.Type); err != nil {
		f.signature = "func()"
		return f.signature
	}

	f.signature = buf.String()
	return f.signature
}

// GetSource returns the complete source code of the function.
func (f *Function) GetSource() (string, error) {
	if f.AST == nil {
		return "", nil
	}

	var buf bytes.Buffer
	err := format.Node(&buf, token.NewFileSet(), f.AST)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// IsValid checks if the function meets the minimum requirements for analysis.
// It must have a non-nil body and meet the minimum line count.
func (f *Function) IsValid(minLines int) bool {
	// Check if function has a body (not an interface method declaration)
	if f.AST == nil || f.AST.Body == nil {
		return false
	}

	// Check minimum line count
	if f.LineCount < minLines {
		return false
	}

	return true
}

// Hash returns a structural hash of the function for quick comparison.
// This will be implemented when we add the hasher component.
func (f *Function) Hash() string {
	if f.hash != "" {
		return f.hash
	}

	// TODO: Implement structural hashing
	// For now, return a placeholder
	f.hash = "placeholder_hash"
	return f.hash
}

// Normalize returns a normalized version of the function for comparison.
// This will be implemented when we add the normalizer component.
func (f *Function) Normalize() *Function {
	if f.Normalized != nil {
		return &Function{
			Name:       f.Name,
			File:       f.File,
			StartLine:  f.StartLine,
			EndLine:    f.EndLine,
			AST:        f.Normalized,
			Normalized: f.Normalized,
			LineCount:  f.LineCount,
		}
	}

	// TODO: Implement normalization
	// For now, return a copy
	f.Normalized = f.AST
	return f
}
