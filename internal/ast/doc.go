// Package ast provides Go abstract syntax tree (AST) parsing and function extraction
// capabilities for the similarity detection tool.
//
// This package implements high-performance AST analysis using Go's standard library
// with thread-safe operations and comprehensive function metadata extraction.
//
// Key Components:
//   - Parser: Parses Go source files and extracts function declarations
//   - Function: Thread-safe function representation with metadata and AST data
//   - Normalization: AST structure normalization for accurate comparison
//
// The Parser efficiently processes Go files using the go/ast and go/parser packages,
// extracting detailed function information including line numbers, signatures, and
// complete AST representations.
//
// Thread Safety:
// All operations are designed to be thread-safe using sync.RWMutex for concurrent
// access patterns, preventing data races during parallel processing.
//
// Example Usage:
//
//	parser := NewParser()
//	result := parser.ParseFile("example.go")
//	if result.IsOk() {
//		functions := result.Unwrap()
//		for _, fn := range functions {
//			fmt.Printf("Function: %s (lines %d-%d)\n",
//				fn.Name, fn.StartLine, fn.EndLine)
//		}
//	}
package ast
