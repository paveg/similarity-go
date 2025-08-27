package ast

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
func (f *Function) Hash() string {
	if f.hash != "" {
		return f.hash
	}

	if f.AST == nil {
		// Include function name in hash even for nil AST
		hashComponents := []string{
			f.Name,
			f.GetSignature(),
			fmt.Sprintf("lines:%d-%d", f.StartLine, f.EndLine),
			fmt.Sprintf("count:%d", f.LineCount),
			"nil_ast",
		}

		combined := fmt.Sprintf("%v", hashComponents)
		hasher := sha256.New()
		hasher.Write([]byte(combined))
		f.hash = hex.EncodeToString(hasher.Sum(nil))[:16]

		return f.hash
	}

	// Create a structural hash based on function signature and basic structure
	hashComponents := []string{
		f.Name,
		f.GetSignature(),
		fmt.Sprintf("lines:%d-%d", f.StartLine, f.EndLine),
		fmt.Sprintf("count:%d", f.LineCount),
	}

	// Add body structure if available
	if f.AST.Body != nil {
		hashComponents = append(hashComponents, fmt.Sprintf("stmts:%d", len(f.AST.Body.List)))
	}

	// Combine all components and create SHA256 hash
	combined := fmt.Sprintf("%v", hashComponents)
	hasher := sha256.New()
	hasher.Write([]byte(combined))
	f.hash = hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars for readability

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

	if f.AST == nil {
		return f
	}

	// Create a normalized copy by removing variable names and other non-structural elements
	// For now, use the original AST as a placeholder until full normalization is implemented
	f.Normalized = f.AST

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
