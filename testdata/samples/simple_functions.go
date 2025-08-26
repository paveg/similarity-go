package samples

import "fmt"

// Simple function for testing
func Add(a, b int) int {
	return a + b
}

// Similar function with different variable names
func Sum(x, y int) int {
	return x + y
}

// Different function structure
func Multiply(a, b int) int {
	result := a * b
	return result
}

// Function with more complex body
func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// Method on struct
type Calculator struct{}

func (c *Calculator) Calculate(a, b int) int {
	return a + b
}

// Generic function (Go 1.18+)
func Generic[T any](value T) T {
	return value
}