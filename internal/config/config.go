package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration.
type Config struct {
	CLI        CLIConfig        `yaml:"cli"`
	Similarity SimilarityConfig `yaml:"similarity"`
	Processing ProcessingConfig `yaml:"processing"`
	Output     OutputConfig     `yaml:"output"`
	Ignore     IgnoreConfig     `yaml:"ignore"`
}

// CLIConfig contains CLI-specific configuration.
type CLIConfig struct {
	DefaultThreshold float64 `yaml:"default_threshold"`
	DefaultMinLines  int     `yaml:"default_min_lines"`
	DefaultFormat    string  `yaml:"default_format"`
	DefaultWorkers   int     `yaml:"default_workers"`
	DefaultCache     bool    `yaml:"default_cache"`
}

// SimilarityConfig contains similarity detection algorithm configuration.
type SimilarityConfig struct {
	Thresholds SimilarityThresholds `yaml:"thresholds"`
	Weights    SimilarityWeights    `yaml:"weights"`
	Limits     SimilarityLimits     `yaml:"limits"`
}

// SimilarityThresholds contains various threshold values.
type SimilarityThresholds struct {
	DefaultSimilarOperations float64 `yaml:"default_similar_operations"`
	StatementCountPenalty    float64 `yaml:"statement_count_penalty"`
	MinSimilarity            float64 `yaml:"min_similarity"`
}

// SimilarityWeights contains algorithm weights.
type SimilarityWeights struct {
	TreeEdit             float64 `yaml:"tree_edit"`
	TokenSimilarity      float64 `yaml:"token_similarity"`
	Structural           float64 `yaml:"structural"`
	Signature            float64 `yaml:"signature"`
	DifferentSignature   float64 `yaml:"different_signature"`
}

// SimilarityLimits contains performance and quality limits.
type SimilarityLimits struct {
	MaxSignatureLengthDiff int     `yaml:"max_signature_length_diff"`
	MaxLineDifferenceRatio float64 `yaml:"max_line_difference_ratio"`
	MaxCacheSize           int     `yaml:"max_cache_size"`
}

// ProcessingConfig contains processing-related configuration.
type ProcessingConfig struct {
	MaxEmptyVsPopulated int `yaml:"max_empty_vs_populated"`
}

// OutputConfig contains output formatting configuration.
type OutputConfig struct {
	RefactorSuggestion string `yaml:"refactor_suggestion"`
}

// IgnoreConfig contains ignore pattern configuration.
type IgnoreConfig struct {
	DefaultFile string   `yaml:"default_file"`
	Patterns    []string `yaml:"patterns"`
}

// Default returns a Config with sensible default values.
func Default() *Config {
	return &Config{
		CLI: CLIConfig{
			DefaultThreshold: 0.7,
			DefaultMinLines:  5,
			DefaultFormat:    "json",
			DefaultWorkers:   0, // 0 means use runtime.NumCPU()
			DefaultCache:     true,
		},
		Similarity: SimilarityConfig{
			Thresholds: SimilarityThresholds{
				DefaultSimilarOperations: 0.5,
				StatementCountPenalty:    0.5,
				MinSimilarity:            0.1,
			},
			Weights: SimilarityWeights{
				TreeEdit:           0.3,
				TokenSimilarity:    0.3,
				Structural:         0.25,
				Signature:          0.15,
				DifferentSignature: 0.3,
			},
			Limits: SimilarityLimits{
				MaxSignatureLengthDiff: 50,
				MaxLineDifferenceRatio: 3.0,
				MaxCacheSize:           10000,
			},
		},
		Processing: ProcessingConfig{
			MaxEmptyVsPopulated: 5,
		},
		Output: OutputConfig{
			RefactorSuggestion: "Consider extracting common logic into a shared function",
		},
		Ignore: IgnoreConfig{
			DefaultFile: ".similarityignore",
			Patterns: []string{
				"*_test.go",
				"testdata/",
				"vendor/",
				".git/",
			},
		},
	}
}

// Load loads configuration from file, falling back to defaults.
func Load(configPath string) (*Config, error) {
	cfg := Default()

	// If no config path provided, try default locations
	if configPath == "" {
		configPath = findConfigFile()
	}

	// If config file exists, load it
	if configPath != "" && fileExists(configPath) {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}
	}

	return cfg, nil
}

// Save saves the configuration to a YAML file.
func (c *Config) Save(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}

	return nil
}

// findConfigFile searches for configuration files in standard locations.
func findConfigFile() string {
	candidates := []string{
		".similarity-config.yaml",
		".similarity-config.yml",
		"similarity-config.yaml",
		"similarity-config.yml",
	}

	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}

	return ""
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Validate validates the configuration values.
func (c *Config) Validate() error {
	if c.CLI.DefaultThreshold < 0.0 || c.CLI.DefaultThreshold > 1.0 {
		return fmt.Errorf("default threshold must be between 0.0 and 1.0, got %f", c.CLI.DefaultThreshold)
	}

	if c.CLI.DefaultMinLines <= 0 {
		return fmt.Errorf("default min lines must be greater than 0, got %d", c.CLI.DefaultMinLines)
	}

	if c.CLI.DefaultFormat != "json" && c.CLI.DefaultFormat != "yaml" {
		return fmt.Errorf("default format must be 'json' or 'yaml', got %s", c.CLI.DefaultFormat)
	}

	if c.Similarity.Limits.MaxCacheSize <= 0 {
		return fmt.Errorf("max cache size must be greater than 0, got %d", c.Similarity.Limits.MaxCacheSize)
	}

	if c.Similarity.Limits.MaxLineDifferenceRatio <= 0 {
		return fmt.Errorf("max line difference ratio must be greater than 0, got %f", c.Similarity.Limits.MaxLineDifferenceRatio)
	}

	// Validate weights sum to reasonable values
	totalWeight := c.Similarity.Weights.TreeEdit + c.Similarity.Weights.TokenSimilarity + 
		c.Similarity.Weights.Structural + c.Similarity.Weights.Signature

	if totalWeight <= 0 {
		return fmt.Errorf("similarity weights must sum to a positive value, got %f", totalWeight)
	}

	return nil
}

// GetIgnoreFilePath returns the ignore file path, with fallback logic.
func (c *Config) GetIgnoreFilePath() string {
	if c.Ignore.DefaultFile == "" {
		return ".similarityignore"
	}
	return c.Ignore.DefaultFile
}