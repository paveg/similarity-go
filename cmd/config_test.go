package main

import (
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		threshold     float64
		format        string
		workers       int
		minLines      int
		expectError   bool
		expectedError string
	}{
		{
			name:        "valid config",
			threshold:   0.8,
			format:      "json",
			workers:     4,
			minLines:    5,
			expectError: false,
		},
		{
			name:          "invalid threshold too low",
			threshold:     -0.1,
			format:        "json",
			workers:       4,
			minLines:      5,
			expectError:   true,
			expectedError: "threshold must be between 0.0 and 1.0",
		},
		{
			name:          "invalid threshold too high",
			threshold:     1.1,
			format:        "json",
			workers:       4,
			minLines:      5,
			expectError:   true,
			expectedError: "threshold must be between 0.0 and 1.0",
		},
		{
			name:          "invalid format",
			threshold:     0.8,
			format:        "xml",
			workers:       4,
			minLines:      5,
			expectError:   true,
			expectedError: "format must be json or yaml",
		},
		{
			name:          "invalid workers",
			threshold:     0.8,
			format:        "json",
			workers:       0,
			minLines:      5,
			expectError:   true,
			expectedError: "workers must be greater than 0",
		},
		{
			name:          "invalid min lines",
			threshold:     0.8,
			format:        "json",
			workers:       4,
			minLines:      0,
			expectError:   true,
			expectedError: "min-lines must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.threshold, tt.format, tt.workers, tt.minLines)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectError && err != nil && err.Error() != tt.expectedError {
				t.Errorf("Expected error %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestRunSimilarityCheck(t *testing.T) {
	// Test that the function runs without error with valid inputs
	cmd := newRootCommand()
	cmd.SetArgs([]string{"./testdata"})

	// Set some config values
	config.threshold = 0.8
	config.format = "json"
	config.workers = 2
	config.verbose = true

	err := runSimilarityCheck(cmd, []string{"./testdata"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
