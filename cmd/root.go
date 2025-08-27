package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	// Default configuration values.
	defaultThreshold = 0.7
	defaultMinLines  = 5
)

var (
	version   = "dev"     // Will be set during build //nolint:gochecknoglobals // build-time variables
	gitCommit = "none"    //nolint:gochecknoglobals // build-time variables
	buildTime = "unknown" //nolint:gochecknoglobals // build-time variables
)

// Config represents the application configuration.
type Config struct {
	threshold  float64
	format     string
	workers    int
	useCache   bool
	ignoreFile string
	output     string
	verbose    bool
	minLines   int
}

func newRootCommand(cfg *Config) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "similarity-go [flags] <targets...>",
		Short: "Go code similarity detection tool",
		Long: `A CLI tool that uses AST analysis to find duplicate and similar code patterns in Go projects.

Detects similar code blocks that could be consolidated, helping with refactoring and maintaining code quality.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, gitCommit, buildTime),
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSimilarityCheck(cfg, cmd, args)
		},
	}

	// Add flags
	rootCmd.Flags().Float64VarP(&cfg.threshold, "threshold", "t", defaultThreshold, "similarity threshold (0.0-1.0)")
	rootCmd.Flags().StringVarP(&cfg.format, "format", "f", "json", "output format (json|yaml)")
	rootCmd.Flags().IntVarP(&cfg.workers, "workers", "w", runtime.NumCPU(), "number of parallel workers")
	rootCmd.Flags().BoolVar(&cfg.useCache, "cache", true, "enable caching")
	rootCmd.Flags().StringVar(&cfg.ignoreFile, "ignore", ".similarityignore", "ignore file path")
	rootCmd.Flags().StringVarP(&cfg.output, "output", "o", "", "output file (default: stdout)")
	rootCmd.Flags().BoolVarP(&cfg.verbose, "verbose", "v", false, "verbose output")
	rootCmd.Flags().IntVar(&cfg.minLines, "min-lines", defaultMinLines, "minimum function lines to analyze")

	return rootCmd
}

func validateConfig(threshold float64, format string, workers, minLines int) error {
	if threshold < 0.0 || threshold > 1.0 {
		return errors.New("threshold must be between 0.0 and 1.0")
	}

	if format != "json" && format != "yaml" {
		return errors.New("format must be json or yaml")
	}

	if workers <= 0 {
		return errors.New("workers must be greater than 0")
	}

	if minLines <= 0 {
		return errors.New("min-lines must be greater than 0")
	}

	return nil
}

func runSimilarityCheck(cfg *Config, _ *cobra.Command, args []string) error {
	// Validate configuration
	if err := validateConfig(cfg.threshold, cfg.format, cfg.workers, cfg.minLines); err != nil {
		return err
	}

	// TODO: Implement actual similarity checking logic
	if cfg.verbose {
		_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Running similarity analysis on targets: %v\n", args)
		_, _ = fmt.Fprintf(
			os.Stderr,
			"[similarity-go] Configuration - Threshold: %.2f, Format: %s, Workers: %d, Cache: %v\n",
			cfg.threshold,
			cfg.format,
			cfg.workers,
			cfg.useCache,
		)
	}

	_, _ = fmt.Fprint(os.Stderr, "[similarity-go] Similarity analysis not yet implemented\n")

	return nil
}
