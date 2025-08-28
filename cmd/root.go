package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/config"
	"github.com/paveg/similarity-go/internal/similarity"
	"github.com/paveg/similarity-go/internal/worker"
	"github.com/paveg/similarity-go/pkg/mathutil"
)

var (
	version   = "dev"     // Will be set during build //nolint:gochecknoglobals // build-time variables
	gitCommit = "none"    //nolint:gochecknoglobals // build-time variables
	buildTime = "unknown" //nolint:gochecknoglobals // build-time variables
)

// CLIArgs represents the CLI-specific arguments that extend configuration.
type CLIArgs struct {
	configFile string
	output     string
	verbose    bool
}

func newRootCommand(args *CLIArgs) *cobra.Command {
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
		RunE: func(cmd *cobra.Command, targets []string) error {
			return runSimilarityCheck(args, cmd, targets)
		},
	}

	// Add flags - configuration will be loaded inside runSimilarityCheck
	rootCmd.Flags().StringVarP(&args.configFile, "config", "c", "", "config file path")
	rootCmd.Flags().StringVarP(&args.output, "output", "o", "", "output file (default: stdout)")
	rootCmd.Flags().BoolVarP(&args.verbose, "verbose", "v", false, "verbose output")

	// Allow overriding config values via flags - will be parsed in runSimilarityCheck
	rootCmd.Flags().Float64P("threshold", "t", 0, "similarity threshold (0.0-1.0)")
	rootCmd.Flags().StringP("format", "f", "", "output format (json|yaml)")
	rootCmd.Flags().IntP("workers", "w", 0, "number of parallel workers")
	rootCmd.Flags().Bool("cache", false, "enable caching")
	rootCmd.Flags().String("ignore", "", "ignore file path")
	rootCmd.Flags().Int("min-lines", 0, "minimum function lines to analyze")

	return rootCmd
}

func applyFlagOverrides(cfg *config.Config, cmd *cobra.Command) error {
	// Apply flag overrides to configuration
	if threshold, _ := cmd.Flags().GetFloat64("threshold"); threshold > 0 {
		cfg.CLI.DefaultThreshold = threshold
	}
	if format, _ := cmd.Flags().GetString("format"); format != "" {
		cfg.CLI.DefaultFormat = format
	}
	if workers, _ := cmd.Flags().GetInt("workers"); workers > 0 {
		cfg.CLI.DefaultWorkers = workers
	}
	if cache, _ := cmd.Flags().GetBool("cache"); cmd.Flags().Changed("cache") {
		cfg.CLI.DefaultCache = cache
	}
	if ignore, _ := cmd.Flags().GetString("ignore"); ignore != "" {
		cfg.Ignore.DefaultFile = ignore
	}
	if minLines, _ := cmd.Flags().GetInt("min-lines"); minLines > 0 {
		cfg.CLI.DefaultMinLines = minLines
	}

	return cfg.Validate()
}

func runSimilarityCheck(args *CLIArgs, cmd *cobra.Command, targets []string) error {
	// Load configuration
	cfg, err := config.Load(args.configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply flag overrides
	if overrideErr := applyFlagOverrides(cfg, cmd); overrideErr != nil {
		return fmt.Errorf("invalid configuration: %w", overrideErr)
	}

	if args.verbose {
		_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Running similarity analysis on targets: %v\n", targets)
		_, _ = fmt.Fprintf(
			os.Stderr,
			"[similarity-go] Configuration - Threshold: %.2f, Format: %s, Workers: %d, Cache: %v\n",
			cfg.CLI.DefaultThreshold,
			cfg.CLI.DefaultFormat,
			cfg.CLI.DefaultWorkers,
			cfg.CLI.DefaultCache,
		)
	}

	// Initialize parser and detector
	parser := ast.NewParser()
	detector := similarity.NewDetectorWithConfig(cfg.CLI.DefaultThreshold, cfg)

	// Parse all target files
	allFunctions := parseAllTargets(parser, targets, cfg, args.verbose)

	if args.verbose {
		_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Found %d functions for analysis\n", len(allFunctions))
	}

	// Find similar functions (use parallel processing if workers > 1)
	var similarMatches []similarity.Match

	if cfg.CLI.DefaultWorkers > 1 {
		// Use parallel processing
		if args.verbose {
			_, _ = fmt.Fprintf(
				os.Stderr,
				"[similarity-go] Using parallel processing with %d workers\n",
				cfg.CLI.DefaultWorkers,
			)
		}

		// Create progress callback for verbose mode
		var progressCallback func(completed, total int)
		if args.verbose {
			progressCallback = func(completed, total int) {
				if completed%100 == 0 || completed == total {
					_, _ = fmt.Fprintf(os.Stderr, "\r[similarity-go] Progress: %d/%d comparisons (%.1f%%)",
						completed, total, float64(completed)/float64(total)*100)
					if completed == total {
						_, _ = fmt.Fprintf(os.Stderr, "\n")
					}
				}
			}
		}

		// Use worker for parallel processing
		parallelWorker := worker.NewSimilarityWorker(detector, cfg.CLI.DefaultWorkers, cfg.CLI.DefaultThreshold)
		var parallelErr error
		similarMatches, parallelErr = parallelWorker.FindSimilarFunctions(allFunctions, progressCallback)
		if parallelErr != nil {
			return fmt.Errorf("parallel similarity calculation failed: %w", parallelErr)
		}
	} else {
		// Use serial processing
		if args.verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Using serial processing\n")
		}
		similarMatches = detector.FindSimilarFunctions(allFunctions)
	}

	// Group similar matches for better output formatting
	similarGroups := groupSimilarMatches(similarMatches)

	// Prepare output
	output := map[string]any{
		"summary": map[string]any{
			"total_functions":    len(allFunctions),
			"similar_groups":     len(similarGroups),
			"total_duplications": countDuplications(similarGroups),
		},
		"similar_groups": formatSimilarGroups(similarGroups, cfg),
	}

	// Output results
	return writeOutput(output, cfg.CLI.DefaultFormat, args.output)
}

// writeOutput writes the given output in the specified format to the given output path.
func writeOutput(output map[string]any, format, outputPath string) error {
	outputWriter := os.Stdout
	if outputPath != "" {
		file, createErr := os.Create(outputPath)
		if createErr != nil {
			return fmt.Errorf("failed to create output file: %w", createErr)
		}
		defer file.Close()
		outputWriter = file
	}

	// Format output
	switch format {
	case "json":
		encoder := json.NewEncoder(outputWriter)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(output); encodeErr != nil {
			return fmt.Errorf("failed to encode JSON output: %w", encodeErr)
		}
	case "yaml":
		encoder := yaml.NewEncoder(outputWriter)
		defer encoder.Close()
		if yamlEncodeErr := encoder.Encode(output); yamlEncodeErr != nil {
			return fmt.Errorf("failed to encode YAML output: %w", yamlEncodeErr)
		}
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

// buildSimilarityGraph creates a graph of function similarities from matches.
func buildSimilarityGraph(
	matches []similarity.Match,
) (map[string]map[string]similarity.Match, map[string]*ast.Function) {
	functionGraph := make(map[string]map[string]similarity.Match)
	allFunctions := make(map[string]*ast.Function)

	for _, match := range matches {
		hash1 := match.Function1.Hash()
		hash2 := match.Function2.Hash()

		// Store functions by hash
		allFunctions[hash1] = match.Function1
		allFunctions[hash2] = match.Function2

		// Add edges in both directions
		if functionGraph[hash1] == nil {
			functionGraph[hash1] = make(map[string]similarity.Match)
		}
		if functionGraph[hash2] == nil {
			functionGraph[hash2] = make(map[string]similarity.Match)
		}

		functionGraph[hash1][hash2] = match
		functionGraph[hash2][hash1] = similarity.Match{
			Function1:  match.Function2,
			Function2:  match.Function1,
			Similarity: match.Similarity,
		}
	}

	return functionGraph, allFunctions
}

// groupSimilarMatches groups similar matches by functions that appear together.
// Uses transitive clustering to group functions that are similar to each other.
func groupSimilarMatches(matches []similarity.Match) [][]similarity.Match {
	if len(matches) == 0 {
		return nil
	}

	// Create a graph of function similarities
	functionGraph, allFunctions := buildSimilarityGraph(matches)

	// Use Union-Find (Disjoint Set) to group connected components
	groups := findConnectedGroups(functionGraph, allFunctions)

	// Convert to the required format
	return convertGroupsToMatches(groups, functionGraph)
}

// convertGroupsToMatches converts function groups to similarity match groups.
func convertGroupsToMatches(
	groups [][]*ast.Function,
	functionGraph map[string]map[string]similarity.Match,
) [][]similarity.Match {
	var result [][]similarity.Match

	for _, group := range groups {
		const minGroupSize = 2
		if len(group) < minGroupSize {
			continue // Skip single function groups
		}

		var groupMatches []similarity.Match
		added := make(map[string]bool)

		// Generate all pairwise matches within the group
		for i, func1 := range group {
			for j := i + 1; j < len(group); j++ {
				func2 := group[j]
				hash1 := func1.Hash()
				hash2 := func2.Hash()

				// Avoid duplicate matches
				key := generateMatchKey(hash1, hash2)

				if !added[key] {
					if match, exists := functionGraph[hash1][hash2]; exists {
						groupMatches = append(groupMatches, match)
						added[key] = true
					}
				}
			}
		}

		if len(groupMatches) > 0 {
			result = append(result, groupMatches)
		}
	}

	return result
}

// generateMatchKey creates a consistent key for a pair of function hashes.
func generateMatchKey(hash1, hash2 string) string {
	return mathutil.CreateConsistentKey(hash1, hash2)
}

// findConnectedGroups uses DFS to find connected components in the similarity graph.
func findConnectedGroups(
	graph map[string]map[string]similarity.Match,
	allFunctions map[string]*ast.Function,
) [][]*ast.Function {
	visited := make(map[string]bool)
	var groups [][]*ast.Function

	// Perform DFS from each unvisited node
	for functionHash := range allFunctions {
		if !visited[functionHash] {
			var group []*ast.Function
			dfsVisit(graph, allFunctions, functionHash, visited, &group)
			if len(group) > 1 { // Only include groups with multiple functions
				groups = append(groups, group)
			}
		}
	}

	return groups
}

// dfsVisit performs depth-first search to collect all connected functions.
func dfsVisit(graph map[string]map[string]similarity.Match, allFunctions map[string]*ast.Function,
	currentHash string, visited map[string]bool, group *[]*ast.Function) {
	visited[currentHash] = true
	if function, exists := allFunctions[currentHash]; exists {
		*group = append(*group, function)
	}

	// Visit all connected functions
	if neighbors, exists := graph[currentHash]; exists {
		for neighborHash := range neighbors {
			if !visited[neighborHash] {
				dfsVisit(graph, allFunctions, neighborHash, visited, group)
			}
		}
	}
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
func formatSimilarGroups(groups [][]similarity.Match, cfg *config.Config) []map[string]any {
	var result []map[string]any

	for i, group := range groups {
		if len(group) == 0 {
			continue
		}

		// For now, each group contains one match (pair of similar functions)
		match := group[0]

		functions := []map[string]any{
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

		groupData := map[string]any{
			"id":                  fmt.Sprintf("group_%d", i+1),
			"similarity_score":    match.Similarity,
			"functions":           functions,
			"refactor_suggestion": cfg.Output.RefactorSuggestion,
		}

		result = append(result, groupData)
	}

	return result
}

// parseGoFile parses a single Go file and returns functions that meet the minimum line criteria.
func parseGoFile(parser *ast.Parser, filePath string, cfg *config.Config, _ bool) ([]*ast.Function, error) {
	result := parser.ParseFile(filePath)
	if result.IsErr() {
		return nil, result.Error()
	}

	parseResult := result.Unwrap()
	var functions []*ast.Function

	// Filter functions by minimum lines
	for _, fn := range parseResult.Functions {
		if fn.LineCount >= cfg.CLI.DefaultMinLines {
			functions = append(functions, fn)
		}
	}

	return functions, nil
}

// parseAllTargets parses all target files and directories, returning all functions.
// Errors from individual targets are logged but do not stop processing.
func parseAllTargets(parser *ast.Parser, targets []string, cfg *config.Config, verbose bool) []*ast.Function {
	var allFunctions []*ast.Function

	for _, target := range targets {
		if verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Parsing target: %s\n", target)
		}

		// Process target (file or directory)
		var functions []*ast.Function
		var parseErr error

		if strings.HasSuffix(target, ".go") {
			functions, parseErr = parseGoFile(parser, target, cfg, verbose)
		} else {
			functions, parseErr = scanDirectory(parser, target, cfg, verbose)
		}

		if parseErr != nil {
			if verbose {
				_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Error processing %s: %v\n", target, parseErr)
			}
			continue
		}

		allFunctions = append(allFunctions, functions...)
	}

	return allFunctions
}

// scanDirectory recursively scans a directory for Go files and parses them.
func scanDirectory(parser *ast.Parser, dirPath string, cfg *config.Config, verbose bool) ([]*ast.Function, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", dirPath, err)
	}

	if !info.IsDir() {
		return parseGoFile(parser, dirPath, cfg, verbose)
	}

	var allFunctions []*ast.Function
	walkFunc := createWalkFunc(parser, cfg, &allFunctions, verbose)

	err = filepath.Walk(dirPath, walkFunc)
	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", dirPath, err)
	}

	return allFunctions, nil
}

// createWalkFunc creates a filepath.WalkFunc for directory traversal.
func createWalkFunc(
	parser *ast.Parser,
	cfg *config.Config,
	allFunctions *[]*ast.Function,
	verbose bool,
) filepath.WalkFunc {
	return func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			if verbose {
				_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Error accessing %s: %v\n", path, err)
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		if shouldIgnoreFile(path, cfg) {
			if verbose {
				_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Ignoring %s\n", path)
			}
			return nil
		}

		return processGoFile(parser, path, cfg, allFunctions, verbose)
	}
}

// processGoFile parses a Go file and adds functions to the collection.
func processGoFile(
	parser *ast.Parser,
	path string,
	cfg *config.Config,
	allFunctions *[]*ast.Function,
	verbose bool,
) error {
	if verbose {
		_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Parsing file: %s\n", path)
	}

	functions, parseErr := parseGoFile(parser, path, cfg, verbose)
	if parseErr != nil {
		if verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[similarity-go] Error parsing %s: %v\n", path, parseErr)
		}
		return nil
	}

	*allFunctions = append(*allFunctions, functions...)
	return nil
}

// shouldIgnoreFile determines if a file should be ignored based on configuration.
func shouldIgnoreFile(filePath string, cfg *config.Config) bool {
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
	if cfg.Ignore.DefaultFile != "" {
		return matchesIgnorePatterns(filePath, cfg.GetIgnoreFilePath())
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
