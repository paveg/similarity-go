package similarity

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	astpkg "github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/config"
)

const (
	percentage100               = 100.0
	topPairsDisplayLimit        = 10
	topPackageDisplayLimit      = 5
	topPairsPerPackage          = 5
	defaultMinLines             = 5
	defaultMaxFunctions         = 1000
	detectorThreshold           = 0.7
	similarityNearDuplicate     = 0.95
	similarityRefactorCandidate = 0.85
	similaritySimilarLogic      = 0.75
	similarityVeryHigh          = 0.9
	similarityHigh              = 0.8
	similarityMedium            = 0.7
	similarityLow               = 0.6
	reportSeparatorLength       = 60
	pairsDivisor                = 2
	lineCountSmall              = 10
	lineCountMedium             = 30
	lineCountLarge              = 100
	complexityLowThreshold      = 3
	complexityMediumThreshold   = 7
	complexityHighThreshold     = 15
)

// CodebaseBenchmark provides benchmarking against real Go codebases.
type CodebaseBenchmark struct {
	basePaths    []string
	excludePaths []string
	minLines     int
	maxFunctions int
	fileSet      *token.FileSet
}

// BenchmarkResult contains results from codebase benchmarking.
type BenchmarkResult struct {
	CodebasePath    string
	TotalFiles      int
	TotalFunctions  int
	ProcessingTime  time.Duration
	SimilarityPairs []RealWorldPair
	Statistics      CodebaseStats
	PerformanceData PerformanceMetrics
}

// RealWorldPair represents a pair of similar functions found in real code.
type RealWorldPair struct {
	Function1    FunctionMetadata
	Function2    FunctionMetadata
	Similarity   float64
	Components   SimilarityComponents
	ManualRating float64 // Manual assessment if available
	Category     string  // auto-generated, refactoring-candidate, etc.
}

// FunctionMetadata contains metadata about a real function.
type FunctionMetadata struct {
	Name       string
	File       string
	Package    string
	StartLine  int
	EndLine    int
	LineCount  int
	Complexity int // Cyclomatic complexity estimate
	Signature  string
	AST        *ast.FuncDecl
}

// SimilarityComponents breaks down similarity calculation.
//
//nolint:revive // exported name communicates intent despite stutter
type SimilarityComponents struct {
	TreeEdit        float64
	TokenSimilarity float64
	Structural      float64
	Signature       float64
	WeightedScore   float64
}

// CodebaseStats contains statistical analysis of the codebase.
type CodebaseStats struct {
	FunctionSizeDistribution map[string]int // Small, Medium, Large, XLarge
	ComplexityDistribution   map[string]int // Low, Medium, High, Very High
	SimilarityDistribution   map[string]int // Ranges of similarity scores
	PackageAnalysis          map[string]PackageStats
	DuplicationMetrics       DuplicationMetrics
}

// PackageStats contains per-package analysis.
type PackageStats struct {
	PackageName      string
	FunctionCount    int
	AvgComplexity    float64
	AvgSimilarity    float64
	DuplicationRatio float64
	TopSimilarPairs  []RealWorldPair
}

// DuplicationMetrics contains code duplication analysis.
type DuplicationMetrics struct {
	HighSimilarityCount   int     // > 0.8
	MediumSimilarityCount int     // 0.6 - 0.8
	EstimatedDuplication  float64 // % of code that's duplicated
	RefactoringCandidates int     // Functions that could be refactored
}

// PerformanceMetrics contains performance analysis data.
type PerformanceMetrics struct {
	AvgComparisonTime  time.Duration
	TotalComparisons   int
	MemoryUsage        int64
	CacheHitRate       float64
	ParallelEfficiency float64
}

// NewCodebaseBenchmark creates a new codebase benchmark.
func NewCodebaseBenchmark(basePaths []string) *CodebaseBenchmark {
	return &CodebaseBenchmark{
		basePaths: basePaths,
		excludePaths: []string{
			"vendor/",
			".git/",
			"testdata/",
			"_test.go",
			".pb.go",
			"mock_",
			"generated",
		},
		minLines:     defaultMinLines,
		maxFunctions: defaultMaxFunctions,
		fileSet:      token.NewFileSet(),
	}
}

// SetParameters configures benchmark parameters.
func (cb *CodebaseBenchmark) SetParameters(minLines, maxFunctions int, excludePaths []string) {
	cb.minLines = minLines
	cb.maxFunctions = maxFunctions
	if excludePaths != nil {
		cb.excludePaths = append(cb.excludePaths, excludePaths...)
	}
}

// BenchmarkWeights benchmarks weights against real codebases.
func (cb *CodebaseBenchmark) BenchmarkWeights(
	_ *testing.T,
	weights config.SimilarityWeights,
) ([]BenchmarkResult, error) {
	var results []BenchmarkResult

	for _, basePath := range cb.basePaths {
		result, err := cb.benchmarkSingleCodebase(basePath, weights)
		if err != nil {
			return nil, fmt.Errorf("failed to benchmark %s: %w", basePath, err)
		}
		results = append(results, *result)
	}

	return results, nil
}

// benchmarkSingleCodebase benchmarks weights against a single codebase.
func (cb *CodebaseBenchmark) benchmarkSingleCodebase(
	basePath string,
	weights config.SimilarityWeights,
) (*BenchmarkResult, error) {
	startTime := time.Now()

	// Extract functions from codebase
	functions, err := cb.extractFunctions(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract functions: %w", err)
	}

	// Limit functions for performance
	if len(functions) > cb.maxFunctions {
		functions = functions[:cb.maxFunctions]
	}

	// Calculate similarity pairs
	detector := NewDetector(detectorThreshold) // Use 0.7 threshold for real-world analysis
	cfg := config.Default()
	cfg.Similarity.Weights = weights
	detector.config = cfg

	similarityPairs := cb.findSimilarityPairs(functions, detector)

	// Calculate statistics
	stats := cb.calculateCodebaseStats(functions, similarityPairs)

	// Calculate performance metrics
	perfMetrics := cb.calculatePerformanceMetrics(startTime, len(functions), len(similarityPairs))

	result := &BenchmarkResult{
		CodebasePath:    basePath,
		TotalFiles:      cb.countFiles(basePath),
		TotalFunctions:  len(functions),
		ProcessingTime:  time.Since(startTime),
		SimilarityPairs: similarityPairs,
		Statistics:      stats,
		PerformanceData: perfMetrics,
	}

	return result, nil
}

// extractFunctions extracts all functions from a codebase.
func (cb *CodebaseBenchmark) extractFunctions(basePath string) ([]FunctionMetadata, error) {
	var functions []FunctionMetadata

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded paths
		for _, exclude := range cb.excludePaths {
			if strings.Contains(path, exclude) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Process Go files
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			funcs, fileErr := cb.extractFunctionsFromFile(path)
			if fileErr != nil {
				// Log error but continue processing
				_, _ = fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, fileErr)
				return nil
			}
			functions = append(functions, funcs...)
		}

		return nil
	})

	return functions, err
}

// extractFunctionsFromFile extracts functions from a single file.
func (cb *CodebaseBenchmark) extractFunctionsFromFile(filePath string) ([]FunctionMetadata, error) {
	var functions []FunctionMetadata

	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	file, err := parser.ParseFile(cb.fileSet, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Extract package name
	packageName := file.Name.Name

	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Body != nil { // Skip function declarations without body
				startPos := cb.fileSet.Position(funcDecl.Pos())
				endPos := cb.fileSet.Position(funcDecl.End())
				lineCount := endPos.Line - startPos.Line + 1

				if lineCount >= cb.minLines {
					function := FunctionMetadata{
						Name:       funcDecl.Name.Name,
						File:       filePath,
						Package:    packageName,
						StartLine:  startPos.Line,
						EndLine:    endPos.Line,
						LineCount:  lineCount,
						Complexity: cb.estimateComplexity(funcDecl),
						Signature:  cb.extractSignature(funcDecl),
						AST:        funcDecl,
					}
					functions = append(functions, function)
				}
			}
		}
		return true
	})

	return functions, nil
}

// estimateComplexity provides a rough cyclomatic complexity estimate.
func (cb *CodebaseBenchmark) estimateComplexity(funcDecl *ast.FuncDecl) int {
	complexity := 1 // Base complexity

	ast.Inspect(funcDecl, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})

	return complexity
}

// extractSignature creates a normalized function signature.
func (cb *CodebaseBenchmark) extractSignature(funcDecl *ast.FuncDecl) string {
	var parts []string

	// Add function name
	parts = append(parts, funcDecl.Name.Name)

	// Add parameter types (simplified)
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			parts = append(parts, fmt.Sprintf("%T", param.Type))
		}
	}

	// Add return types (simplified)
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			parts = append(parts, fmt.Sprintf("%T", result.Type))
		}
	}

	return strings.Join(parts, "_")
}

// findSimilarityPairs finds similar function pairs using the detector.
func (cb *CodebaseBenchmark) findSimilarityPairs(functions []FunctionMetadata, detector *Detector) []RealWorldPair {
	var pairs []RealWorldPair

	// Convert to internal function format
	internalFuncs := make([]*astpkg.Function, len(functions))
	for i, f := range functions {
		internalFuncs[i] = &astpkg.Function{
			Name:      f.Name,
			File:      f.File,
			StartLine: f.StartLine,
			EndLine:   f.EndLine,
			AST:       f.AST,
			LineCount: f.LineCount,
		}
	}

	// Find pairs above threshold
	for i := range internalFuncs {
		for j := i + 1; j < len(internalFuncs); j++ {
			func1, func2 := internalFuncs[i], internalFuncs[j]

			// Skip functions from the same file to focus on cross-file duplication
			if func1.File == func2.File {
				continue
			}

			similarity := detector.CalculateSimilarity(func1, func2)

			if similarity >= detector.threshold {
				// Calculate component breakdown
				components := cb.calculateComponents(func1, func2, detector)

				pair := RealWorldPair{
					Function1:  functions[i],
					Function2:  functions[j],
					Similarity: similarity,
					Components: components,
					Category:   cb.categorizePair(similarity, functions[i], functions[j]),
				}

				pairs = append(pairs, pair)
			}
		}
	}

	// Sort by similarity (descending)
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Similarity > pairs[j].Similarity
	})

	return pairs
}

// calculateComponents calculates individual similarity components.
func (cb *CodebaseBenchmark) calculateComponents(
	func1, func2 *astpkg.Function,
	detector *Detector,
) SimilarityComponents {
	// This is a simplified version - in reality you'd need access to detector internals
	// For now, we'll estimate based on the final similarity score
	totalSim := detector.CalculateSimilarity(func1, func2)

	weights := detector.config.Similarity.Weights

	// Rough estimation - in practice you'd calculate each component separately
	components := SimilarityComponents{
		TreeEdit:        totalSim * weights.TreeEdit,
		TokenSimilarity: totalSim * weights.TokenSimilarity,
		Structural:      totalSim * weights.Structural,
		Signature:       totalSim * weights.Signature,
		WeightedScore:   totalSim,
	}

	return components
}

// categorizePair categorizes a similarity pair.
func (cb *CodebaseBenchmark) categorizePair(similarity float64, _ FunctionMetadata, _ FunctionMetadata) string {
	switch {
	case similarity >= similarityNearDuplicate:
		return "near-duplicate"
	case similarity >= similarityRefactorCandidate:
		return "refactoring-candidate"
	case similarity >= similaritySimilarLogic:
		return "similar-logic"
	default:
		return "related-functionality"
	}
}

// calculateCodebaseStats computes comprehensive statistics.
//
//nolint:gocognit,funlen // statistical aggregation requires iterative categorization
func (cb *CodebaseBenchmark) calculateCodebaseStats(functions []FunctionMetadata, pairs []RealWorldPair) CodebaseStats {
	stats := CodebaseStats{
		FunctionSizeDistribution: make(map[string]int),
		ComplexityDistribution:   make(map[string]int),
		SimilarityDistribution:   make(map[string]int),
		PackageAnalysis:          make(map[string]PackageStats),
	}

	// Function size distribution
	for _, f := range functions {
		switch {
		case f.LineCount < lineCountSmall:
			stats.FunctionSizeDistribution["Small"]++
		case f.LineCount < lineCountMedium:
			stats.FunctionSizeDistribution["Medium"]++
		case f.LineCount < lineCountLarge:
			stats.FunctionSizeDistribution["Large"]++
		default:
			stats.FunctionSizeDistribution["XLarge"]++
		}
	}

	// Complexity distribution
	for _, f := range functions {
		switch {
		case f.Complexity < complexityLowThreshold:
			stats.ComplexityDistribution["Low"]++
		case f.Complexity < complexityMediumThreshold:
			stats.ComplexityDistribution["Medium"]++
		case f.Complexity < complexityHighThreshold:
			stats.ComplexityDistribution["High"]++
		default:
			stats.ComplexityDistribution["Very High"]++
		}
	}

	// Similarity distribution
	for _, p := range pairs {
		switch {
		case p.Similarity >= similarityVeryHigh:
			stats.SimilarityDistribution["Very High (0.9+)"]++
		case p.Similarity >= similarityHigh:
			stats.SimilarityDistribution["High (0.8-0.9)"]++
		case p.Similarity >= similarityMedium:
			stats.SimilarityDistribution["Medium (0.7-0.8)"]++
		case p.Similarity >= config.MinSimilarity:
			stats.SimilarityDistribution["Low (0.6-0.7)"]++
		}
	}

	// Package analysis
	packageFuncs := make(map[string][]FunctionMetadata)
	for _, f := range functions {
		packageFuncs[f.Package] = append(packageFuncs[f.Package], f)
	}

	for pkg, pkgFuncs := range packageFuncs {
		var complexitySum int
		for _, f := range pkgFuncs {
			complexitySum += f.Complexity
		}

		// Find pairs involving this package
		var packagePairs []RealWorldPair
		var similaritySum float64
		for _, pair := range pairs {
			if pair.Function1.Package == pkg || pair.Function2.Package == pkg {
				packagePairs = append(packagePairs, pair)
				similaritySum += pair.Similarity
			}
		}

		avgSimilarity := 0.0
		if len(packagePairs) > 0 {
			avgSimilarity = similaritySum / float64(len(packagePairs))
		}

		// Sort package pairs by similarity
		sort.Slice(packagePairs, func(i, j int) bool {
			return packagePairs[i].Similarity > packagePairs[j].Similarity
		})

		topPairs := packagePairs
		if len(topPairs) > topPairsPerPackage {
			topPairs = topPairs[:topPairsPerPackage]
		}

		stats.PackageAnalysis[pkg] = PackageStats{
			PackageName:      pkg,
			FunctionCount:    len(pkgFuncs),
			AvgComplexity:    float64(complexitySum) / float64(len(pkgFuncs)),
			AvgSimilarity:    avgSimilarity,
			DuplicationRatio: float64(len(packagePairs)) / float64(len(pkgFuncs)),
			TopSimilarPairs:  topPairs,
		}
	}

	// Duplication metrics
	stats.DuplicationMetrics = cb.calculateDuplicationMetrics(pairs)

	return stats
}

// calculateDuplicationMetrics calculates code duplication metrics.
func (cb *CodebaseBenchmark) calculateDuplicationMetrics(pairs []RealWorldPair) DuplicationMetrics {
	var highSim, mediumSim, refactoringCandidates int

	for _, pair := range pairs {
		if pair.Similarity >= similarityHigh {
			highSim++
		} else if pair.Similarity >= similarityLow {
			mediumSim++
		}

		if pair.Category == "refactoring-candidate" || pair.Category == "near-duplicate" {
			refactoringCandidates++
		}
	}

	// Estimate duplication percentage (rough heuristic)
	totalPairs := len(pairs)
	estimatedDuplication := 0.0
	if totalPairs > 0 {
		estimatedDuplication = float64(highSim) / float64(totalPairs) * percentage100
	}

	return DuplicationMetrics{
		HighSimilarityCount:   highSim,
		MediumSimilarityCount: mediumSim,
		EstimatedDuplication:  estimatedDuplication,
		RefactoringCandidates: refactoringCandidates,
	}
}

// calculatePerformanceMetrics calculates performance-related metrics.
func (cb *CodebaseBenchmark) calculatePerformanceMetrics(
	startTime time.Time,
	functionCount, _ int,
) PerformanceMetrics {
	totalTime := time.Since(startTime)
	comparisons := functionCount * (functionCount - 1) / pairsDivisor

	avgComparisonTime := time.Duration(0)
	if comparisons > 0 {
		avgComparisonTime = totalTime / time.Duration(comparisons)
	}

	return PerformanceMetrics{
		AvgComparisonTime:  avgComparisonTime,
		TotalComparisons:   comparisons,
		MemoryUsage:        0,   // Would need runtime.ReadMemStats() for actual measurement
		CacheHitRate:       0.0, // Would need cache instrumentation
		ParallelEfficiency: 1.0, // Simplified for now
	}
}

// countFiles counts the number of Go files in a directory.
func (cb *CodebaseBenchmark) countFiles(basePath string) int {
	count := 0
	_ = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip excluded paths
		for _, exclude := range cb.excludePaths {
			if strings.Contains(path, exclude) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			count++
		}

		return nil
	})

	return count
}

// PrintBenchmarkReport prints a comprehensive benchmark report.
//
//nolint:gocognit,funlen // report formatting naturally requires branching for sections
func (cb *CodebaseBenchmark) PrintBenchmarkReport(results []BenchmarkResult) {
	out := os.Stdout
	write := func(format string, args ...any) {
		_, _ = fmt.Fprintf(out, format, args...)
	}

	write("\n=== CODEBASE BENCHMARK REPORT ===\n")

	for i, result := range results {
		if i > 0 {
			write("\n%s\n", strings.Repeat("=", reportSeparatorLength))
		}

		write("Codebase: %s\n", result.CodebasePath)
		write("Files Analyzed: %d\n", result.TotalFiles)
		write("Functions Extracted: %d\n", result.TotalFunctions)
		write("Processing Time: %s\n", result.ProcessingTime)
		write("Similar Pairs Found: %d\n", len(result.SimilarityPairs))

		// Function size distribution
		write("\n--- FUNCTION SIZE DISTRIBUTION ---\n")
		for size, count := range result.Statistics.FunctionSizeDistribution {
			percentage := float64(count) / float64(result.TotalFunctions) * percentage100
			write("%-8s: %4d (%5.1f%%)\n", size, count, percentage)
		}

		// Complexity distribution
		write("\n--- COMPLEXITY DISTRIBUTION ---\n")
		for complexity, count := range result.Statistics.ComplexityDistribution {
			percentage := float64(count) / float64(result.TotalFunctions) * percentage100
			write("%-10s: %4d (%5.1f%%)\n", complexity, count, percentage)
		}

		// Similarity distribution
		if len(result.SimilarityPairs) > 0 {
			write("\n--- SIMILARITY DISTRIBUTION ---\n")
			for simRange, count := range result.Statistics.SimilarityDistribution {
				percentage := float64(count) / float64(len(result.SimilarityPairs)) * percentage100
				write("%-20s: %4d (%5.1f%%)\n", simRange, count, percentage)
			}
		}

		// Duplication metrics
		write("\n--- DUPLICATION ANALYSIS ---\n")
		dup := result.Statistics.DuplicationMetrics
		write("High Similarity Pairs (>0.8):  %d\n", dup.HighSimilarityCount)
		write("Medium Similarity Pairs (0.6-0.8): %d\n", dup.MediumSimilarityCount)
		write("Refactoring Candidates:         %d\n", dup.RefactoringCandidates)
		write("Estimated Duplication:          %.1f%%\n", dup.EstimatedDuplication)

		// Performance metrics
		write("\n--- PERFORMANCE METRICS ---\n")
		perf := result.PerformanceData
		write("Total Comparisons:      %d\n", perf.TotalComparisons)
		write("Avg Comparison Time:    %s\n", perf.AvgComparisonTime)
		write("Functions per Second:    %.1f\n",
			float64(result.TotalFunctions)/result.ProcessingTime.Seconds())

		// Top similar pairs
		write("\n--- TOP 10 SIMILAR PAIRS ---\n")
		topPairs := result.SimilarityPairs
		if len(topPairs) > topPairsDisplayLimit {
			topPairs = topPairs[:topPairsDisplayLimit]
		}

		for j, pair := range topPairs {
			write("%2d. %.3f %s:%s ↔ %s:%s (%s)\n",
				j+1, pair.Similarity,
				filepath.Base(pair.Function1.File), pair.Function1.Name,
				filepath.Base(pair.Function2.File), pair.Function2.Name,
				pair.Category)
		}

		// Package analysis summary
		write("\n--- TOP PACKAGES BY DUPLICATION ---\n")
		type pkgDup struct {
			name  string
			ratio float64
			count int
		}

		var packageDups []pkgDup
		for name, stats := range result.Statistics.PackageAnalysis {
			packageDups = append(packageDups, pkgDup{
				name:  name,
				ratio: stats.DuplicationRatio,
				count: stats.FunctionCount,
			})
		}

		sort.Slice(packageDups, func(i, j int) bool {
			return packageDups[i].ratio > packageDups[j].ratio
		})

		for j, pkg := range packageDups {
			if j >= topPackageDisplayLimit {
				break
			}
			write("%d. %-20s: %.3f (%d functions)\n",
				j+1, pkg.name, pkg.ratio, pkg.count)
		}
	}

	// Overall summary
	if len(results) > 1 {
		write("\n=== OVERALL SUMMARY ===\n")
		var totalFiles, totalFunctions, totalPairs int
		var totalTime time.Duration

		for _, result := range results {
			totalFiles += result.TotalFiles
			totalFunctions += result.TotalFunctions
			totalPairs += len(result.SimilarityPairs)
			totalTime += result.ProcessingTime
		}

		write("Total Codebases:   %d\n", len(results))
		write("Total Files:       %d\n", totalFiles)
		write("Total Functions:   %d\n", totalFunctions)
		write("Total Pairs:       %d\n", totalPairs)
		write("Total Time:        %s\n", totalTime)
		write("Avg Duplication:   %.1f%%\n",
			float64(totalPairs)/float64(totalFunctions)*percentage100)
	}
}
