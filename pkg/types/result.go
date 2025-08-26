package types

// Result represents a value that can be either successful (Ok) or an error (Err).
// This follows the Rust-like Result pattern for explicit error handling.
type Result[T any] struct {
	value *T
	err   error
}

// Ok creates a successful Result containing the given value.
func Ok[T any](value T) Result[T] {
	return Result[T]{value: &value, err: nil}
}

// Err creates a failed Result containing the given error.
func Err[T any](err error) Result[T] {
	return Result[T]{value: nil, err: err}
}

// IsOk returns true if the Result contains a successful value.
func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// IsErr returns true if the Result contains an error.
func (r Result[T]) IsErr() bool {
	return r.err != nil
}

// Unwrap returns the contained value if Ok, panics if Err.
// Use this only when you are certain the Result is Ok.
func (r Result[T]) Unwrap() T {
	if r.err != nil {
		panic(r.err)
	}
	return *r.value
}

// UnwrapOr returns the contained value if Ok, otherwise returns the default value.
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.err != nil {
		return defaultValue
	}
	return *r.value
}

// Error returns the contained error if Err, nil if Ok.
func (r Result[T]) Error() error {
	return r.err
}

// Map applies a function to the contained value if Ok, otherwise returns the error as-is.
func Map[T, U any](r Result[T], mapper func(T) U) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return Ok(mapper(*r.value))
}

// AndThen applies a function that returns a Result to the contained value if Ok.
// This is useful for chaining operations that can fail.
func AndThen[T, U any](r Result[T], f func(T) Result[U]) Result[U] {
	if r.err != nil {
		return Err[U](r.err)
	}
	return f(*r.value)
}
