package types_test

import (
	"testing"

	"github.com/paveg/similarity-go/pkg/types"
)

func TestOptional_Some(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{"positive integer", 42},
		{"zero", 0},
		{"negative integer", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := types.Some(tt.value)

			if !opt.IsSome() {
				t.Errorf("Expected IsSome() to be true")
			}

			if opt.IsNone() {
				t.Errorf("Expected IsNone() to be false")
			}

			got := opt.Unwrap()
			if got != tt.value {
				t.Errorf("Expected %v, got %v", tt.value, got)
			}
		})
	}
}

func TestOptional_None(t *testing.T) {
	opt := types.None[int]()

	if opt.IsSome() {
		t.Errorf("Expected IsSome() to be false")
	}

	if !opt.IsNone() {
		t.Errorf("Expected IsNone() to be true")
	}
}

func TestOptional_UnwrapOr(t *testing.T) {
	tests := []struct {
		name         string
		optional     types.Optional[int]
		defaultValue int
		expected     int
	}{
		{
			name:         "some value returns value",
			optional:     types.Some(42),
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "none returns default",
			optional:     types.None[int](),
			defaultValue: 100,
			expected:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.optional.UnwrapOr(tt.defaultValue)
			if got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestOptional_Map(t *testing.T) {
	doubleFunc := func(x int) int { return x * 2 }

	tests := []struct {
		name     string
		optional types.Optional[int]
		expected types.Optional[int]
	}{
		{
			name:     "some value maps to some",
			optional: types.Some(21),
			expected: types.Some(42),
		},
		{
			name:     "none remains none",
			optional: types.None[int](),
			expected: types.None[int](),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := types.MapOptional(tt.optional, doubleFunc)

			if got.IsSome() != tt.expected.IsSome() {
				t.Errorf("Expected IsSome() to be %v", tt.expected.IsSome())
			}

			if got.IsSome() {
				if got.Unwrap() != tt.expected.Unwrap() {
					t.Errorf("Expected %v, got %v", tt.expected.Unwrap(), got.Unwrap())
				}
			}
		})
	}
}

func TestOptional_Filter(t *testing.T) {
	evenFilter := func(x int) bool { return x%2 == 0 }

	tests := []struct {
		name     string
		optional types.Optional[int]
		expected types.Optional[int]
	}{
		{
			name:     "some value passes filter",
			optional: types.Some(42),
			expected: types.Some(42),
		},
		{
			name:     "some value fails filter",
			optional: types.Some(21),
			expected: types.None[int](),
		},
		{
			name:     "none remains none",
			optional: types.None[int](),
			expected: types.None[int](),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := types.Filter(tt.optional, evenFilter)

			if got.IsSome() != tt.expected.IsSome() {
				t.Errorf("Expected IsSome() to be %v", tt.expected.IsSome())
			}

			if got.IsSome() {
				if got.Unwrap() != tt.expected.Unwrap() {
					t.Errorf("Expected %v, got %v", tt.expected.Unwrap(), got.Unwrap())
				}
			}
		})
	}
}

func TestOptional_AndThen(t *testing.T) {
	safeDiv := func(x int) types.Optional[int] {
		if x == 0 {
			return types.None[int]()
		}

		return types.Some(100 / x)
	}

	tests := []struct {
		name     string
		optional types.Optional[int]
		expected types.Optional[int]
	}{
		{
			name:     "some value with valid operation",
			optional: types.Some(10),
			expected: types.Some(10),
		},
		{
			name:     "some value with invalid operation",
			optional: types.Some(0),
			expected: types.None[int](),
		},
		{
			name:     "none remains none",
			optional: types.None[int](),
			expected: types.None[int](),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := types.AndThenOptional(tt.optional, safeDiv)

			if got.IsSome() != tt.expected.IsSome() {
				t.Errorf("Expected IsSome() to be %v", tt.expected.IsSome())
			}

			if got.IsSome() {
				if got.Unwrap() != tt.expected.Unwrap() {
					t.Errorf("Expected %v, got %v", tt.expected.Unwrap(), got.Unwrap())
				}
			}
		})
	}
}

func TestOptional_UnwrapPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when unwrapping None")
		}
	}()

	opt := types.None[int]()
	opt.Unwrap() // Should panic
}
