package similarity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	astpkg "github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/testhelpers"
)

// BenchmarkCase represents a test case for weight optimization.
type BenchmarkCase struct {
	Name               string
	Function1Source    string
	Function2Source    string
	ExpectedSimilarity float64 // Ground truth similarity (0.0-1.0)
	Category           string  // e.g., "identical", "high_similar", "medium_similar", "low_similar", "different"
}

// GetBenchmarkDataset returns a comprehensive dataset for weight optimization.
//
//nolint:funlen,mnd // dataset definition requires explicit listing of scenarios and literal expectations
func GetBenchmarkDataset() []BenchmarkCase {
	return []BenchmarkCase{
		// Identical functions
		{
			Name: "identical_functions",
			Function1Source: `package main
func add(a, b int) int {
	return a + b
}`,
			Function2Source: `package main
func add(a, b int) int {
	return a + b
}`,
			ExpectedSimilarity: 1.0,
			Category:           "identical",
		},

		// High similarity - same logic, different variable names
		{
			Name: "same_logic_different_vars",
			Function1Source: `package main
func calculateSum(x, y int) int {
	result := x + y
	return result
}`,
			Function2Source: `package main
func addNumbers(a, b int) int {
	sum := a + b
	return sum
}`,
			ExpectedSimilarity: 0.85,
			Category:           "high_similar",
		},

		// High similarity - same structure, minor differences
		{
			Name: "same_structure_minor_diff",
			Function1Source: `package main
func processData(data []int) int {
	total := 0
	for _, value := range data {
		total += value
	}
	return total
}`,
			Function2Source: `package main
func sumSlice(numbers []int) int {
	sum := 0
	for _, num := range numbers {
		sum += num
	}
	return sum
}`,
			ExpectedSimilarity: 0.9,
			Category:           "high_similar",
		},

		// Medium similarity - similar pattern, different operations
		{
			Name: "similar_pattern_different_ops",
			Function1Source: `package main
func multiply(a, b int) int {
	return a * b
}`,
			Function2Source: `package main
func divide(a, b int) int {
	return a / b
}`,
			ExpectedSimilarity: 0.7,
			Category:           "medium_similar",
		},

		// Medium similarity - similar control flow, different logic
		{
			Name: "similar_control_flow",
			Function1Source: `package main
func findMax(data []int) int {
	max := data[0]
	for i := 1; i < len(data); i++ {
		if data[i] > max {
			max = data[i]
		}
	}
	return max
}`,
			Function2Source: `package main
func findMin(numbers []int) int {
	min := numbers[0]
	for i := 1; i < len(numbers); i++ {
		if numbers[i] < min {
			min = numbers[i]
		}
	}
	return min
}`,
			ExpectedSimilarity: 0.75,
			Category:           "medium_similar",
		},

		// Low similarity - same signature, different implementation
		{
			Name: "same_signature_different_impl",
			Function1Source: `package main
func process(data []int) []int {
	result := make([]int, len(data))
	for i, v := range data {
		result[i] = v * 2
	}
	return result
}`,
			Function2Source: `package main
func process(data []int) []int {
	var result []int
	for _, v := range data {
		if v > 0 {
			result = append(result, v)
		}
	}
	return result
}`,
			ExpectedSimilarity: 0.4,
			Category:           "low_similar",
		},

		// Very different functions
		{
			Name: "completely_different",
			Function1Source: `package main
func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}`,
			Function2Source: `package main
func printHello(name string) {
	fmt.Printf("Hello, %s!\n", name)
}`,
			ExpectedSimilarity: 0.1,
			Category:           "different",
		},

		// Go-specific patterns: Interface methods
		{
			Name: "interface_implementations",
			Function1Source: `package main
type Writer interface {
	Write([]byte) (int, error)
}

func (w *MyWriter) Write(data []byte) (int, error) {
	return len(data), nil
}`,
			Function2Source: `package main
type Writer interface {
	Write([]byte) (int, error)
}

func (w *AnotherWriter) Write(bytes []byte) (int, error) {
	return len(bytes), nil
}`,
			ExpectedSimilarity: 0.9,
			Category:           "high_similar",
		},

		// Go-specific patterns: Error handling
		{
			Name: "error_handling_patterns",
			Function1Source: `package main
func readFile(filename string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return data, nil
}`,
			Function2Source: `package main
func loadData(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}
	return content, nil
}`,
			ExpectedSimilarity: 0.85,
			Category:           "high_similar",
		},

		// Go-specific patterns: Goroutines and channels
		{
			Name: "goroutine_patterns",
			Function1Source: `package main
func processAsync(data []int, result chan<- int) {
	go func() {
		defer close(result)
		sum := 0
		for _, v := range data {
			sum += v
		}
		result <- sum
	}()
}`,
			Function2Source: `package main
func calculateAsync(numbers []int, output chan<- int) {
	go func() {
		defer close(output)
		total := 0
		for _, num := range numbers {
			total += num
		}
		output <- total
	}()
}`,
			ExpectedSimilarity: 0.9,
			Category:           "high_similar",
		},
	}
}

// CreateFunctionPair creates AST functions from benchmark case.
func (bc BenchmarkCase) CreateFunctionPair(t *testing.T) (*astpkg.Function, *astpkg.Function) {
	// Extract actual function names from source code instead of using hardcoded names
	func1Name := extractFunctionNameFromSource(bc.Function1Source)
	func2Name := extractFunctionNameFromSource(bc.Function2Source)

	if func1Name == "" {
		t.Errorf("Could not extract function name from Function1Source in case %s", bc.Name)
		return nil, nil
	}
	if func2Name == "" {
		t.Errorf("Could not extract function name from Function2Source in case %s", bc.Name)
		return nil, nil
	}

	func1 := testhelpers.CreateFunctionFromSource(t, bc.Function1Source, func1Name)
	func2 := testhelpers.CreateFunctionFromSource(t, bc.Function2Source, func2Name)
	return func1, func2
}

// extractFunctionNameFromSource extracts the first non-main function name from Go source code.
func extractFunctionNameFromSource(source string) string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		return ""
	}

	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name != nil && fn.Name.Name != "main" {
			return fn.Name.Name
		}
	}
	return ""
}
