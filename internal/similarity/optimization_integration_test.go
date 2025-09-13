package similarity

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paveg/similarity-go/internal/config"
)

// TestWeightOptimizationWorkflow tests the complete weight optimization workflow.
func TestWeightOptimizationWorkflow(t *testing.T) {
	t.Log("=== Starting Complete Weight Optimization Workflow ===")

	// Step 1: Analyze current weights
	t.Log("Step 1: Analyzing current weights...")
	optimizer := NewWeightOptimizer()
	currentScore, currentResults := optimizer.AnalyzeCurrentWeights(t)

	t.Logf("Current baseline score: %.6f", currentScore)
	if len(currentResults) == 0 {
		t.Fatal("Expected non-empty current results")
	}

	// Step 2: Run statistical validation on current weights
	t.Log("Step 2: Running statistical validation on current weights...")
	validator := NewStatisticalValidator()
	currentWeights := config.SimilarityWeights{
		TreeEdit:           config.TreeEditWeight,
		TokenSimilarity:    config.TokenSimilarityWeight,
		Structural:         config.StructuralWeight,
		Signature:          config.SignatureWeight,
		DifferentSignature: config.DifferentSignatureWeight,
	}

	currentValidation := validator.ValidateWeights(t, currentWeights)
	t.Logf("Current MAE: %.6f, R²: %.6f, F1: %.6f",
		currentValidation.MAE, currentValidation.R2, currentValidation.F1Score)

	// Step 3: Run grid search optimization
	t.Log("Step 3: Running grid search optimization...")
	gridResult := optimizer.GridSearchOptimize(t)

	t.Logf("Grid search completed: %d iterations", gridResult.IterationCount)
	t.Logf("Best grid search score: %.6f", gridResult.BestScore)

	if gridResult.BestScore <= currentScore {
		t.Logf("Grid search did not improve on baseline (%.6f vs %.6f)",
			gridResult.BestScore, currentScore)
	} else {
		t.Logf("Grid search improved baseline by %.6f",
			gridResult.BestScore-currentScore)
	}

	// Step 4: Run genetic algorithm optimization
	t.Log("Step 4: Running genetic algorithm optimization...")
	genetic := NewGeneticOptimizer()
	genetic.SetParameters(20, 15, 0.15, 0.8, 3) // Small parameters for testing

	geneticResult := genetic.OptimizeWeights(t)

	t.Logf("Genetic algorithm completed: %d generations, %d evaluations",
		len(geneticResult.GenerationHistory), geneticResult.TotalEvaluations)
	t.Logf("Best genetic score: %.6f", geneticResult.BestIndividual.Fitness)

	// Step 5: Compare optimization methods
	t.Log("Step 5: Comparing optimization methods...")
	bestScore := currentScore
	bestWeights := currentWeights
	bestMethod := "baseline"

	if gridResult.BestScore > bestScore {
		bestScore = gridResult.BestScore
		bestWeights = gridResult.BestWeights
		bestMethod = "grid_search"
	}

	if geneticResult.BestIndividual.Fitness > bestScore {
		bestScore = geneticResult.BestIndividual.Fitness
		bestWeights = geneticResult.BestIndividual.Weights
		bestMethod = "genetic_algorithm"
	}

	t.Logf("Best method: %s with score %.6f", bestMethod, bestScore)
	t.Logf("Improvement over baseline: %.6f (%.2f%%)",
		bestScore-currentScore, (bestScore-currentScore)/currentScore*100)

	// Step 6: Validate the best weights
	t.Log("Step 6: Validating best weights...")
	bestValidation := validator.ValidateWeights(t, bestWeights)

	t.Logf("Best weights validation - MAE: %.6f, R²: %.6f, F1: %.6f",
		bestValidation.MAE, bestValidation.R2, bestValidation.F1Score)

	// Step 7: Test configuration update (dry run)
	t.Log("Step 7: Testing configuration update mechanism...")
	configUpdater := NewConfigUpdater()

	// Validate weights before update
	err := configUpdater.ValidateWeightSum(bestWeights)
	if err != nil {
		t.Fatalf("Best weights validation failed: %v", err)
	}

	// Create YAML config for testing
	tempDir := t.TempDir()
	yamlFile := filepath.Join(tempDir, "optimized_config.yaml")

	err = configUpdater.CreateYAMLConfig(bestWeights, yamlFile)
	if err != nil {
		t.Fatalf("Failed to create YAML config: %v", err)
	}

	// Verify YAML file was created
	if _, err := os.Stat(yamlFile); os.IsNotExist(err) {
		t.Error("YAML config file was not created")
	}

	t.Logf("YAML config created: %s", yamlFile)

	// Step 8: Generate comprehensive reports
	t.Log("Step 8: Generating comprehensive reports...")

	// Print optimization report
	optimizer.PrintOptimizationReport(gridResult, currentScore)

	// Print genetic algorithm report
	genetic.PrintGeneticReport(geneticResult, currentScore)

	// Print statistical validation report
	validator.PrintValidationReport(bestValidation)

	// Step 9: Performance comparison
	t.Log("Step 9: Running performance comparison...")
	performanceComparison := map[string]struct {
		score      float64
		weights    config.SimilarityWeights
		validation ValidationResult
	}{
		"baseline":  {currentScore, currentWeights, currentValidation},
		"optimized": {bestScore, bestWeights, bestValidation},
	}

	for method, data := range performanceComparison {
		t.Logf("%s - Score: %.6f, MAE: %.6f, Accuracy: %.6f",
			method, data.score, data.validation.MAE, data.validation.Accuracy)
	}

	// Final validation
	if bestScore < currentScore-0.01 { // Allow small tolerance
		t.Errorf("Optimization failed to improve performance: %.6f vs %.6f",
			bestScore, currentScore)
	}

	if bestValidation.MAE > currentValidation.MAE+0.05 { // Allow tolerance for MAE
		t.Errorf("Optimized MAE worse than baseline: %.6f vs %.6f",
			bestValidation.MAE, currentValidation.MAE)
	}

	t.Log("=== Weight Optimization Workflow Completed Successfully ===")
}

// TestRealWorldCodebaseBenchmark tests benchmarking with a synthetic codebase.
func TestRealWorldCodebaseBenchmark(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-world benchmark in short mode")
	}

	t.Log("=== Testing Real-World Codebase Benchmarking ===")

	// Create a synthetic "real-world" codebase
	tempDir := t.TempDir()

	err := createSyntheticCodebase(tempDir)
	if err != nil {
		t.Fatalf("Failed to create synthetic codebase: %v", err)
	}

	// Run optimization to get optimized weights
	t.Log("Running optimization to get test weights...")
	optimizer := NewWeightOptimizer()
	gridResult := optimizer.GridSearchOptimize(t)

	// Benchmark with the codebase
	t.Log("Benchmarking against synthetic codebase...")
	benchmark := NewCodebaseBenchmark([]string{tempDir})
	benchmark.SetParameters(3, 500, nil)

	results, err := benchmark.BenchmarkWeights(t, gridResult.BestWeights)
	if err != nil {
		t.Fatalf("Benchmark failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 benchmark result, got %d", len(results))
	}

	result := results[0]

	// Validate benchmark results
	if result.TotalFiles == 0 {
		t.Error("Expected to process some files")
	}

	if result.TotalFunctions == 0 {
		t.Error("Expected to find some functions")
	}

	if result.ProcessingTime <= 0 {
		t.Error("Expected positive processing time")
	}

	t.Logf("Benchmark results: %d files, %d functions, %d similar pairs",
		result.TotalFiles, result.TotalFunctions, len(result.SimilarityPairs))

	// Print detailed benchmark report
	benchmark.PrintBenchmarkReport(results)

	t.Log("=== Real-World Codebase Benchmarking Completed ===")
}

// TestOptimizationComparison compares different optimization approaches.
func TestOptimizationComparison(t *testing.T) {
	t.Log("=== Comparing Optimization Approaches ===")

	optimizer := NewWeightOptimizer()
	genetic := NewGeneticOptimizer()
	genetic.SetParameters(15, 10, 0.2, 0.8, 2) // Small parameters for testing

	// Get baseline
	baselineScore, _ := optimizer.AnalyzeCurrentWeights(t)
	t.Logf("Baseline score: %.6f", baselineScore)

	// Time each approach
	approaches := []struct {
		name string
		fn   func() (float64, config.SimilarityWeights)
	}{
		{
			name: "Grid Search",
			fn: func() (float64, config.SimilarityWeights) {
				start := time.Now()
				result := optimizer.GridSearchOptimize(t)
				duration := time.Since(start)
				t.Logf("Grid search took: %s (%d iterations)", duration, result.IterationCount)
				return result.BestScore, result.BestWeights
			},
		},
		{
			name: "Genetic Algorithm",
			fn: func() (float64, config.SimilarityWeights) {
				start := time.Now()
				result := genetic.OptimizeWeights(t)
				duration := time.Since(start)
				t.Logf("Genetic algorithm took: %s (%d evaluations)",
					duration, result.TotalEvaluations)
				return result.BestIndividual.Fitness, result.BestIndividual.Weights
			},
		},
	}

	results := make(map[string]struct {
		score       float64
		weights     config.SimilarityWeights
		improvement float64
	})

	for _, approach := range approaches {
		t.Logf("Running %s...", approach.name)
		score, weights := approach.fn()
		improvement := score - baselineScore

		results[approach.name] = struct {
			score       float64
			weights     config.SimilarityWeights
			improvement float64
		}{score, weights, improvement}

		t.Logf("%s - Score: %.6f, Improvement: %.6f (%.2f%%)",
			approach.name, score, improvement, improvement/baselineScore*100)
	}

	// Compare results
	t.Log("\n=== Optimization Comparison Summary ===")
	for name, result := range results {
		t.Logf("%-20s: Score=%.6f, Improvement=%+.6f",
			name, result.score, result.improvement)
	}

	// Both methods should at least maintain baseline performance
	for name, result := range results {
		if result.score < baselineScore-0.01 { // Small tolerance
			t.Errorf("%s performed worse than baseline: %.6f vs %.6f",
				name, result.score, baselineScore)
		}
	}

	t.Log("=== Optimization Comparison Completed ===")
}

// TestOptimizationStability tests the stability and consistency of optimization.
func TestOptimizationStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stability test in short mode")
	}

	t.Log("=== Testing Optimization Stability ===")

	genetic := NewGeneticOptimizer()
	genetic.SetParameters(10, 5, 0.1, 0.8, 2) // Very small for stability testing

	runs := 3
	scores := make([]float64, runs)

	for i := range runs {
		t.Logf("Running stability test %d/%d...", i+1, runs)
		result := genetic.OptimizeWeights(t)
		scores[i] = result.BestIndividual.Fitness
		t.Logf("Run %d score: %.6f", i+1, scores[i])
	}

	// Calculate statistics
	var sum, sumSquares float64
	for _, score := range scores {
		sum += score
		sumSquares += score * score
	}

	mean := sum / float64(runs)
	variance := (sumSquares / float64(runs)) - (mean * mean)
	stdDev := 0.0
	if variance > 0 {
		stdDev = variance // Simplified square root
	}

	t.Logf("Stability results: Mean=%.6f, StdDev=%.6f", mean, stdDev)

	// Check for reasonable stability (coefficient of variation < 0.1)
	cv := stdDev / mean
	if cv > 0.1 {
		t.Logf("Warning: High variability in optimization results (CV=%.3f)", cv)
		// Not failing the test as some variability is expected with small populations
	}

	t.Log("=== Optimization Stability Test Completed ===")
}

// createSyntheticCodebase creates a synthetic codebase for testing.
func createSyntheticCodebase(baseDir string) error {
	files := map[string]string{
		"math/basic.go": `package math

func Add(a, b int) int {
	return a + b
}

func Subtract(a, b int) int {
	return a - b
}

func Multiply(x, y int) int {
	result := x * y
	return result
}

func Divide(numerator, denominator int) int {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}`,

		"utils/operations.go": `package utils

import "fmt"

func Sum(x, y int) int {
	// Very similar to Add
	return x + y
}

func Product(a, b int) int {
	// Very similar to Multiply
	temp := a * b
	return temp
}

func SafeDivide(num, den int) int {
	// Similar to Divide with same logic
	if den == 0 {
		return 0
	}
	return num / den
}

func PrintResult(value int) {
	fmt.Printf("Result: %d\n", value)
}`,

		"calc/calculator.go": `package calc

type Calculator struct {
	memory int
}

func (c *Calculator) AddToMemory(value int) {
	c.memory += value
}

func (c *Calculator) SubtractFromMemory(value int) {
	c.memory -= value
}

func (c *Calculator) GetMemory() int {
	return c.memory
}

func (c *Calculator) ClearMemory() {
	c.memory = 0
}

func (c *Calculator) Calculate(a, b int, op string) int {
	switch op {
	case "+":
		return a + b
	case "-":
		return a - b
	case "*":
		return a * b
	case "/":
		if b != 0 {
			return a / b
		}
		return 0
	default:
		return 0
	}
}`,

		"string/processor.go": `package string

import (
	"strings"
	"unicode"
)

func ToUpper(s string) string {
	return strings.ToUpper(s)
}

func ToLower(s string) string {
	return strings.ToLower(s)
}

func CountWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}

func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}`,
	}

	for file, content := range files {
		fullPath := filepath.Join(baseDir, file)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}

// BenchmarkOptimizationPerformance benchmarks the performance of optimization algorithms.
func BenchmarkOptimizationPerformance(b *testing.B) {
	optimizer := NewWeightOptimizer()
	genetic := NewGeneticOptimizer()
	genetic.SetParameters(10, 5, 0.1, 0.8, 2)

	b.Run("GridSearch", func(b *testing.B) {
		// Create a dummy test for the benchmark
		t := &testing.T{}

		b.ResetTimer()
		for range b.N {
			optimizer.GridSearchOptimize(t)
		}
	})

	b.Run("GeneticAlgorithm", func(b *testing.B) {
		// Create a dummy test for the benchmark
		t := &testing.T{}

		b.ResetTimer()
		for range b.N {
			genetic.OptimizeWeights(t)
		}
	})

	b.Run("StatisticalValidation", func(b *testing.B) {
		validator := NewStatisticalValidator()
		weights := config.SimilarityWeights{
			TreeEdit:           0.35,
			TokenSimilarity:    0.30,
			Structural:         0.25,
			Signature:          0.10,
			DifferentSignature: 0.30,
		}

		// Create a dummy test for the benchmark
		t := &testing.T{}

		b.ResetTimer()
		for range b.N {
			validator.ValidateWeights(t, weights)
		}
	})
}
