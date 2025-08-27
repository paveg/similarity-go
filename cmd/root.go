package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/similarity"
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

//nolint:gocognit // Complex CLI integration logic acceptable
func runSimilarityCheck(cfg *Config, _ *cobra.Command, args []string) error {
	// Validate configuration
	if err := validateConfig(cfg.threshold, cfg.format, cfg.workers, cfg.minLines); err != nil {
		return err
	}

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

	// Initialize parser and detector
	parser := ast.NewParser()
	detector := similarity.NewDetector(cfg.threshold)

	// Parse all target files
	var allFunctions []*ast.Function
	for _, target := range args {
		if cfg.verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Parsing target: %s\n", target)
		}

		// Check if target is a file or directory
		if strings.HasSuffix(target, ".go") {
			result := parser.ParseFile(target)
			if result.IsErr() {
				if cfg.verbose {
					_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Error parsing %s: %v\n", target, result.Error())
				}
				continue
			}
			parseResult := result.Unwrap()

			// Filter functions by minimum lines
			for _, fn := range parseResult.Functions {
				if fn.LineCount >= cfg.minLines {
					allFunctions = append(allFunctions, fn)
				}
			}
		}
		// TODO: Handle directory scanning in Phase 4
	}

	if cfg.verbose {
		_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Found %d functions for analysis\n", len(allFunctions))
	}

	// Find similar functions
	similarMatches := detector.FindSimilarFunctions(allFunctions)

	// Group similar matches for better output formatting
	similarGroups := groupSimilarMatches(similarMatches)

	// Prepare output
	output := map[string]interface{}{
		"summary": map[string]interface{}{
			"total_functions":    len(allFunctions),
			"similar_groups":     len(similarGroups),
			"total_duplications": countDuplications(similarGroups),
		},
		"similar_groups": formatSimilarGroups(similarGroups),
	}

	// Output results
	outputWriter := os.Stdout
	if cfg.output != "" {
		file, err := os.Create(cfg.output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		outputWriter = file
	}

	// Format output
	switch cfg.format {
	case "json":
		encoder := json.NewEncoder(outputWriter)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("failed to encode JSON output: %w", err)
		}
	case "yaml":
		// TODO: Implement YAML output in Phase 5
		return errors.New("YAML output format not yet implemented")
	}

	return nil
}

// groupSimilarMatches groups similar matches by functions that appear together.
func groupSimilarMatches(matches []similarity.Match) [][]similarity.Match {
	if len(matches) == 0 {
		return nil
	}

	// Simple grouping: each match becomes its own group for now
	// TODO: Implement more sophisticated grouping in Phase 4
	var groups [][]similarity.Match
	for _, match := range matches {
		groups = append(groups, []similarity.Match{match})
	}

	return groups
}

// countDuplications counts the total number of duplicate functions across all groups.
func countDuplications(groups [][]similarity.Match) int {
	total := 0
	for _, group := range groups {
		if len(group) > 0 {
			// Each match represents 2 functions, so count the unique functions
			const functionsPerMatch = 2
			total += len(group) * functionsPerMatch
		}
	}
	return total
}

// formatSimilarGroups formats similarity groups for output.
func formatSimilarGroups(groups [][]similarity.Match) []map[string]interface{} {
	var result []map[string]interface{}

	for i, group := range groups {
		if len(group) == 0 {
			continue
		}

		// For now, each group contains one match (pair of similar functions)
		match := group[0]

		functions := []map[string]interface{}{
			{
				"file":       match.Function1.File,
				"function":   match.Function1.Name,
				"start_line": match.Function1.StartLine,
				"end_line":   match.Function1.EndLine,
				"hash":       match.Function1.Hash(),
			},
			{
				"file":       match.Function2.File,
				"function":   match.Function2.Name,
				"start_line": match.Function2.StartLine,
				"end_line":   match.Function2.EndLine,
				"hash":       match.Function2.Hash(),
			},
		}

		groupData := map[string]interface{}{
			"id":                  fmt.Sprintf("group_%d", i+1),
			"similarity_score":    match.Similarity,
			"functions":           functions,
			"refactor_suggestion": "Consider extracting common logic into a shared function",
		}

		result = append(result, groupData)
	}

	return result
}
