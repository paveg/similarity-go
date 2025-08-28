// Package similarity implements multi-factor similarity detection algorithms
// for Go code analysis and duplicate code identification.
//
// This package provides sophisticated similarity detection using a weighted
// combination of four key metrics to achieve high-precision code similarity analysis.
//
// Multi-Factor Algorithm Components:
//   - AST Tree Edit Distance (30% weight): Dynamic programming-based tree comparison
//   - Token Sequence Analysis (30% weight): Levenshtein distance on normalized tokens
//   - Structural Signatures (25% weight): Function body structure comparison
//   - Function Signatures (15% weight): Parameter and return type analysis
//
// The Detector class orchestrates the similarity analysis process, providing
// configurable thresholds and performance optimizations including early termination
// and hash-based deduplication.
//
// Key Features:
//   - Configurable similarity weights and thresholds
//   - Performance optimizations for large codebases
//   - Thread-safe operations for concurrent processing
//   - Comprehensive similarity scoring and grouping
//
// Example Usage:
//
//	detector := NewDetector(0.8) // 80% similarity threshold
//	matches := detector.FindSimilarFunctions(functions)
//	for _, group := range matches {
//		fmt.Printf("Found %d similar functions with score %.2f\n",
//			len(group.Functions), group.SimilarityScore)
//	}
//
// The algorithm is optimized for Go-specific patterns and provides actionable
// insights for code refactoring and maintainability improvements.
package similarity
