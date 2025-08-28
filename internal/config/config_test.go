package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	// Test that all fields are properly initialized
	if cfg.CLI.DefaultThreshold != DefaultThreshold {
		t.Errorf("Expected CLI threshold %f, got %f", DefaultThreshold, cfg.CLI.DefaultThreshold)
	}

	if cfg.CLI.DefaultMinLines != DefaultMinLines {
		t.Errorf("Expected CLI min lines %d, got %d", DefaultMinLines, cfg.CLI.DefaultMinLines)
	}

	// Test similarity weights sum to 1.0
	weights := cfg.Similarity.Weights
	total := weights.TreeEdit + weights.TokenSimilarity + weights.Structural + weights.Signature
	if total != 1.0 {
		t.Errorf("Expected similarity weights to sum to 1.0, got %f", total)
	}

	// Test thresholds are within valid range
	if cfg.Similarity.Thresholds.DefaultSimilarOperations < 0 ||
		cfg.Similarity.Thresholds.DefaultSimilarOperations > 1 {
		t.Errorf(
			"DefaultSimilarOperations threshold out of range: %f",
			cfg.Similarity.Thresholds.DefaultSimilarOperations,
		)
	}

	// Test limits are positive
	if cfg.Similarity.Limits.MaxCacheSize <= 0 {
		t.Errorf("MaxCacheSize should be positive, got %d", cfg.Similarity.Limits.MaxCacheSize)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		modifier  func(*Config)
		wantError bool
	}{
		{
			name:      "valid config",
			modifier:  func(_ *Config) {},
			wantError: false,
		},
		{
			name: "negative threshold",
			modifier: func(c *Config) {
				c.CLI.DefaultThreshold = -0.1
			},
			wantError: true,
		},
		{
			name: "threshold too high",
			modifier: func(c *Config) {
				c.CLI.DefaultThreshold = 1.1
			},
			wantError: true,
		},
		{
			name: "negative min lines",
			modifier: func(c *Config) {
				c.CLI.DefaultMinLines = -1
			},
			wantError: true,
		},
		{
			name: "invalid format",
			modifier: func(c *Config) {
				c.CLI.DefaultFormat = "invalid"
			},
			wantError: true,
		},
		{
			name: "zero cache size",
			modifier: func(c *Config) {
				c.Similarity.Limits.MaxCacheSize = 0
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default()
			tt.modifier(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Config.Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestLoadAndSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Test loading non-existent file returns default
	cfg, err := Load(configPath)
	if err != nil {
		t.Errorf("Load() with non-existent file should not error, got: %v", err)
	}

	// Should return default config
	defaultCfg := Default()
	if cfg.CLI.DefaultThreshold != defaultCfg.CLI.DefaultThreshold {
		t.Errorf("Load() should return default config when file doesn't exist")
	}

	// Test saving config
	cfg.CLI.DefaultThreshold = 0.9
	err = cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Test loading saved config
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loadedCfg.CLI.DefaultThreshold != 0.9 {
		t.Errorf("Expected loaded threshold 0.9, got %f", loadedCfg.CLI.DefaultThreshold)
	}
}

func TestGetIgnoreFilePath(t *testing.T) {
	cfg := Default()

	// Test basic functionality
	result := cfg.GetIgnoreFilePath()
	if result == "" {
		t.Error("GetIgnoreFilePath() should not return empty string")
	}

	// Should contain "ignore" in the name
	expectedName := ".similarityignore"
	baseName := filepath.Base(result)
	if baseName != expectedName {
		t.Errorf("GetIgnoreFilePath() = %s, expected base name %s", result, expectedName)
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()
	existingFile := filepath.Join(tempDir, "existing.txt")
	nonExistingFile := filepath.Join(tempDir, "nonexisting.txt")

	// Create existing file
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if !fileExists(existingFile) {
		t.Error("fileExists() should return true for existing file")
	}

	if fileExists(nonExistingFile) {
		t.Error("fileExists() should return false for non-existing file")
	}
}

func TestFindConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, ".similarity-config.yaml")

	// Create config file
	if err := os.WriteFile(configFile, []byte("cli:\n  threshold: 0.8"), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Change to temp directory using t.Chdir (Go 1.20+)
	t.Chdir(tempDir)

	found := findConfigFile()
	expected := ".similarity-config.yaml"
	if found != expected {
		t.Errorf("findConfigFile() = %s, expected %s", found, expected)
	}
}
