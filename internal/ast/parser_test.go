package ast

import (
	"os"
	"testing"
)

func TestParser_ParseFile_ValidGoFiles(t *testing.T) {
	tests := []struct {
		name          string
		source        string
		expectedFuncs int
		expectedNames []string
		expectError   bool
	}{
		{
			name: "single function",
			source: `package main

func add(a, b int) int {
	return a + b
}`,
			expectedFuncs: 1,
			expectedNames: []string{"add"},
			expectError:   false,
		},
		{
			name: "multiple functions",
			source: `package main

import "fmt"

func add(a, b int) int {
	return a + b
}

func subtract(a, b int) int {
	return a - b
}

func greet(name string) {
	fmt.Printf("Hello, %s!\n", name)
}`,
			expectedFuncs: 3,
			expectedNames: []string{"add", "subtract", "greet"},
			expectError:   false,
		},
		{
			name: "function with complex signature",
			source: `package main

func processData(data []byte, callback func(string) error) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	str := string(data)
	if err := callback(str); err != nil {
		return nil, err
	}
	result["processed"] = true
	return result, nil
}`,
			expectedFuncs: 1,
			expectedNames: []string{"processData"},
			expectError:   false,
		},
		{
			name: "generic function",
			source: `package main

func Map[T, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}`,
			expectedFuncs: 1,
			expectedNames: []string{"Map"},
			expectError:   false,
		},
		{
			name: "methods on struct",
			source: `package main

type Calculator struct {
	value int
}

func (c *Calculator) Add(n int) int {
	c.value += n
	return c.value
}

func (c Calculator) GetValue() int {
	return c.value
}`,
			expectedFuncs: 2,
			expectedNames: []string{"Add", "GetValue"},
			expectError:   false,
		},
		{
			name: "interface with methods (should be ignored)",
			source: `package main

type Writer interface {
	Write([]byte) (int, error)
	Close() error
}

func realFunction() {
	// This should be detected
}`,
			expectedFuncs: 1, // Only realFunction, interface methods ignored
			expectedNames: []string{"realFunction"},
			expectError:   false,
		},
		{
			name: "empty package",
			source: `package main

import "fmt"

var globalVar = "test"
`,
			expectedFuncs: 0,
			expectedNames: []string{},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile := createTempFile(t, tt.source)
			defer os.Remove(tmpFile)

			parser := NewParser()
			result := parser.ParseFile(tmpFile)

			if tt.expectError {
				if result.IsOk() {
					t.Errorf("Expected error, but parsing succeeded")
				}

				return
			}

			if result.IsErr() {
				t.Fatalf("Unexpected error: %v", result.Error())
			}

			parseResult := result.Unwrap()
			if len(parseResult.Functions) != tt.expectedFuncs {
				t.Errorf("Expected %d functions, got %d", tt.expectedFuncs, len(parseResult.Functions))
			}

			// Check function names
			actualNames := make([]string, len(parseResult.Functions))
			for i, fn := range parseResult.Functions {
				actualNames[i] = fn.Name
			}

			if !stringSliceEqual(actualNames, tt.expectedNames) {
				t.Errorf("Expected function names %v, got %v", tt.expectedNames, actualNames)
			}

			// Verify each function has proper metadata
			for _, fn := range parseResult.Functions {
				if fn.Name == "" {
					t.Error("Function name should not be empty")
				}

				if fn.File != tmpFile {
					t.Errorf("Expected file %s, got %s", tmpFile, fn.File)
				}

				if fn.StartLine <= 0 {
					t.Error("StartLine should be positive")
				}

				if fn.EndLine <= 0 {
					t.Error("EndLine should be positive")
				}

				if fn.StartLine > fn.EndLine {
					t.Errorf("StartLine (%d) should not be greater than EndLine (%d)", fn.StartLine, fn.EndLine)
				}

				if fn.AST == nil {
					t.Error("AST should not be nil")
				}

				if fn.LineCount <= 0 {
					t.Error("LineCount should be positive")
				}
			}
		})
	}
}

func TestParser_ParseFile_InvalidFiles(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "syntax error",
			source: `package main\n\nfunc invalid syntax {`,
		},
		{
			name:   "missing package declaration",
			source: `func test() {}`,
		},
		{
			name:   "incomplete function",
			source: `package main\n\nfunc incomplete(`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.source)
			defer os.Remove(tmpFile)

			parser := NewParser()
			result := parser.ParseFile(tmpFile)

			if result.IsOk() {
				t.Error("Expected parsing to fail for invalid Go code")
			}
		})
	}
}

func TestParser_ParseFile_FileSystemErrors(t *testing.T) {
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "non-existent file",
			filename: "/path/that/does/not/exist.go",
		},
		{
			name:     "directory instead of file",
			filename: "/tmp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			result := parser.ParseFile(tt.filename)

			if result.IsOk() {
				t.Error("Expected error when parsing non-existent or invalid file")
			}
		})
	}
}

func TestParser_ParseFiles_MultiplFiles(t *testing.T) {
	// Create multiple temporary files
	source1 := `package main
func func1() { }`
	source2 := `package main  
func func2() { }
func func3() { }`
	source3 := `package main
// No functions here
var x = 1`

	file1 := createTempFile(t, source1)
	file2 := createTempFile(t, source2)
	file3 := createTempFile(t, source3)

	defer func() {
		os.Remove(file1)
		os.Remove(file2)
		os.Remove(file3)
	}()

	parser := NewParser()
	result := parser.ParseFiles([]string{file1, file2, file3})

	if result.IsErr() {
		t.Fatalf("Unexpected error: %v", result.Error())
	}

	parseResult := result.Unwrap()

	// Should have 3 functions total (1 + 2 + 0)
	if len(parseResult.Functions) != 3 {
		t.Errorf("Expected 3 functions total, got %d", len(parseResult.Functions))
	}

	// Check that functions are from correct files
	expectedFiles := map[string]int{
		file1: 1,
		file2: 2,
		file3: 0,
	}

	actualFiles := make(map[string]int)
	for _, fn := range parseResult.Functions {
		actualFiles[fn.File]++
	}

	for file, expectedCount := range expectedFiles {
		if actualFiles[file] != expectedCount {
			t.Errorf("Expected %d functions from %s, got %d", expectedCount, file, actualFiles[file])
		}
	}
}

func TestParser_ParseFiles_WithErrors(t *testing.T) {
	validSource := `package main
func validFunc() {}`
	invalidSource := `package main
func invalid syntax {`

	validFile := createTempFile(t, validSource)
	invalidFile := createTempFile(t, invalidSource)
	nonExistentFile := "/path/does/not/exist.go"

	defer func() {
		os.Remove(validFile)
		os.Remove(invalidFile)
	}()

	parser := NewParser()
	result := parser.ParseFiles([]string{validFile, invalidFile, nonExistentFile})

	if result.IsErr() {
		t.Fatalf("Expected partial success, got error: %v", result.Error())
	}

	parseResult := result.Unwrap()

	// Should have 1 valid function
	if len(parseResult.Functions) != 1 {
		t.Errorf("Expected 1 valid function, got %d", len(parseResult.Functions))
	}

	// Should have errors for the invalid files
	if len(parseResult.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(parseResult.Errors))
	}
}

func TestParser_LineCountCalculation(t *testing.T) {
	tests := []struct {
		name              string
		source            string
		expectedLineCount int
	}{
		{
			name: "single line function",
			source: `package main
func oneLine() { return }`,
			expectedLineCount: 1,
		},
		{
			name: "multi-line function",
			source: `package main
func multiLine() {
	x := 1
	y := 2
	return x + y
}`,
			expectedLineCount: 5,
		},
		{
			name: "function with comments",
			source: `package main
// This is a comment
func withComments() {
	// Internal comment
	x := 1 // Inline comment
	return x
}`,
			expectedLineCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempFile(t, tt.source)
			defer os.Remove(tmpFile)

			parser := NewParser()
			result := parser.ParseFile(tmpFile)

			if result.IsErr() {
				t.Fatalf("Unexpected error: %v", result.Error())
			}

			parseResult := result.Unwrap()
			if len(parseResult.Functions) != 1 {
				t.Fatalf("Expected 1 function, got %d", len(parseResult.Functions))
			}

			fn := parseResult.Functions[0]
			if fn.LineCount != tt.expectedLineCount {
				t.Errorf("Expected line count %d, got %d", tt.expectedLineCount, fn.LineCount)
			}
		})
	}
}

func TestParser_NewParser(t *testing.T) {
	parser := NewParser()

	if parser == nil {
		t.Error("NewParser() should not return nil")
	}

	// Parser should be ready to use immediately
	if parser.fileSet == nil {
		t.Error("Parser should have initialized fileSet")
	}
}

// Helper functions.
func createTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_*.go")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	return tmpFile.Name()
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}
