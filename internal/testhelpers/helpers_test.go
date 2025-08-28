package testhelpers

import (
	"math"
	"os"
	"strings"
	"testing"
)

func TestDefaultTestConfig(t *testing.T) {
	cfg := DefaultTestConfig()

	if cfg.DefaultTolerance != 0.1 {
		t.Errorf("Expected default tolerance 0.1, got %f", cfg.DefaultTolerance)
	}

	if cfg.DefaultFuncName != "testFunc" {
		t.Errorf("Expected default function name 'testFunc', got %s", cfg.DefaultFuncName)
	}
}

func TestCreateTempGoFile(t *testing.T) {
	content := `package main

func hello() string {
	return "Hello, World!"
}`

	tempFile := CreateTempGoFile(t, content)
	defer os.Remove(tempFile)

	// Check file exists
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("Temp file was not created")
	}

	// Check file has .go extension
	if !strings.HasSuffix(tempFile, ".go") {
		t.Errorf("Temp file should have .go extension, got %s", tempFile)
	}

	// Check file content
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(data) != content {
		t.Errorf("File content mismatch.\nExpected: %s\nGot: %s", content, string(data))
	}
}

func TestParseGoSource(t *testing.T) {
	source := `package main

func add(a, b int) int {
	return a + b
}`

	fileSet, file := ParseGoSource(t, source)

	if fileSet == nil {
		t.Error("FileSet should not be nil")
	}

	if file == nil {
		t.Error("File should not be nil")
		return
	}

	if file.Name.Name != "main" {
		t.Errorf("Expected package name 'main', got %s", file.Name.Name)
	}
}

func TestAssertions(t *testing.T) {
	// Test AssertNoError
	t.Run("AssertNoError", func(t *testing.T) {
		// Should not panic with nil error
		AssertNoError(t, nil)
	})

	// Test AssertError
	t.Run("AssertError", func(t *testing.T) {
		// Should not panic with actual error
		AssertError(t, os.ErrNotExist)
	})

	// Test AssertEqual
	t.Run("AssertEqual", func(t *testing.T) {
		AssertEqual(t, "test", "test")
		AssertEqual(t, 42, 42)
		AssertEqual(t, true, true)
	})

	// Test AssertNotEmpty
	t.Run("AssertNotEmpty", func(t *testing.T) {
		AssertNotEmpty(t, "not empty")
	})

	// Test AssertContains
	t.Run("AssertContains", func(t *testing.T) {
		AssertContains(t, "hello world", "world")
	})
}

func TestAbsFloat(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{5.5, 5.5},
		{-3.2, 3.2},
		{0.0, 0.0},
		{math.Copysign(0, -1), 0.0},
	}

	for _, test := range tests {
		result := AbsFloat(test.input)
		if result != test.expected {
			t.Errorf("AbsFloat(%f) = %f, expected %f", test.input, result, test.expected)
		}
	}
}

func TestTableTestBasic(t *testing.T) {
	// Test that the TableTest type exists and can be used
	test := TableTest[string]{
		Name:     "test",
		Input:    "input",
		Expected: "expected",
	}

	if test.Name != "test" {
		t.Errorf("Expected name 'test', got %s", test.Name)
	}

	// Use the fields to avoid unused write warnings
	_ = test.Input
	_ = test.Expected
}
