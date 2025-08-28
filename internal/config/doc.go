// Package config provides comprehensive configuration management for the
// similarity-go tool with YAML-based configuration and validation.
//
// This package implements a hierarchical configuration system that supports
// both file-based configuration and command-line overrides with proper
// validation and sensible defaults.
//
// Configuration Structure:
//   - CLI: Command-line interface defaults and options
//   - Similarity: Algorithm weights, thresholds, and limits
//   - Processing: Parallel processing and performance settings
//   - Output: Output formatting and reporting configuration
//   - Ignore: File and directory ignore patterns
//
// The configuration system supports YAML files with the following discovery order:
//  1. File specified by --config flag
//  2. .similarity-config.yaml (current directory)
//  3. Built-in defaults
//
// Example Configuration:
//
//	cli:
//	  default_threshold: 0.8
//	  default_min_lines: 5
//	  default_workers: 0
//
//	similarity:
//	  weights:
//	    tree_edit: 0.3
//	    token_similarity: 0.3
//	    structural: 0.25
//	    signature: 0.15
//
//	ignore:
//	  patterns:
//	    - "*_test.go"
//	    - "vendor/"
//
// All configuration values are validated at startup with clear error messages
// for invalid settings, ensuring reliable operation across different environments.
package config
