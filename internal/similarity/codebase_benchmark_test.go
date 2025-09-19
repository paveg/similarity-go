package similarity

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	astpkg "github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/config"
)

func TestCodebaseBenchmark_NewCodebaseBenchmark(t *testing.T) {
	paths := []string{"/path1", "/path2"}
	benchmark := NewCodebaseBenchmark(paths)

	if benchmark == nil {
		t.Fatal("Expected non-nil benchmark")
	}

	if len(benchmark.basePaths) != 2 {
		t.Errorf("Expected 2 base paths, got %d", len(benchmark.basePaths))
	}

	if benchmark.minLines <= 0 {
		t.Error("Expected positive minLines")
	}

	if benchmark.maxFunctions <= 0 {
		t.Error("Expected positive maxFunctions")
	}

	if len(benchmark.excludePaths) == 0 {
		t.Error("Expected default exclude paths")
	}

	if benchmark.fileSet == nil {
		t.Error("Expected non-nil fileSet")
	}
}

func TestCodebaseBenchmark_SetParameters(t *testing.T) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})

	initialExcludeCount := len(benchmark.excludePaths)

	benchmark.SetParameters(10, 500, []string{"custom_exclude/"})

	if benchmark.minLines != 10 {
		t.Errorf("Expected minLines 10, got %d", benchmark.minLines)
	}

	if benchmark.maxFunctions != 500 {
		t.Errorf("Expected maxFunctions 500, got %d", benchmark.maxFunctions)
	}

	if len(benchmark.excludePaths) != initialExcludeCount+1 {
		t.Errorf("Expected %d exclude paths, got %d",
			initialExcludeCount+1, len(benchmark.excludePaths))
	}

	// Check custom exclude path was added
	found := false
	for _, path := range benchmark.excludePaths {
		if path == "custom_exclude/" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom exclude path was not added")
	}
}

func TestCodebaseBenchmark_estimateComplexity(t *testing.T) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})

	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{
			name: "simple_function",
			code: `func simple() {
				return
			}`,
			expected: 1, // Base complexity
		},
		{
			name: "function_with_if",
			code: `func withIf() {
				if true {
					return
				}
			}`,
			expected: 2, // Base + 1 if
		},
		{
			name: "function_with_multiple_conditions",
			code: `func complex() {
				if condition1 {
					return
				}
				for i := 0; i < 10; i++ {
					if condition2 {
						continue
					}
				}
				switch value {
				case 1:
					break
				case 2:
					break
				}
			}`,
			expected: 7, // Base + if + for + if + switch + 2 cases
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the test code
			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, "", "package test\n"+tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse test code: %v", err)
			}

			// Find the function declaration
			var funcDecl *ast.FuncDecl
			ast.Inspect(file, func(n ast.Node) bool {
				if fd, ok := n.(*ast.FuncDecl); ok {
					funcDecl = fd
					return false
				}
				return true
			})

			if funcDecl == nil {
				t.Fatal("Function declaration not found")
			}

			complexity := benchmark.estimateComplexity(funcDecl)
			if complexity != tt.expected {
				t.Errorf("Expected complexity %d, got %d", tt.expected, complexity)
			}
		})
	}
}

func TestCodebaseBenchmark_extractSignature(t *testing.T) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})

	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "simple_function",
			code:     `func simple() {}`,
			expected: "simple",
		},
		{
			name:     "function_with_params",
			code:     `func withParams(a int, b string) {}`,
			expected: "withParams_*ast.Ident_*ast.Ident", // Simplified type representation
		},
		{
			name:     "function_with_return",
			code:     `func withReturn() int { return 0 }`,
			expected: "withReturn_*ast.Ident",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the test code
			fileSet := token.NewFileSet()
			file, err := parser.ParseFile(fileSet, "", "package test\n"+tt.code, 0)
			if err != nil {
				t.Fatalf("Failed to parse test code: %v", err)
			}

			// Find the function declaration
			var funcDecl *ast.FuncDecl
			ast.Inspect(file, func(n ast.Node) bool {
				if fd, ok := n.(*ast.FuncDecl); ok {
					funcDecl = fd
					return false
				}
				return true
			})

			if funcDecl == nil {
				t.Fatal("Function declaration not found")
			}

			signature := benchmark.extractSignature(funcDecl)
			if signature != tt.expected {
				t.Errorf("Expected signature %q, got %q", tt.expected, signature)
			}
		})
	}
}

func TestCodebaseBenchmark_categorizePair(t *testing.T) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})

	func1 := FunctionMetadata{Name: "func1", Package: "pkg1"}
	func2 := FunctionMetadata{Name: "func2", Package: "pkg2"}

	tests := []struct {
		similarity float64
		expected   string
	}{
		{0.97, "near-duplicate"},
		{0.90, "refactoring-candidate"},
		{0.80, "similar-logic"},
		{0.70, "related-functionality"},
	}

	for _, tt := range tests {
		category := benchmark.categorizePair(tt.similarity, func1, func2)
		if category != tt.expected {
			t.Errorf("For similarity %.2f, expected category %q, got %q",
				tt.similarity, tt.expected, category)
		}
	}
}

func TestCodebaseBenchmark_calculateDuplicationMetrics(t *testing.T) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})

	pairs := []RealWorldPair{
		{Similarity: 0.95, Category: "near-duplicate"},
		{Similarity: 0.85, Category: "refactoring-candidate"},
		{Similarity: 0.75, Category: "similar-logic"},
		{Similarity: 0.65, Category: "related-functionality"},
		{Similarity: 0.55, Category: "related-functionality"},
	}

	metrics := benchmark.calculateDuplicationMetrics(pairs)

	expectedHigh := 2        // >= 0.8
	expectedMedium := 2      // >= 0.6 and < 0.8
	expectedRefactoring := 2 // near-duplicate + refactoring-candidate

	if metrics.HighSimilarityCount != expectedHigh {
		t.Errorf("Expected %d high similarity pairs, got %d",
			expectedHigh, metrics.HighSimilarityCount)
	}

	if metrics.MediumSimilarityCount != expectedMedium {
		t.Errorf("Expected %d medium similarity pairs, got %d",
			expectedMedium, metrics.MediumSimilarityCount)
	}

	if metrics.RefactoringCandidates != expectedRefactoring {
		t.Errorf("Expected %d refactoring candidates, got %d",
			expectedRefactoring, metrics.RefactoringCandidates)
	}

	expectedDuplication := float64(expectedHigh) / float64(len(pairs)) * 100
	if metrics.EstimatedDuplication != expectedDuplication {
		t.Errorf("Expected %.1f%% estimated duplication, got %.1f%%",
			expectedDuplication, metrics.EstimatedDuplication)
	}
}

//nolint:gocognit // extensive assertions for multiple scenarios
func TestCodebaseBenchmark_extractFunctionsFromFile(t *testing.T) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})
	benchmark.SetParameters(3, 1000, nil) // Minimum 3 lines

	// Create a temporary Go file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	testCode := `package testpkg

import "fmt"

// This function should be included (>= 3 lines)
func included() {
	fmt.Println("Hello")
	return
}

// This function should be excluded (< 3 lines)
func excluded() { return }

// This function should be included
func anotherIncluded() {
	x := 1
	y := 2
	z := x + y
	fmt.Println(z)
}

// Function declaration without body should be excluded
func declaration() int
`

	err := os.WriteFile(testFile, []byte(testCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	functions, err := benchmark.extractFunctionsFromFile(testFile)
	if err != nil {
		t.Fatalf("extractFunctionsFromFile() error = %v", err)
	}

	// Should find 2 functions (included and anotherIncluded)
	if len(functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(functions))
	}

	// Check function properties
	for _, f := range functions {
		if f.Package != "testpkg" {
			t.Errorf("Expected package 'testpkg', got '%s'", f.Package)
		}

		if f.File != testFile {
			t.Errorf("Expected file path %s, got %s", testFile, f.File)
		}

		if f.LineCount < 3 {
			t.Errorf("Function %s should have >= 3 lines, got %d", f.Name, f.LineCount)
		}

		if f.Complexity < 1 {
			t.Errorf("Function %s should have complexity >= 1, got %d", f.Name, f.Complexity)
		}

		if f.StartLine <= 0 || f.EndLine <= 0 {
			t.Errorf("Function %s has invalid line numbers: start=%d, end=%d",
				f.Name, f.StartLine, f.EndLine)
		}

		if f.EndLine <= f.StartLine {
			t.Errorf("Function %s end line should be > start line: start=%d, end=%d",
				f.Name, f.StartLine, f.EndLine)
		}

		if f.AST == nil {
			t.Errorf("Function %s should have non-nil AST", f.Name)
		}
	}

	// Check specific function names
	functionNames := make(map[string]bool)
	for _, f := range functions {
		functionNames[f.Name] = true
	}

	if !functionNames["included"] {
		t.Error("Expected to find function 'included'")
	}

	if !functionNames["anotherIncluded"] {
		t.Error("Expected to find function 'anotherIncluded'")
	}

	if functionNames["excluded"] {
		t.Error("Should not find function 'excluded' (too short)")
	}

	if functionNames["declaration"] {
		t.Error("Should not find function 'declaration' (no body)")
	}
}

func TestCodebaseBenchmark_extractFunctions_ExcludePaths(t *testing.T) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})
	benchmark.SetParameters(1, 1000, []string{"excluded/"})

	// Create temporary directory structure
	tempDir := t.TempDir()

	// Create included directory and file
	includedDir := filepath.Join(tempDir, "included")
	err := os.MkdirAll(includedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create included dir: %v", err)
	}

	includedFile := filepath.Join(includedDir, "included.go")
	err = os.WriteFile(includedFile, []byte(`package included
func shouldBeIncluded() {
	return
}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create included file: %v", err)
	}

	// Create excluded directory and file
	excludedDir := filepath.Join(tempDir, "excluded")
	err = os.MkdirAll(excludedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create excluded dir: %v", err)
	}

	excludedFile := filepath.Join(excludedDir, "excluded.go")
	err = os.WriteFile(excludedFile, []byte(`package excluded
func shouldBeExcluded() {
	return
}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create excluded file: %v", err)
	}

	// Create test file
	testFile := filepath.Join(tempDir, "test_file_test.go")
	err = os.WriteFile(testFile, []byte(`package main
func testFunc() {
	return
}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	functions, err := benchmark.extractFunctions(tempDir)
	if err != nil {
		t.Fatalf("extractFunctions() error = %v", err)
	}

	// Should only find the included function (test file excluded by default)
	if len(functions) != 1 {
		t.Errorf("Expected 1 function, got %d", len(functions))
	}

	if len(functions) > 0 && functions[0].Name != "shouldBeIncluded" {
		t.Errorf("Expected function 'shouldBeIncluded', got '%s'", functions[0].Name)
	}

	// Verify excluded functions are not present
	for _, f := range functions {
		if f.Name == "shouldBeExcluded" {
			t.Error("Found excluded function 'shouldBeExcluded'")
		}
		if f.Name == "testFunc" {
			t.Error("Found test function 'testFunc' (should be excluded)")
		}
	}
}

func TestCodebaseBenchmark_countFiles(t *testing.T) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})

	// Create temporary directory structure
	tempDir := t.TempDir()

	// Create regular Go files
	files := []string{"file1.go", "file2.go", "subdir/file3.go"}
	for _, file := range files {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		err = os.WriteFile(fullPath, []byte("package main"), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Create files that should be excluded
	excludedFiles := []string{
		"file_test.go",     // Test file
		"vendor/vendor.go", // Vendor directory
		"other.txt",        // Non-Go file
	}
	for _, file := range excludedFiles {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		err = os.WriteFile(fullPath, []byte("content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create excluded file %s: %v", fullPath, err)
		}
	}

	count := benchmark.countFiles(tempDir)

	// Should count 3 regular Go files (excluding test file, vendor file, and non-Go file)
	if count != 3 {
		t.Errorf("Expected 3 files, got %d", count)
	}
}

func TestCodebaseBenchmark_calculateComponents(t *testing.T) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})

	// Create test functions
	func1 := &astpkg.Function{Name: "test1"}
	func2 := &astpkg.Function{Name: "test2"}

	// Create detector with known weights
	detector := NewDetector(0.7)
	cfg := config.Default()
	cfg.Similarity.Weights = config.SimilarityWeights{
		TreeEdit:        0.4,
		TokenSimilarity: 0.3,
		Structural:      0.2,
		Signature:       0.1,
	}
	detector.config = cfg

	components := benchmark.calculateComponents(func1, func2, detector)

	// Verify components structure
	if components.WeightedScore < 0 || components.WeightedScore > 1 {
		t.Errorf("WeightedScore should be between 0 and 1, got %f", components.WeightedScore)
	}

	if components.TreeEdit < 0 {
		t.Errorf("TreeEdit component should be non-negative, got %f", components.TreeEdit)
	}

	if components.TokenSimilarity < 0 {
		t.Errorf("TokenSimilarity component should be non-negative, got %f", components.TokenSimilarity)
	}

	if components.Structural < 0 {
		t.Errorf("Structural component should be non-negative, got %f", components.Structural)
	}

	if components.Signature < 0 {
		t.Errorf("Signature component should be non-negative, got %f", components.Signature)
	}
}

func BenchmarkCodebaseBenchmark_extractFunctionsFromFile(b *testing.B) {
	benchmark := NewCodebaseBenchmark([]string{"/test"})

	// Create a temporary file with multiple functions
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "bench_test.go")

	// Generate a file with many functions
	var content strings.Builder
	content.WriteString("package benchmark\n\n")

	for i := range 50 {
		content.WriteString(fmt.Sprintf(`
func function%d() {
	x := %d
	y := x * 2
	z := y + 1
	return z
}
`, i, i))
	}

	err := os.WriteFile(testFile, []byte(content.String()), 0644)
	if err != nil {
		b.Fatalf("Failed to create benchmark file: %v", err)
	}

	b.ResetTimer()
	for range b.N {
		_, extractErr := benchmark.extractFunctionsFromFile(testFile)
		if extractErr != nil {
			b.Fatalf("extractFunctionsFromFile failed: %v", extractErr)
		}
	}
}

func TestCodebaseBenchmark_Integration(t *testing.T) {
	// Integration test with a small synthetic codebase
	tempDir := t.TempDir()

	// Create a small test codebase
	testFiles := map[string]string{
		"pkg1/math.go": `package pkg1

func Add(a, b int) int {
	return a + b
}

func Subtract(a, b int) int {
	return a - b
}

func Multiply(x, y int) int {
	result := x * y
	return result
}`,
		"pkg2/operations.go": `package pkg2

func Sum(x, y int) int {
	return x + y
}

func Product(a, b int) int {
	temp := a * b
	return temp
}

func Divide(numerator, denominator int) int {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}`,
	}

	// Create test files
	for file, content := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	// Run benchmark
	benchmark := NewCodebaseBenchmark([]string{tempDir})
	benchmark.SetParameters(3, 100, nil)

	weights := config.SimilarityWeights{
		TreeEdit:           0.35,
		TokenSimilarity:    0.30,
		Structural:         0.25,
		Signature:          0.10,
		DifferentSignature: 0.30,
	}

	results, err := benchmark.BenchmarkWeights(t, weights)
	if err != nil {
		t.Fatalf("BenchmarkWeights failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]

	// Verify basic result properties
	if result.TotalFiles != 2 {
		t.Errorf("Expected 2 files, got %d", result.TotalFiles)
	}

	if result.TotalFunctions == 0 {
		t.Error("Expected to find some functions")
	}

	if result.ProcessingTime <= 0 {
		t.Error("Expected positive processing time")
	}

	// Should find some similar pairs (Add/Sum, Multiply/Product should be similar)
	if len(result.SimilarityPairs) == 0 {
		t.Error("Expected to find some similar pairs")
	}

	// Verify statistics structure
	if len(result.Statistics.FunctionSizeDistribution) == 0 {
		t.Error("Expected function size distribution")
	}

	if len(result.Statistics.ComplexityDistribution) == 0 {
		t.Error("Expected complexity distribution")
	}

	if len(result.Statistics.PackageAnalysis) == 0 {
		t.Error("Expected package analysis")
	}

	// Verify performance metrics
	if result.PerformanceData.TotalComparisons <= 0 {
		t.Error("Expected positive comparison count")
	}

	// Test report generation (should not panic)
	benchmark.PrintBenchmarkReport(results)
}
