// Package types provides common type definitions and utilities for the similarity-go project.
package types //nolint:revive // types is an appropriate name for utility type definitions

import "fmt"

// Result represents a result that can either be a success value or an error.
// This is inspired by Rust's Result<T, E> type for better error handling.
type Result[T any] struct {
	value T
	err   error
	isOk  bool
}

// Ok creates a successful Result containing the given value.
func Ok[T any](value T) Result[T] {
	return Result[T]{
		value: value,
		err:   nil,
		isOk:  true,
	}
}

// Err creates an error Result containing the given error.
func Err[T any](err error) Result[T] {
	var zero T
	return Result[T]{
		value: zero,
		err:   err,
		isOk:  false,
	}
}

// IsOk returns true if the Result contains a success value.
func (r Result[T]) IsOk() bool {
	return r.isOk
}

// IsErr returns true if the Result contains an error.
func (r Result[T]) IsErr() bool {
	return !r.isOk
}

// Unwrap returns the success value if the Result is Ok, or panics if it contains an error.
// Use this only when you're certain the Result is Ok.
func (r Result[T]) Unwrap() T {
	if !r.isOk {
		panic(fmt.Sprintf("called Unwrap on an Err result: %v", r.err))
	}
	return r.value
}

// UnwrapOr returns the success value if the Result is Ok, or the provided default value if it's an error.
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.isOk {
		return r.value
	}
	return defaultValue
}

// Error returns the error if the Result contains one, or nil if it's a success.
func (r Result[T]) Error() error {
	return r.err
}

// Map applies a function to the success value if the Result is Ok, returning a new Result.
// If the Result is an error, it returns a new error Result of the target type.
func Map[T, U any](r Result[T], fn func(T) U) Result[U] {
	if r.isOk {
		return Ok(fn(r.value))
	}
	return Err[U](r.err)
}

// AndThen applies a function that returns a Result to the success value if the Result is Ok.
// This is useful for chaining operations that might fail.
func AndThen[T, U any](r Result[T], fn func(T) Result[U]) Result[U] {
	if r.isOk {
		return fn(r.value)
	}
	return Err[U](r.err)
}
