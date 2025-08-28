package mathutil_test

import (
	"testing"

	"github.com/paveg/similarity-go/internal/testhelpers"
	"github.com/paveg/similarity-go/pkg/mathutil"
)

func TestMin(t *testing.T) {
	tests := []testhelpers.MathTestCase{
		{Name: "a less than b", A: 1, B: 2, Expected: 1},
		{Name: "a greater than b", A: 5, B: 3, Expected: 3},
		{Name: "a equal to b", A: 4, B: 4, Expected: 4},
		{Name: "negative numbers", A: -3, B: -1, Expected: -3},
		{Name: "mixed signs", A: -2, B: 3, Expected: -2},
	}

	testhelpers.ExecuteMathTest(t, tests, mathutil.Min, "Min")
}

func TestMax(t *testing.T) {
	tests := []testhelpers.MathTestCase{
		{Name: "a less than b", A: 1, B: 2, Expected: 2},
		{Name: "a greater than b", A: 5, B: 3, Expected: 5},
		{Name: "a equal to b", A: 4, B: 4, Expected: 4},
		{Name: "negative numbers", A: -3, B: -1, Expected: -1},
		{Name: "mixed signs", A: -2, B: 3, Expected: 3},
	}

	testhelpers.ExecuteMathTest(t, tests, mathutil.Max, "Max")
}

func TestAbs(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"positive number", 5, 5},
		{"negative number", -5, 5},
		{"zero", 0, 0},
		{"large negative", -1000, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mathutil.Abs(tt.input)
			if result != tt.expected {
				t.Errorf("Abs(%d) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMinFloat(t *testing.T) {
	result := mathutil.Min(3.14, 2.71)
	expected := 2.71
	if result != expected {
		t.Errorf("Min(3.14, 2.71) = %f, expected %f", result, expected)
	}
}

func TestMaxFloat(t *testing.T) {
	result := mathutil.Max(3.14, 2.71)
	expected := 3.14
	if result != expected {
		t.Errorf("Max(3.14, 2.71) = %f, expected %f", result, expected)
	}
}

func TestAbsFloat(t *testing.T) {
	result := mathutil.Abs(-3.14)
	expected := 3.14
	if result != expected {
		t.Errorf("Abs(-3.14) = %f, expected %f", result, expected)
	}
}

// Test deprecated functions for backward compatibility.
func TestDeprecatedFunctions(t *testing.T) {
	// Test MinInt
	result := mathutil.MinInt(5, 3)
	if result != 3 {
		t.Errorf("MinInt(5, 3) = %d, expected 3", result)
	}

	// Test MaxInt
	result = mathutil.MaxInt(5, 3)
	if result != 5 {
		t.Errorf("MaxInt(5, 3) = %d, expected 5", result)
	}

	// Test AbsInt
	result = mathutil.AbsInt(-5)
	if result != 5 {
		t.Errorf("AbsInt(-5) = %d, expected 5", result)
	}
}
