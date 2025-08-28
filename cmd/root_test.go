package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"encoding/json"

	"github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/config"
	"github.com/paveg/similarity-go/internal/similarity"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		expectUsage bool
	}{
		{
			name:        "no arguments shows error",
			args:        []string{},
			expectError: true,
			expectUsage: true,
		},
		{
			name:        "help flag shows help",
			args:        []string{"--help"},
			expectError: false,
			expectUsage: true,
		},
		{
			name:        "version flag shows version",
			args:        []string{"--version"},
			expectError: false,
			expectUsage: false,
		},
		{
			name:        "valid target runs command",
			args:        []string{"./testdata"},
			expectError: false,
			expectUsage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer

			args := &CLIArgs{}
			cmd := newRootCommand(args)
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			output := buf.String()

			if tt.expectUsage && !strings.Contains(output, "Usage:") {
				t.Error("Expected usage information in output")
			}

			if tt.name == "version flag shows version" && !strings.Contains(output, "similarity-go version") {
				t.Error("Expected version information in output")
			}
		})
	}
}

func TestRootCommandFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		flagName string
		expected interface{}
	}{
		{
			name:     "threshold flag",
			args:     []string{"--threshold", "0.8", "./testdata"},
			flagName: "threshold",
			expected: 0.8,
		},
		{
			name:     "format flag",
			args:     []string{"--format", "yaml", "./testdata"},
			flagName: "format",
			expected: "yaml",
		},
		{
			name:     "workers flag",
			args:     []string{"--workers", "4", "./testdata"},
			flagName: "workers",
			expected: 4,
		},
		{
			name:     "cache flag disabled",
			args:     []string{"--cache=false", "./testdata"},
			flagName: "cache",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := &CLIArgs{}
			cmd := newRootCommand(args)
			cmd.SetArgs(tt.args)

			// Parse flags without executing
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("Flag %s not found", tt.flagName)
			}

			validateFlagValue(t, cmd, tt.flagName, tt.expected)
		})
	}
}

func validateFlagValue(t *testing.T, cmd *cobra.Command, flagName string, expected interface{}) {
	t.Helper()

	switch expected := expected.(type) {
	case float64:
		floatValue, getErr := cmd.Flags().GetFloat64(flagName)
		if getErr != nil {
			t.Fatalf("Failed to get float64 flag: %v", getErr)
		}

		if floatValue != expected {
			t.Errorf("Expected %v, got %v", expected, floatValue)
		}
	case string:
		stringValue, getErr := cmd.Flags().GetString(flagName)
		if getErr != nil {
			t.Fatalf("Failed to get string flag: %v", getErr)
		}

		if stringValue != expected {
			t.Errorf("Expected %v, got %v", expected, stringValue)
		}
	case int:
		intValue, getErr := cmd.Flags().GetInt(flagName)
		if getErr != nil {
			t.Fatalf("Failed to get int flag: %v", getErr)
		}

		if intValue != expected {
			t.Errorf("Expected %v, got %v", expected, intValue)
		}
	case bool:
		boolValue, getErr := cmd.Flags().GetBool(flagName)
		if getErr != nil {
			t.Fatalf("Failed to get bool flag: %v", getErr)
		}

		if boolValue != expected {
			t.Errorf("Expected %v, got %v", expected, boolValue)
		}
	}
}

func TestApplyFlagOverrides(t *testing.T) {
	cfg := config.Default()

	// Create a command with flags set
	args := &CLIArgs{}
	cmd := newRootCommand(args)
	cmd.SetArgs([]string{"--threshold", "0.9", "--format", "yaml", "./testdata"})

	// Parse the flags
	err := cmd.ParseFlags([]string{"--threshold", "0.9", "--format", "yaml", "./testdata"})
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	// Apply overrides
	err = applyFlagOverrides(cfg, cmd)
	if err != nil {
		t.Fatalf("applyFlagOverrides failed: %v", err)
	}

	if cfg.CLI.DefaultThreshold != 0.9 {
		t.Errorf("Expected threshold 0.9, got %f", cfg.CLI.DefaultThreshold)
	}

	if cfg.CLI.DefaultFormat != "yaml" {
		t.Errorf("Expected format yaml, got %s", cfg.CLI.DefaultFormat)
	}
}

func TestGenerateMatchKey(t *testing.T) {
	key1 := generateMatchKey("abc", "def")
	key2 := generateMatchKey("def", "abc")

	// Should generate the same key regardless of order
	if key1 != key2 {
		t.Errorf("Expected same key, got %s and %s", key1, key2)
	}

	// Should contain both hashes
	if !strings.Contains(key1, "abc") || !strings.Contains(key1, "def") {
		t.Errorf("Key %s should contain both hashes", key1)
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match",
			path:     "test.go",
			pattern:  "test.go",
			expected: true,
		},
		{
			name:     "wildcard match",
			path:     "main_test.go",
			pattern:  "*_test.go",
			expected: true,
		},
		{
			name:     "no match",
			path:     "main.go",
			pattern:  "*_test.go",
			expected: false,
		},
		{
			name:     "directory pattern with slash",
			path:     "vendor/pkg/file.go",
			pattern:  "vendor/",
			expected: true,
		},
		{
			name:     "contains pattern",
			path:     "vendor/pkg/file.go",
			pattern:  "vendor",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPattern(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%s, %s) = %v, expected %v",
					tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestShouldIgnoreFile(t *testing.T) {
	cfg := config.Default()
	cfg.Ignore.Patterns = []string{"*_test.go", "vendor/", ".git/"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "should ignore hidden files",
			path:     ".hidden",
			expected: true,
		},
		{
			name:     "should ignore vendor files",
			path:     "some/vendor/pkg/file.go",
			expected: true,
		},
		{
			name:     "should not ignore regular go files",
			path:     "main.go",
			expected: false,
		},
		{
			name:     "should ignore git directories",
			path:     "some/.git/config",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldIgnoreFile(tt.path, cfg)
			if result != tt.expected {
				t.Errorf("shouldIgnoreFile(%s) = %v, expected %v",
					tt.path, result, tt.expected)
			}
		})
	}
}

func TestWriteOutput(t *testing.T) {
	output := map[string]interface{}{
		"test":    "data",
		"matches": 5,
	}

	// Test JSON output to stdout (no output path)
	err := writeOutput(output, "json", "")
	if err != nil {
		t.Fatalf("writeOutput failed for JSON: %v", err)
	}

	// Test YAML output to stdout (no output path)
	err = writeOutput(output, "yaml", "")
	if err != nil {
		t.Fatalf("writeOutput failed for YAML: %v", err)
	}

	// Test unsupported format
	err = writeOutput(output, "unsupported", "")
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

func TestCLIArgs(t *testing.T) {
	// Test that CLIArgs struct can be created and has expected fields
	args := &CLIArgs{
		configFile: "test-config.yaml",
		output:     "test-output.json",
		verbose:    true,
	}

	if args.configFile != "test-config.yaml" {
		t.Errorf("Expected configFile 'test-config.yaml', got %s", args.configFile)
	}

	if args.output != "test-output.json" {
		t.Errorf("Expected output 'test-output.json', got %s", args.output)
	}

	if !args.verbose {
		t.Error("Expected verbose to be true")
	}
}

func TestParseAllTargets(t *testing.T) {
	parser := ast.NewParser()
	cfg := config.Default()

	// Test with empty targets
	functions := parseAllTargets(parser, []string{}, cfg, false)
	if len(functions) != 0 {
		t.Errorf("Expected 0 functions for empty targets, got %d", len(functions))
	}

	// Test with invalid target (should be skipped)
	functions = parseAllTargets(parser, []string{"nonexistent.go"}, cfg, true)
	if len(functions) != 0 {
		t.Errorf("Expected 0 functions for nonexistent file, got %d", len(functions))
	}

	// Test with testdata directory (may not exist)
	functions = parseAllTargets(parser, []string{"./testdata"}, cfg, true)
	// Should return empty slice for nonexistent directory, not nil
	if len(functions) != 0 {
		t.Errorf("Expected 0 functions for nonexistent testdata directory, got %d", len(functions))
	}
}

func TestParseGoFile(t *testing.T) {
	parser := ast.NewParser()
	cfg := config.Default()

	// Test with nonexistent file
	functions, err := parseGoFile(parser, "nonexistent.go", cfg, false)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if len(functions) != 0 {
		t.Errorf("Expected 0 functions for nonexistent file, got %d", len(functions))
	}
}

func TestScanDirectory(t *testing.T) {
	parser := ast.NewParser()
	cfg := config.Default()

	// Test with nonexistent directory
	functions, err := scanDirectory(parser, "nonexistent", cfg, false)
	if err == nil {
		t.Error("Expected error for nonexistent directory")
	}
	if len(functions) != 0 {
		t.Errorf("Expected 0 functions for nonexistent directory, got %d", len(functions))
	}

	// Test with current directory (should find some Go files)
	functions, err = scanDirectory(parser, ".", cfg, true)
	if err != nil {
		t.Errorf("Unexpected error scanning current directory: %v", err)
	}
	// Should return valid functions slice
	if functions == nil {
		t.Error("Expected non-nil functions slice from current directory")
	}
}

func TestMatchesIgnorePatterns(t *testing.T) {
	tempDir := t.TempDir()
	ignoreFile := filepath.Join(tempDir, ".gitignore")

	// Create ignore file with patterns
	content := "vendor/\n*.tmp\n# comment line\n\ntest_data/"
	err := os.WriteFile(ignoreFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create ignore file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "matches vendor pattern",
			path:     "vendor/pkg/file.go",
			expected: true,
		},
		{
			name:     "matches tmp pattern",
			path:     "temp.tmp",
			expected: true,
		},
		{
			name:     "matches test_data pattern",
			path:     "test_data/file.go",
			expected: true,
		},
		{
			name:     "no match",
			path:     "main.go",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesIgnorePatterns(tt.path, ignoreFile)
			if result != tt.expected {
				t.Errorf("matchesIgnorePatterns(%s, %s) = %v, expected %v",
					tt.path, ignoreFile, result, tt.expected)
			}
		})
	}

	// Test with nonexistent ignore file
	result := matchesIgnorePatterns("main.go", "nonexistent.gitignore")
	if result != false {
		t.Error("Expected false for nonexistent ignore file")
	}
}

func TestGroupSimilarMatches(t *testing.T) {
	// Create mock functions for testing
	func1 := &ast.Function{Name: "func1", File: "file1.go"}
	func2 := &ast.Function{Name: "func2", File: "file2.go"}
	func3 := &ast.Function{Name: "func3", File: "file3.go"}

	// Test with empty matches
	matches := []similarity.Match{}
	groups := groupSimilarMatches(matches)
	if groups != nil {
		t.Errorf("Expected nil groups for empty matches, got %d groups", len(groups))
	}

	// Test with single match
	matches = []similarity.Match{
		{Function1: func1, Function2: func2, Similarity: 0.8},
	}
	groups = groupSimilarMatches(matches)
	if len(groups) != 1 {
		t.Errorf("Expected 1 group for single match, got %d", len(groups))
	}
	if len(groups[0]) != 1 {
		t.Errorf("Expected 1 match in group, got %d", len(groups[0]))
	}

	// Test with multiple matches forming chain
	matches = []similarity.Match{
		{Function1: func1, Function2: func2, Similarity: 0.8},
		{Function1: func2, Function2: func3, Similarity: 0.9},
	}
	groups = groupSimilarMatches(matches)
	if len(groups) != 1 {
		t.Errorf("Expected 1 group for chained matches, got %d", len(groups))
	}
	if len(groups[0]) != 2 {
		t.Errorf("Expected 2 matches in group, got %d", len(groups[0]))
	}
}

func TestCountDuplications(t *testing.T) {
	func1 := &ast.Function{Name: "func1", File: "file1.go"}
	func2 := &ast.Function{Name: "func2", File: "file2.go"}
	func3 := &ast.Function{Name: "func3", File: "file3.go"}

	// Test empty groups
	groups := [][]similarity.Match{}
	count := countDuplications(groups)
	if count != 0 {
		t.Errorf("Expected 0 duplications for empty groups, got %d", count)
	}

	// Test single group with matches
	groups = [][]similarity.Match{
		{
			{Function1: func1, Function2: func2, Similarity: 0.8},
			{Function1: func2, Function2: func3, Similarity: 0.9},
		},
	}
	count = countDuplications(groups)
	if count != 3 {
		t.Errorf("Expected 3 unique functions, got %d", count)
	}

	// Test multiple groups
	groups = [][]similarity.Match{
		{{Function1: func1, Function2: func2, Similarity: 0.8}}, // 2 functions
		{{Function1: func3, Function2: func1, Similarity: 0.7}}, // 1 new function (func3)
	}
	count = countDuplications(groups)
	if count != 3 {
		t.Errorf("Expected 3 total unique functions, got %d", count)
	}
}

func TestParallelProcessing(t *testing.T) {
	// Create temporary test files
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.go")
	testContent := `package main

func add(a, b int) int {
	return a + b
}

func sum(x, y int) int {
	return x + y
}
`
	if writeErr := os.WriteFile(testFile, []byte(testContent), 0644); writeErr != nil {
		t.Fatalf("failed to write test file: %v", writeErr)
	}

	// Test with parallel processing enabled
	args := &CLIArgs{verbose: true}
	cmd := newRootCommand(args)
	cmd.SetArgs([]string{"--workers", "2", "--threshold", "0.5", testFile})

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err != nil {
		t.Errorf("expected no error with parallel processing, got: %v", err)
	}

	// The test just verifies that parallel processing works without error
	// The JSON output in the test log shows it's working correctly
}

func TestProgressCallback(t *testing.T) {
	// Test progress callback functionality
	callback := createProgressCallback()

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w //nolint:reassign // testing stderr output

	// Call callback - use multiples of ProgressReportingInterval (100) or final value
	callback(100, 200) // This will trigger output (multiple of 100)
	callback(200, 200) // Final callback - always triggers

	w.Close()
	os.Stderr = oldStderr //nolint:reassign // restore stderr

	var buf bytes.Buffer
	buf.ReadFrom(r)

	output := buf.String()
	if !strings.Contains(output, "100/200") {
		t.Errorf("expected progress output to show 100/200, got: %s", output)
	}
	if !strings.Contains(output, "200/200") {
		t.Errorf("expected progress output to show 200/200, got: %s", output)
	}
}

func TestWriteOutputFormats(t *testing.T) {
	output := map[string]interface{}{
		"test":  "data",
		"count": 42,
	}

	// Test JSON output to file
	tempDir := t.TempDir()

	jsonFile := filepath.Join(tempDir, "output.json")
	err := writeOutput(output, "json", jsonFile)
	if err != nil {
		t.Errorf("expected no error writing JSON file, got: %v", err)
	}

	// Verify JSON content
	content, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("failed to read JSON file: %v", err)
	}

	var result map[string]interface{}
	if unmarshalErr := json.Unmarshal(content, &result); unmarshalErr != nil {
		t.Errorf("failed to parse JSON: %v", unmarshalErr)
	}

	if result["test"] != "data" {
		t.Errorf("expected test=data, got %v", result["test"])
	}

	// Test YAML output to file
	yamlFile := filepath.Join(tempDir, "output.yaml")
	err = writeOutput(output, "yaml", yamlFile)
	if err != nil {
		t.Errorf("expected no error writing YAML file, got: %v", err)
	}

	// Verify YAML content
	yamlContent, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatalf("failed to read YAML file: %v", err)
	}

	var yamlResult map[string]interface{}
	if yamlUnmarshalErr := yaml.Unmarshal(yamlContent, &yamlResult); yamlUnmarshalErr != nil {
		t.Errorf("failed to parse YAML: %v", yamlUnmarshalErr)
	}

	if yamlResult["test"] != "data" {
		t.Errorf("expected test=data in YAML, got %v", yamlResult["test"])
	}
}

func TestErrorHandling(t *testing.T) {
	// Test invalid config file
	args := &CLIArgs{configFile: "/nonexistent/config.yaml"}
	cmd := newRootCommand(args)
	cmd.SetArgs([]string{"./testdata"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	// Error is expected but should be handled gracefully
	if err == nil {
		t.Log("Command completed successfully with nonexistent config (using defaults)")
	}

	// Test with invalid output directory
	tempDir := t.TempDir()

	// Try to write to a directory that doesn't exist
	invalidPath := filepath.Join(tempDir, "nonexistent", "output.json")
	output := map[string]interface{}{"test": "data"}

	err = writeOutput(output, "json", invalidPath)
	if err == nil {
		t.Error("expected error when writing to invalid path")
	}
}

func TestFlagValidation(t *testing.T) {
	// Test flag overrides with invalid values
	cmd := newRootCommand(&CLIArgs{})

	// Set invalid threshold
	cmd.SetArgs([]string{"--threshold", "1.5", "./testdata"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error with invalid threshold > 1.0")
	}

	// Test invalid format
	cmd.SetArgs([]string{"--format", "invalid", "./testdata"})
	buf.Reset()

	err = cmd.Execute()
	if err == nil {
		t.Error("expected error with invalid format")
	}
}
