package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

Targets can be:
  - Individual Go files: file1.go file2.go
  - Directories: ./internal ./cmd
  - Mixed: ./internal file.go ./pkg

Automatically scans directories recursively for .go files while ignoring:
  - Hidden files and directories (starting with .)
  - vendor/ directories
  - Build directories (bin/, build/, dist/, target/, .git/)

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

		// Process target (file or directory)
		var functions []*ast.Function
		var err error

		if strings.HasSuffix(target, ".go") {
			functions, err = parseGoFile(parser, target, cfg)
		} else {
			functions, err = scanDirectory(parser, target, cfg)
		}

		if err != nil {
			if cfg.verbose {
				_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Error processing %s: %v\n", target, err)
			}
			continue
		}

		allFunctions = append(allFunctions, functions...)
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

// countDuplications counts the total number of unique duplicate functions across all groups.
func countDuplications(groups [][]similarity.Match) int {
	uniqueFunctions := make(map[string]bool)

	for _, group := range groups {
		for _, match := range group {
			// Use function hash to identify unique functions
			uniqueFunctions[match.Function1.Hash()] = true
			uniqueFunctions[match.Function2.Hash()] = true
		}
	}

	return len(uniqueFunctions)
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

// parseGoFile parses a single Go file and returns functions that meet the minimum line criteria.
func parseGoFile(parser *ast.Parser, filePath string, cfg *Config) ([]*ast.Function, error) {
	result := parser.ParseFile(filePath)
	if result.IsErr() {
		return nil, result.Error()
	}

	parseResult := result.Unwrap()
	var functions []*ast.Function

	// Filter functions by minimum lines
	for _, fn := range parseResult.Functions {
		if fn.LineCount >= cfg.minLines {
			functions = append(functions, fn)
		}
	}

	return functions, nil
}

// scanDirectory recursively scans a directory for Go files and parses them.
func scanDirectory(parser *ast.Parser, dirPath string, cfg *Config) ([]*ast.Function, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", dirPath, err)
	}

	if !info.IsDir() {
		return parseGoFile(parser, dirPath, cfg)
	}

	var allFunctions []*ast.Function
	walkFunc := createWalkFunc(parser, cfg, &allFunctions)

	err = filepath.Walk(dirPath, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", dirPath, err)
	}

	return allFunctions, nil
}

// createWalkFunc creates a filepath.WalkFunc for directory traversal.
func createWalkFunc(parser *ast.Parser, cfg *Config, allFunctions *[]*ast.Function) filepath.WalkFunc {
	return func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			if cfg.verbose {
				_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Error accessing %s: %v\n", path, err)
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		if shouldIgnoreFile(path, cfg) {
			if cfg.verbose {
				_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Ignoring %s\n", path)
			}
			return nil
		}

		return processGoFile(parser, path, cfg, allFunctions)
	}
}

// processGoFile parses a Go file and adds functions to the collection.
func processGoFile(parser *ast.Parser, path string, cfg *Config, allFunctions *[]*ast.Function) error {
	if cfg.verbose {
		_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Parsing file: %s\n", path)
	}

	functions, parseErr := parseGoFile(parser, path, cfg)
	if parseErr != nil {
		if cfg.verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Error parsing %s: %v\n", path, parseErr)
		}
		return nil
	}

	*allFunctions = append(*allFunctions, functions...)
	return nil
}

// shouldIgnoreFile determines if a file should be ignored based on configuration.
func shouldIgnoreFile(filePath string, cfg *Config) bool {
	// Skip hidden files and directories
	base := filepath.Base(filePath)
	if strings.HasPrefix(base, ".") {
		return true
	}

	// Skip vendor directories
	if strings.Contains(filePath, "/vendor/") || strings.Contains(filePath, "\\vendor\\") {
		return true
	}

	// Skip common build/output directories
	ignoreDirs := []string{"/bin/", "/build/", "/dist/", "/target/", "/.git/"}
	for _, ignoreDir := range ignoreDirs {
		if strings.Contains(filePath, ignoreDir) ||
			strings.Contains(filePath, strings.ReplaceAll(ignoreDir, "/", "\\")) {
			return true
		}
	}

	// Check .similarityignore file patterns
	if cfg.ignoreFile != "" {
		return matchesIgnorePatterns(filePath, cfg.ignoreFile)
	}

	return false
}

// matchesIgnorePatterns checks if a file path matches any patterns in the ignore file.
func matchesIgnorePatterns(filePath, ignoreFilePath string) bool {
	ignoreFile, err := os.Open(ignoreFilePath)
	if err != nil {
		// If ignore file doesn't exist or can't be read, don't ignore anything
		return false
	}
	defer ignoreFile.Close()

	scanner := bufio.NewScanner(ignoreFile)
	for scanner.Scan() {
		pattern := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}

		// Check if pattern matches the file path
		if matchesPattern(filePath, pattern) {
			return true
		}
	}

	return false
}

// matchesPattern checks if a file path matches a glob-like pattern.
func matchesPattern(filePath, pattern string) bool {
	// Normalize path separators
	filePath = filepath.ToSlash(filePath)
	pattern = filepath.ToSlash(pattern)

	// Handle simple wildcards and exact matches
	matched, err := filepath.Match(pattern, filepath.Base(filePath))
	if err == nil && matched {
		return true
	}

	// Check if pattern matches anywhere in the path
	if strings.Contains(filePath, pattern) {
		return true
	}

	// Handle directory patterns (ending with /)
	if strings.HasSuffix(pattern, "/") {
		dirPattern := strings.TrimSuffix(pattern, "/")
		if strings.Contains(filePath, "/"+dirPattern+"/") || strings.HasPrefix(filePath, dirPattern+"/") {
			return true
		}
	}

	return false
}
