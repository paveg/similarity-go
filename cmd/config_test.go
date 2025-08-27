package main

import (
	"os"
	"testing"
	"github.com/paveg/similarity-go/internal/config"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "valid config",
			config:      config.Default(),
			expectError: false,
		},
		{
			name: "invalid threshold too low",
			config: func() *config.Config {
				cfg := config.Default()
				cfg.CLI.DefaultThreshold = -0.1
				return cfg
			}(),
			expectError: true,
		},
		{
			name: "invalid threshold too high",
			config: func() *config.Config {
				cfg := config.Default()
				cfg.CLI.DefaultThreshold = 1.1
				return cfg
			}(),
			expectError: true,
		},
		{
			name: "invalid format",
			config: func() *config.Config {
				cfg := config.Default()
				cfg.CLI.DefaultFormat = "xml"
				return cfg
			}(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRunSimilarityCheck(t *testing.T) {
	// Test that the function runs without error with valid inputs
	args := &CLIArgs{
		verbose: true,
	}
	cmd := newRootCommand(args)
	cmd.SetArgs([]string{"../testdata"})

	// Skip this test if testdata doesn't exist
	if _, err := os.Stat("../testdata"); os.IsNotExist(err) {
		t.Skip("testdata directory not found")
		return
	}

	err := runSimilarityCheck(args, cmd, []string{"../testdata"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
