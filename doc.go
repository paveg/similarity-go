// Package similarity-go provides a high-performance Go code similarity detection CLI tool
// that uses multi-factor AST analysis to identify duplicate and similar code patterns.
//
// The tool employs sophisticated algorithms combining AST tree edit distance, token sequence
// analysis, structural signatures, and signature matching with weighted scoring to provide
// accurate similarity detection for Go projects of all sizes.
//
// Key Features:
//   - Multi-factor similarity detection with configurable thresholds
//   - High-performance parallel processing with thread-safe operations
//   - Intelligent directory scanning with smart filtering
//   - JSON/YAML structured output formats
//   - Comprehensive configuration management
//   - Production-ready with 78-88% test coverage
//
// Basic Usage:
//
//	// Analyze entire directory
//	similarity-go ./internal
//
//	// Analyze specific files
//	similarity-go file1.go file2.go
//
//	// Custom threshold and output format
//	similarity-go --threshold 0.7 --format yaml ./project
//
// The tool is designed for large-scale refactoring, code review, and quality metrics,
// providing actionable insights for code maintainability and duplication reduction.
package main
