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
		minLines:     5,
		maxFunctions: 1000, // Limit for performance
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
	t *testing.T,
	weights config.SimilarityWeights,
) ([]BenchmarkResult, error) {
	var results []BenchmarkResult

	for _, basePath := range cb.basePaths {
		result, err := cb.benchmarkSingleCodebase(t, basePath, weights)
		if err != nil {
			return nil, fmt.Errorf("failed to benchmark %s: %w", basePath, err)
		}
		results = append(results, *result)
	}

	return results, nil
}

// benchmarkSingleCodebase benchmarks weights against a single codebase.
func (cb *CodebaseBenchmark) benchmarkSingleCodebase(
	t *testing.T,
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
	detector := NewDetector(0.7) // Use 0.7 threshold for real-world analysis
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
			funcs, err := cb.extractFunctionsFromFile(path)
			if err != nil {
				// Log error but continue processing
				fmt.Printf("Warning: failed to parse %s: %v\n", path, err)
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
	for i := range len(internalFuncs) {
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
func (cb *CodebaseBenchmark) categorizePair(similarity float64, func1, func2 FunctionMetadata) string {
	if similarity >= 0.95 {
		return "near-duplicate"
	} else if similarity >= 0.85 {
		return "refactoring-candidate"
	} else if similarity >= 0.75 {
		return "similar-logic"
	} else {
		return "related-functionality"
	}
}

// calculateCodebaseStats computes comprehensive statistics.
func (cb *CodebaseBenchmark) calculateCodebaseStats(functions []FunctionMetadata, pairs []RealWorldPair) CodebaseStats {
	stats := CodebaseStats{
		FunctionSizeDistribution: make(map[string]int),
		ComplexityDistribution:   make(map[string]int),
		SimilarityDistribution:   make(map[string]int),
		PackageAnalysis:          make(map[string]PackageStats),
	}

	// Function size distribution
	for _, f := range functions {
		if f.LineCount < 10 {
			stats.FunctionSizeDistribution["Small"]++
		} else if f.LineCount < 30 {
			stats.FunctionSizeDistribution["Medium"]++
		} else if f.LineCount < 100 {
			stats.FunctionSizeDistribution["Large"]++
		} else {
			stats.FunctionSizeDistribution["XLarge"]++
		}
	}

	// Complexity distribution
	for _, f := range functions {
		if f.Complexity < 3 {
			stats.ComplexityDistribution["Low"]++
		} else if f.Complexity < 7 {
			stats.ComplexityDistribution["Medium"]++
		} else if f.Complexity < 15 {
			stats.ComplexityDistribution["High"]++
		} else {
			stats.ComplexityDistribution["Very High"]++
		}
	}

	// Similarity distribution
	for _, p := range pairs {
		if p.Similarity >= 0.9 {
			stats.SimilarityDistribution["Very High (0.9+)"]++
		} else if p.Similarity >= 0.8 {
			stats.SimilarityDistribution["High (0.8-0.9)"]++
		} else if p.Similarity >= 0.7 {
			stats.SimilarityDistribution["Medium (0.7-0.8)"]++
		} else {
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
		if len(topPairs) > 5 {
			topPairs = topPairs[:5]
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
		if pair.Similarity >= 0.8 {
			highSim++
		} else if pair.Similarity >= 0.6 {
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
		estimatedDuplication = float64(highSim) / float64(totalPairs) * 100
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
	functionCount, pairCount int,
) PerformanceMetrics {
	totalTime := time.Since(startTime)
	comparisons := functionCount * (functionCount - 1) / 2

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
func (cb *CodebaseBenchmark) PrintBenchmarkReport(results []BenchmarkResult) {
	fmt.Printf("\n=== CODEBASE BENCHMARK REPORT ===\n")

	for i, result := range results {
		if i > 0 {
			fmt.Printf("\n%s\n", strings.Repeat("=", 60))
		}

		fmt.Printf("Codebase: %s\n", result.CodebasePath)
		fmt.Printf("Files Analyzed: %d\n", result.TotalFiles)
		fmt.Printf("Functions Extracted: %d\n", result.TotalFunctions)
		fmt.Printf("Processing Time: %s\n", result.ProcessingTime)
		fmt.Printf("Similar Pairs Found: %d\n", len(result.SimilarityPairs))

		// Function size distribution
		fmt.Printf("\n--- FUNCTION SIZE DISTRIBUTION ---\n")
		for size, count := range result.Statistics.FunctionSizeDistribution {
			percentage := float64(count) / float64(result.TotalFunctions) * 100
			fmt.Printf("%-8s: %4d (%5.1f%%)\n", size, count, percentage)
		}

		// Complexity distribution
		fmt.Printf("\n--- COMPLEXITY DISTRIBUTION ---\n")
		for complexity, count := range result.Statistics.ComplexityDistribution {
			percentage := float64(count) / float64(result.TotalFunctions) * 100
			fmt.Printf("%-10s: %4d (%5.1f%%)\n", complexity, count, percentage)
		}

		// Similarity distribution
		if len(result.SimilarityPairs) > 0 {
			fmt.Printf("\n--- SIMILARITY DISTRIBUTION ---\n")
			for simRange, count := range result.Statistics.SimilarityDistribution {
				percentage := float64(count) / float64(len(result.SimilarityPairs)) * 100
				fmt.Printf("%-20s: %4d (%5.1f%%)\n", simRange, count, percentage)
			}
		}

		// Duplication metrics
		fmt.Printf("\n--- DUPLICATION ANALYSIS ---\n")
		dup := result.Statistics.DuplicationMetrics
		fmt.Printf("High Similarity Pairs (>0.8):  %d\n", dup.HighSimilarityCount)
		fmt.Printf("Medium Similarity Pairs (0.6-0.8): %d\n", dup.MediumSimilarityCount)
		fmt.Printf("Refactoring Candidates:         %d\n", dup.RefactoringCandidates)
		fmt.Printf("Estimated Duplication:          %.1f%%\n", dup.EstimatedDuplication)

		// Performance metrics
		fmt.Printf("\n--- PERFORMANCE METRICS ---\n")
		perf := result.PerformanceData
		fmt.Printf("Total Comparisons:      %d\n", perf.TotalComparisons)
		fmt.Printf("Avg Comparison Time:    %s\n", perf.AvgComparisonTime)
		fmt.Printf("Functions per Second:    %.1f\n",
			float64(result.TotalFunctions)/result.ProcessingTime.Seconds())

		// Top similar pairs
		fmt.Printf("\n--- TOP 10 SIMILAR PAIRS ---\n")
		topPairs := result.SimilarityPairs
		if len(topPairs) > 10 {
			topPairs = topPairs[:10]
		}

		for j, pair := range topPairs {
			fmt.Printf("%2d. %.3f %s:%s ↔ %s:%s (%s)\n",
				j+1, pair.Similarity,
				filepath.Base(pair.Function1.File), pair.Function1.Name,
				filepath.Base(pair.Function2.File), pair.Function2.Name,
				pair.Category)
		}

		// Package analysis summary
		fmt.Printf("\n--- TOP PACKAGES BY DUPLICATION ---\n")
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
			if j >= 5 { // Show top 5
				break
			}
			fmt.Printf("%d. %-20s: %.3f (%d functions)\n",
				j+1, pkg.name, pkg.ratio, pkg.count)
		}
	}

	// Overall summary
	if len(results) > 1 {
		fmt.Printf("\n=== OVERALL SUMMARY ===\n")
		var totalFiles, totalFunctions, totalPairs int
		var totalTime time.Duration

		for _, result := range results {
			totalFiles += result.TotalFiles
			totalFunctions += result.TotalFunctions
			totalPairs += len(result.SimilarityPairs)
			totalTime += result.ProcessingTime
		}

		fmt.Printf("Total Codebases:   %d\n", len(results))
		fmt.Printf("Total Files:       %d\n", totalFiles)
		fmt.Printf("Total Functions:   %d\n", totalFunctions)
		fmt.Printf("Total Pairs:       %d\n", totalPairs)
		fmt.Printf("Total Time:        %s\n", totalTime)
		fmt.Printf("Avg Duplication:   %.1f%%\n",
			float64(totalPairs)/float64(totalFunctions)*100)
	}
}
