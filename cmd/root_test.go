package main

import (
	"bytes"
	"strings"
	"testing"
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

			cmd := newRootCommand()
			cmd.SetOutput(&buf)
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
			cmd := newRootCommand()
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

			switch expected := tt.expected.(type) {
			case float64:
				value, err := cmd.Flags().GetFloat64(tt.flagName)
				if err != nil {
					t.Fatalf("Failed to get float64 flag: %v", err)
				}

				if value != expected {
					t.Errorf("Expected %v, got %v", expected, value)
				}
			case string:
				value, err := cmd.Flags().GetString(tt.flagName)
				if err != nil {
					t.Fatalf("Failed to get string flag: %v", err)
				}

				if value != expected {
					t.Errorf("Expected %v, got %v", expected, value)
				}
			case int:
				value, err := cmd.Flags().GetInt(tt.flagName)
				if err != nil {
					t.Fatalf("Failed to get int flag: %v", err)
				}

				if value != expected {
					t.Errorf("Expected %v, got %v", expected, value)
				}
			case bool:
				value, err := cmd.Flags().GetBool(tt.flagName)
				if err != nil {
					t.Fatalf("Failed to get bool flag: %v", err)
				}

				if value != expected {
					t.Errorf("Expected %v, got %v", expected, value)
				}
			}
		})
	}
}
