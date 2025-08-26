package types

import (
	"errors"
	"testing"
)

func TestResult_Ok(t *testing.T) {
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
			result := Ok(tt.value)

			if !result.IsOk() {
				t.Errorf("Expected IsOk() to be true")
			}

			if result.IsErr() {
				t.Errorf("Expected IsErr() to be false")
			}

			got := result.Unwrap()
			if got != tt.value {
				t.Errorf("Expected %v, got %v", tt.value, got)
			}
		})
	}
}

func TestResult_Err(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"basic error", errors.New("test error")},
		{"custom error", errors.New("custom message")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Err[int](tt.err)

			if result.IsOk() {
				t.Errorf("Expected IsOk() to be false")
			}

			if !result.IsErr() {
				t.Errorf("Expected IsErr() to be true")
			}

			if !errors.Is(result.Error(), tt.err) {
				t.Errorf("Expected error %v, got %v", tt.err, result.Error())
			}
		})
	}
}

func TestResult_UnwrapOr(t *testing.T) {
	tests := []struct {
		name         string
		result       Result[int]
		defaultValue int
		expected     int
	}{
		{
			name:         "ok result returns value",
			result:       Ok(42),
			defaultValue: 0,
			expected:     42,
		},
		{
			name:         "err result returns default",
			result:       Err[int](errors.New("error")),
			defaultValue: 100,
			expected:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.UnwrapOr(tt.defaultValue)
			if got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestResult_Map(t *testing.T) {
	doubleFunc := func(x int) int { return x * 2 }

	tests := []struct {
		name     string
		result   Result[int]
		expected Result[int]
	}{
		{
			name:     "ok result maps value",
			result:   Ok(21),
			expected: Ok(42),
		},
		{
			name:     "err result remains err",
			result:   Err[int](errors.New("error")),
			expected: Err[int](errors.New("error")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Map(tt.result, doubleFunc)

			if got.IsOk() != tt.expected.IsOk() {
				t.Errorf("Expected IsOk() to be %v", tt.expected.IsOk())
			}

			if got.IsOk() {
				if got.Unwrap() != tt.expected.Unwrap() {
					t.Errorf("Expected %v, got %v", tt.expected.Unwrap(), got.Unwrap())
				}
			} else {
				if got.Error().Error() != tt.expected.Error().Error() {
					t.Errorf("Expected error %v, got %v", tt.expected.Error(), got.Error())
				}
			}
		})
	}
}

func TestResult_AndThen(t *testing.T) {
	divideByTwo := func(x int) Result[int] {
		if x%2 != 0 {
			return Err[int](errors.New("odd number"))
		}

		return Ok(x / 2)
	}

	tests := []struct {
		name     string
		result   Result[int]
		expected Result[int]
	}{
		{
			name:     "ok result with valid operation",
			result:   Ok(42),
			expected: Ok(21),
		},
		{
			name:     "ok result with invalid operation",
			result:   Ok(21),
			expected: Err[int](errors.New("odd number")),
		},
		{
			name:     "err result remains err",
			result:   Err[int](errors.New("original error")),
			expected: Err[int](errors.New("original error")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AndThen(tt.result, divideByTwo)

			if got.IsOk() != tt.expected.IsOk() {
				t.Errorf("Expected IsOk() to be %v", tt.expected.IsOk())
			}

			if got.IsOk() {
				if got.Unwrap() != tt.expected.Unwrap() {
					t.Errorf("Expected %v, got %v", tt.expected.Unwrap(), got.Unwrap())
				}
			}
		})
	}
}

func TestResult_UnwrapPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when unwrapping error result")
		}
	}()

	result := Err[int](errors.New("test error"))
	result.Unwrap() // Should panic
}
