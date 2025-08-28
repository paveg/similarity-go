package testhelpers

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	astpkg "github.com/paveg/similarity-go/internal/ast"
	"github.com/spf13/cobra"
)

// TestConfig contains configuration for test helpers.
type TestConfig struct {
	DefaultTolerance float64
	DefaultFuncName  string
}

// DefaultTestConfig returns the default test configuration.
func DefaultTestConfig() *TestConfig {
	const defaultTolerance = 0.1
	return &TestConfig{
		DefaultTolerance: defaultTolerance,
		DefaultFuncName:  "testFunc",
	}
}

// CreateTempGoFile creates a temporary Go file with the given content for testing.
func CreateTempGoFile(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.go")

	err := os.WriteFile(tempFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return tempFile
}

// ParseGoSource parses Go source code and returns the AST.
func ParseGoSource(t *testing.T, source string) (*token.FileSet, *ast.File) {
	t.Helper()

	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "test.go", source, 0)
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}

	return fileSet, file
}

// ExtractFunctionFromSource parses source and extracts the first function with the given name.
func ExtractFunctionFromSource(t *testing.T, source, funcName string) *ast.FuncDecl {
	t.Helper()

	_, file := ParseGoSource(t, source)

	var funcDecl *ast.FuncDecl

	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Name.Name == funcName {
			funcDecl = fd

			return false
		}

		return true
	})

	if funcDecl == nil {
		t.Fatalf("Function %s not found in source", funcName)
	}

	return funcDecl
}

// CreateFunctionFromSource creates a Function instance from Go source code.
// Returns nil if the function is not found instead of failing the test.
func CreateFunctionFromSource(t *testing.T, source, funcName string) *astpkg.Function {
	t.Helper()

	fileSet, file := ParseGoSource(t, source)

	var funcDecl *ast.FuncDecl

	ast.Inspect(file, func(n ast.Node) bool {
		if fd, ok := n.(*ast.FuncDecl); ok && fd.Name.Name == funcName {
			funcDecl = fd
			return false
		}
		return true
	})

	if funcDecl == nil {
		return nil // Return nil instead of failing the test
	}

	return &astpkg.Function{
		Name:      funcName,
		File:      "test.go",
		AST:       funcDecl,
		StartLine: fileSet.Position(funcDecl.Pos()).Line,
		EndLine:   fileSet.Position(funcDecl.End()).Line,
		LineCount: fileSet.Position(funcDecl.End()).Line - fileSet.Position(funcDecl.Pos()).Line + 1,
	}
}

// AssertNoError is a helper to assert that an error is nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

// AssertError is a helper to assert that an error is not nil.
func AssertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// AssertEqual is a generic helper to assert equality.
func AssertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()

	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// AssertNotEmpty is a helper to assert that a string is not empty.
func AssertNotEmpty(t *testing.T, value string) {
	t.Helper()

	if value == "" {
		t.Error("Expected non-empty string")
	}
}

// AssertContains is a helper to assert that a string contains a substring.
func AssertContains(t *testing.T, haystack, needle string) {
	t.Helper()

	if !contains(haystack, needle) {
		t.Errorf("Expected %q to contain %q", haystack, needle)
	}
}

// contains checks if a string contains a substring (helper for AssertContains).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr || containsAt(s, substr, 1)))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}

	if start+len(substr) > len(s) {
		return containsAt(s, substr, start+1)
	}

	if s[start:start+len(substr)] == substr {
		return true
	}

	return containsAt(s, substr, start+1)
}

// AbsFloat returns the absolute value of a float64.
func AbsFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Common Test Patterns

// TableTest represents a generic test case structure.
type TableTest[T any] struct {
	Name        string
	Input       T
	ExpectError bool
	Expected    interface{}
}

// ExecuteTableTest runs a table-driven test with common assertions.
func ExecuteTableTest[T any](t *testing.T, tests []TableTest[T], testFunc func(T) (interface{}, error)) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result, err := testFunc(tt.Input)

			if tt.ExpectError {
				AssertError(t, err)
			} else {
				AssertNoError(t, err)
				if tt.Expected != nil {
					AssertEqual(t, tt.Expected, result)
				}
			}
		})
	}
}

// CommandTestCase represents a CLI command test case.
type CommandTestCase struct {
	Name        string
	Args        []string
	ExpectError bool
	ExpectUsage bool
	Contains    []string // Expected strings in output
}

// ExecuteCommandTest runs a CLI command test with common setup and assertions.
func ExecuteCommandTest(t *testing.T, tests []CommandTestCase, createCmd func() *cobra.Command) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var buf bytes.Buffer

			cmd := createCmd()
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.Args)

			err := cmd.Execute()

			if tt.ExpectError {
				AssertError(t, err)
			} else {
				AssertNoError(t, err)
			}

			output := buf.String()

			if tt.ExpectUsage && !strings.Contains(output, "Usage:") {
				t.Error("Expected usage information in output")
			}

			for _, expected := range tt.Contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got: %s", expected, output)
				}
			}
		})
	}
}

// ConfigTestCase represents a configuration validation test case.
type ConfigTestCase struct {
	Name          string
	Threshold     float64
	Format        string
	Workers       int
	MinLines      int
	ExpectError   bool
	ExpectedError string
}

// ExecuteConfigTest runs configuration validation tests.
func ExecuteConfigTest(t *testing.T, tests []ConfigTestCase, validateFunc func(float64, string, int, int) error) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			err := validateFunc(tt.Threshold, tt.Format, tt.Workers, tt.MinLines)

			if tt.ExpectError {
				AssertError(t, err)
				if tt.ExpectedError != "" && err != nil && err.Error() != tt.ExpectedError {
					t.Errorf("Expected error %q, got %q", tt.ExpectedError, err.Error())
				}
			} else {
				AssertNoError(t, err)
			}
		})
	}
}

// SimilarityTestCase represents a similarity detection test case.
type SimilarityTestCase struct {
	Name      string
	Source1   string
	Source2   string
	Func1Name string
	Func2Name string
	Expected  float64
	Threshold float64
	Tolerance float64 // Default 0.1 if not specified
}

// ExecuteSimilarityTest runs similarity detection tests with function creation.
// extractFunctionName extracts the first non-main function name from Go source code.
func extractFunctionName(source string) string {
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

func ExecuteSimilarityTest(t *testing.T, tests []SimilarityTestCase, detector interface {
	CalculateSimilarity(func1, func2 interface{}) float64
}) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			executeSimilarityTestCase(t, tt, detector)
		})
	}
}

// executeSimilarityTestCase executes a single similarity test case.
func executeSimilarityTestCase(t *testing.T, tt SimilarityTestCase, detector interface {
	CalculateSimilarity(func1, func2 interface{}) float64
}) {
	t.Helper()

	func1Name := getFunctionName(tt.Func1Name, tt.Source1)
	func2Name := getFunctionName(tt.Func2Name, tt.Source2)
	if func2Name == "" {
		func2Name = func1Name
	}

	func1 := CreateFunctionFromSource(t, tt.Source1, func1Name)
	func2 := CreateFunctionFromSource(t, tt.Source2, func2Name)

	if func1 == nil || func2 == nil {
		t.Fatal("Failed to create test functions")
	}

	similarity := detector.CalculateSimilarity(func1, func2)
	tolerance := getToleranceOrDefault(tt.Tolerance)

	if AbsFloat(similarity-tt.Expected) > tolerance {
		t.Errorf("Expected similarity %.2f, got %.2f", tt.Expected, similarity)
	}
}

// getFunctionName returns the function name or extracts it from source.
func getFunctionName(providedName, source string) string {
	cfg := DefaultTestConfig()
	return getFunctionNameWithConfig(providedName, source, cfg)
}

// getFunctionNameWithConfig returns the function name using provided configuration.
func getFunctionNameWithConfig(providedName, source string, cfg *TestConfig) string {
	if providedName != "" {
		return providedName
	}

	extractedName := extractFunctionName(source)
	if extractedName != "" {
		return extractedName
	}

	return cfg.DefaultFuncName
}

// getToleranceOrDefault returns the tolerance value or default if zero.
func getToleranceOrDefault(tolerance float64) float64 {
	cfg := DefaultTestConfig()
	return getToleranceOrDefaultWithConfig(tolerance, cfg)
}

// getToleranceOrDefaultWithConfig returns the tolerance value using provided configuration.
func getToleranceOrDefaultWithConfig(tolerance float64, cfg *TestConfig) float64 {
	if tolerance == 0 {
		return cfg.DefaultTolerance
	}
	return tolerance
}

// StringSliceTestCase represents a test case for slice operations.
type StringSliceTestCase struct {
	Name     string
	Input1   []string
	Input2   []string
	Expected bool
}

// ExecuteSliceTest runs slice comparison tests.
func ExecuteSliceTest(t *testing.T, tests []StringSliceTestCase, compareFunc func([]string, []string) bool) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := compareFunc(tt.Input1, tt.Input2)
			AssertEqual(t, tt.Expected, result)
		})
	}
}

// MathTestCase represents a test case for binary integer math functions.
type MathTestCase struct {
	Name     string
	A, B     int
	Expected int
}

// ExecuteMathTest runs binary integer math function tests.
func ExecuteMathTest(t *testing.T, tests []MathTestCase, mathFunc func(int, int) int, funcName string) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := mathFunc(tt.A, tt.B)
			if result != tt.Expected {
				t.Errorf("%s(%d, %d) = %d, expected %d", funcName, tt.A, tt.B, result, tt.Expected)
			}
		})
	}
}

// FunctionTestCase represents a generic function testing scenario.
type FunctionTestCase[TInput, TOutput any] struct {
	Name        string
	Input       TInput
	Expected    TOutput
	ExpectError bool
	Setup       func() error // Optional setup function
	Teardown    func()       // Optional teardown function
}

// ExecuteFunctionTest runs table-driven tests for functions with typed input/output.
func ExecuteFunctionTest[TInput any, TOutput comparable](
	t *testing.T,
	tests []FunctionTestCase[TInput, TOutput],
	testFunc func(TInput) (TOutput, error),
) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if tt.Setup != nil {
				if err := tt.Setup(); err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			if tt.Teardown != nil {
				defer tt.Teardown()
			}

			result, err := testFunc(tt.Input)

			if tt.ExpectError {
				AssertError(t, err)
			} else {
				AssertNoError(t, err)
				AssertEqual(t, tt.Expected, result)
			}
		})
	}
}

// ValidationTestCase represents validation testing scenarios.
type ValidationTestCase[T any] struct {
	Name     string
	Input    T
	IsValid  bool
	ErrorMsg string
}

// ExecuteValidationTest runs validation tests.
func ExecuteValidationTest[T any](
	t *testing.T,
	tests []ValidationTestCase[T],
	validateFunc func(T) error,
) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			err := validateFunc(tt.Input)

			if tt.IsValid {
				AssertNoError(t, err)
			} else {
				AssertError(t, err)
				if tt.ErrorMsg != "" {
					AssertContains(t, err.Error(), tt.ErrorMsg)
				}
			}
		})
	}
}

// TransformTestCase represents transformation testing scenarios.
type TransformTestCase[TInput, TOutput any] struct {
	Name      string
	Input     TInput
	Expected  TOutput
	Transform func(TInput) TOutput
}

// ExecuteTransformTest runs transformation tests.
func ExecuteTransformTest[TInput any, TOutput comparable](
	t *testing.T,
	tests []TransformTestCase[TInput, TOutput],
) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result := tt.Transform(tt.Input)
			AssertEqual(t, tt.Expected, result)
		})
	}
}

// ParseTestCase represents parsing test scenarios.
type ParseTestCase[T any] struct {
	Name        string
	Source      string
	Expected    T
	ExpectError bool
}

// ExecuteParseTest runs parsing tests.
func ExecuteParseTest[T any](
	t *testing.T,
	tests []ParseTestCase[T],
	parseFunc func(string) (T, error),
) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			result, err := parseFunc(tt.Source)

			if tt.ExpectError {
				AssertError(t, err)
			} else {
				AssertNoError(t, err)
				// Use reflect.DeepEqual for non-comparable types like slices
				if !reflect.DeepEqual(tt.Expected, result) {
					t.Errorf("Expected %v, got %v", tt.Expected, result)
				}
			}
		})
	}
}
