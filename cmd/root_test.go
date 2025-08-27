package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

			config := &Config{}
			cmd := newRootCommand(config)
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
			config := &Config{}
			cmd := newRootCommand(config)
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
