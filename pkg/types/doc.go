// Package types provides Rust-inspired utility types for robust error handling
// and null safety in Go applications.
//
// This package implements Optional[T] and Result[T] types that bring functional
// programming patterns to Go, enabling more expressive and safer error handling.
//
// Core Types:
//   - Optional[T]: Represents a value that may or may not be present (Rust's Option)
//   - Result[T]: Represents either a successful value or an error (Rust's Result)
//
// Optional[T] Usage:
// The Optional type helps eliminate null pointer exceptions and makes the
// possibility of missing values explicit in the type system.
//
//	userOpt := types.Some(user)
//	if userOpt.IsSome() {
//		user := userOpt.Unwrap()
//		// safely use user
//	}
//
//	emptyOpt := types.None[User]()
//	defaultUser := emptyOpt.UnwrapOr(defaultUser)
//
// Result[T] Usage:
// The Result type makes error handling explicit and composable, reducing
// the need for traditional Go error checking patterns in certain contexts.
//
//	result := parseConfig(filename)
//	if result.IsOk() {
//		config := result.Unwrap()
//		// use config
//	} else {
//		err := result.Error()
//		// handle error
//	}
//
//	// Chainable operations
//	final := result.Map(processConfig).AndThen(validateConfig)
//
// These types integrate seamlessly with traditional Go error handling while
// providing additional expressiveness for complex data flow scenarios.
package types
