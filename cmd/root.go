package main

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	version   = "dev" // Will be set during build
	gitCommit = "none"
	buildTime = "unknown"
)

// Configuration flags.
var config struct {
	threshold  float64
	format     string
	workers    int
	useCache   bool
	ignoreFile string
	output     string
	verbose    bool
	minLines   int
}

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "similarity-go [flags] <targets...>",
		Short: "Go code similarity detection tool",
		Long: `A CLI tool that uses AST analysis to find duplicate and similar code patterns in Go projects.
		
Detects similar code blocks that could be consolidated, helping with refactoring and maintaining code quality.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, gitCommit, buildTime),
		Args:    cobra.MinimumNArgs(1),
		RunE:    runSimilarityCheck,
	}

	// Add flags
	rootCmd.Flags().Float64VarP(&config.threshold, "threshold", "t", 0.7, "similarity threshold (0.0-1.0)")
	rootCmd.Flags().StringVarP(&config.format, "format", "f", "json", "output format (json|yaml)")
	rootCmd.Flags().IntVarP(&config.workers, "workers", "w", runtime.NumCPU(), "number of parallel workers")
	rootCmd.Flags().BoolVar(&config.useCache, "cache", true, "enable caching")
	rootCmd.Flags().StringVar(&config.ignoreFile, "ignore", ".similarityignore", "ignore file path")
	rootCmd.Flags().StringVarP(&config.output, "output", "o", "", "output file (default: stdout)")
	rootCmd.Flags().BoolVarP(&config.verbose, "verbose", "v", false, "verbose output")
	rootCmd.Flags().IntVar(&config.minLines, "min-lines", 5, "minimum function lines to analyze")

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

func runSimilarityCheck(cmd *cobra.Command, args []string) error {
	// Validate configuration
	if err := validateConfig(config.threshold, config.format, config.workers, config.minLines); err != nil {
		return err
	}

	// TODO: Implement actual similarity checking logic
	if config.verbose {
		fmt.Printf("Running similarity analysis on: %v\n", args)
		fmt.Printf("Threshold: %.2f\n", config.threshold)
		fmt.Printf("Format: %s\n", config.format)
		fmt.Printf("Workers: %d\n", config.workers)
		fmt.Printf("Cache: %v\n", config.useCache)
	}

	fmt.Println("Similarity analysis not yet implemented")

	return nil
}
