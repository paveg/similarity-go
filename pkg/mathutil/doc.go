// Package mathutil provides type-safe generic mathematical utility functions
// using Go 1.21+ generics for zero-cost abstractions.
//
// This package consolidates common mathematical operations with compile-time
// type safety, eliminating code duplication across the similarity-go codebase.
//
// Generic Functions:
//   - Min[T constraints.Ordered]: Returns the minimum of two comparable values
//   - Max[T constraints.Ordered]: Returns the maximum of two comparable values
//   - Abs[T Number]: Returns the absolute value of numeric types
//
// All functions are implemented with zero-cost abstractions, providing
// optimal performance through compile-time type checking and inline optimization.
//
// Type Constraints:
// The functions use Go's type constraints to ensure type safety while
// supporting all comparable and numeric types including integers, floats,
// and custom types implementing the required interfaces.
//
// Example Usage:
//
//	// Works with any ordered type
//	maxInt := mathutil.Max(10, 20)        // returns 20
//	minFloat := mathutil.Min(3.14, 2.71)  // returns 2.71
//	absValue := mathutil.Abs(-42)         // returns 42
//
//	// Also works with custom types
//	type Score int
//	highScore := mathutil.Max(Score(100), Score(85))
//
// These utilities eliminate the need for type-specific implementations
// while maintaining full type safety and optimal performance characteristics.
package mathutil
