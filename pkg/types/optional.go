package types

// Optional represents a value that may or may not be present.
// This follows the Rust-like Option pattern for null safety.
type Optional[T any] struct {
	value *T
}

// Some creates an Optional containing the given value.
func Some[T any](value T) Optional[T] {
	return Optional[T]{value: &value}
}

// None creates an empty Optional.
func None[T any]() Optional[T] {
	return Optional[T]{value: nil}
}

// IsSome returns true if the Optional contains a value.
func (o Optional[T]) IsSome() bool {
	return o.value != nil
}

// IsNone returns true if the Optional is empty.
func (o Optional[T]) IsNone() bool {
	return o.value == nil
}

// Unwrap returns the contained value if Some, panics if None.
// Use this only when you are certain the Optional is Some.
func (o Optional[T]) Unwrap() T {
	if o.value == nil {
		panic("called Unwrap on None value")
	}
	return *o.value
}

// UnwrapOr returns the contained value if Some, otherwise returns the default value.
func (o Optional[T]) UnwrapOr(defaultValue T) T {
	if o.value == nil {
		return defaultValue
	}
	return *o.value
}

// MapOptional applies a function to the contained value if Some, otherwise returns None.
func MapOptional[T, U any](o Optional[T], mapper func(T) U) Optional[U] {
	if o.value == nil {
		return None[U]()
	}
	return Some(mapper(*o.value))
}

// Filter returns the Optional if it contains a value and the predicate returns true,
// otherwise returns None.
func Filter[T any](o Optional[T], predicate func(T) bool) Optional[T] {
	if o.value == nil || !predicate(*o.value) {
		return None[T]()
	}
	return o
}

// AndThenOptional applies a function that returns an Optional to the contained value if Some.
// This is useful for chaining operations that may not return a value.
func AndThenOptional[T, U any](o Optional[T], f func(T) Optional[U]) Optional[U] {
	if o.value == nil {
		return None[U]()
	}
	return f(*o.value)
}
