package similarity

import (
	"fmt"
	"math"
	"testing"

	"github.com/paveg/similarity-go/internal/config"
)

// WeightOptimizer optimizes similarity algorithm weights using benchmark data.
type WeightOptimizer struct {
	dataset []BenchmarkCase
}

// NewWeightOptimizer creates a new weight optimizer.
func NewWeightOptimizer() *WeightOptimizer {
	return &WeightOptimizer{
		dataset: GetBenchmarkDataset(),
	}
}

// OptimizationResult contains the results of weight optimization.
type OptimizationResult struct {
	BestWeights     config.SimilarityWeights
	BestScore       float64
	IterationCount  int
	DetailedResults []CaseResult
}

// CaseResult contains the result for a single test case.
type CaseResult struct {
	CaseName string
	Expected float64
	Actual   float64
	Error    float64
	Category string
}

// EvaluateWeights evaluates a set of weights against the benchmark dataset.
func (wo *WeightOptimizer) EvaluateWeights(t *testing.T, weights config.SimilarityWeights) (float64, []CaseResult) {
	// Create detector with given weights
	cfg := config.Default()
	cfg.Similarity.Weights = weights
	detector := NewDetector(0.8)
	detector.config = cfg

	var totalError float64
	var results []CaseResult

	for _, testCase := range wo.dataset {
		func1, func2 := testCase.CreateFunctionPair(t)

		actualSimilarity := detector.CalculateSimilarity(func1, func2)
		error := math.Abs(actualSimilarity - testCase.ExpectedSimilarity)
		totalError += error

		results = append(results, CaseResult{
			CaseName: testCase.Name,
			Expected: testCase.ExpectedSimilarity,
			Actual:   actualSimilarity,
			Error:    error,
			Category: testCase.Category,
		})
	}

	// Calculate mean absolute error (lower is better)
	meanError := totalError / float64(len(wo.dataset))
	// Convert to score (higher is better)
	score := 1.0 - meanError

	return score, results
}

// GridSearchOptimize performs grid search optimization over weight space.
func (wo *WeightOptimizer) GridSearchOptimize(t *testing.T) OptimizationResult {
	bestScore := -1.0
	var bestWeights config.SimilarityWeights
	var bestResults []CaseResult
	iterationCount := 0

	// Define search space (step size 0.05 for reasonable granularity)
	step := 0.05

	for treeEdit := 0.1; treeEdit <= 0.5; treeEdit += step {
		for tokenSim := 0.1; tokenSim <= 0.5; tokenSim += step {
			for structural := 0.1; structural <= 0.4; structural += step {
				for signature := 0.05; signature <= 0.3; signature += step {
					// Ensure weights sum to approximately 1.0
					total := treeEdit + tokenSim + structural + signature
					if math.Abs(total-1.0) > 0.01 {
						continue
					}

					weights := config.SimilarityWeights{
						TreeEdit:           treeEdit,
						TokenSimilarity:    tokenSim,
						Structural:         structural,
						Signature:          signature,
						DifferentSignature: 0.3, // Keep penalty constant
					}

					score, results := wo.EvaluateWeights(t, weights)
					iterationCount++

					if score > bestScore {
						bestScore = score
						bestWeights = weights
						bestResults = results
					}
				}
			}
		}
	}

	return OptimizationResult{
		BestWeights:     bestWeights,
		BestScore:       bestScore,
		IterationCount:  iterationCount,
		DetailedResults: bestResults,
	}
}

// AnalyzeCurrentWeights analyzes the performance of current default weights.
func (wo *WeightOptimizer) AnalyzeCurrentWeights(t *testing.T) (float64, []CaseResult) {
	currentWeights := config.SimilarityWeights{
		TreeEdit:           config.TreeEditWeight,
		TokenSimilarity:    config.TokenSimilarityWeight,
		Structural:         config.StructuralWeight,
		Signature:          config.SignatureWeight,
		DifferentSignature: config.DifferentSignatureWeight,
	}

	return wo.EvaluateWeights(t, currentWeights)
}

// PrintOptimizationReport prints a detailed report of optimization results.
func (wo *WeightOptimizer) PrintOptimizationReport(result OptimizationResult, currentScore float64) {
	fmt.Printf("\n=== WEIGHT OPTIMIZATION REPORT ===\n")
	fmt.Printf("Total iterations: %d\n", result.IterationCount)
	fmt.Printf("Current weights score: %.4f\n", currentScore)
	fmt.Printf("Best optimized score: %.4f\n", result.BestScore)
	fmt.Printf("Improvement: %.4f (%.2f%%)\n",
		result.BestScore-currentScore,
		(result.BestScore-currentScore)/currentScore*100)

	fmt.Printf("\n--- CURRENT vs OPTIMIZED WEIGHTS ---\n")
	fmt.Printf("Algorithm        Current  Optimized  Change\n")
	fmt.Printf("TreeEdit         %.3f    %.3f      %+.3f\n",
		config.TreeEditWeight, result.BestWeights.TreeEdit,
		result.BestWeights.TreeEdit-config.TreeEditWeight)
	fmt.Printf("TokenSimilarity  %.3f    %.3f      %+.3f\n",
		config.TokenSimilarityWeight, result.BestWeights.TokenSimilarity,
		result.BestWeights.TokenSimilarity-config.TokenSimilarityWeight)
	fmt.Printf("Structural       %.3f    %.3f      %+.3f\n",
		config.StructuralWeight, result.BestWeights.Structural,
		result.BestWeights.Structural-config.StructuralWeight)
	fmt.Printf("Signature        %.3f    %.3f      %+.3f\n",
		config.SignatureWeight, result.BestWeights.Signature,
		result.BestWeights.Signature-config.SignatureWeight)

	fmt.Printf("\n--- PERFORMANCE BY CATEGORY ---\n")
	categoryErrors := make(map[string][]float64)
	for _, result := range result.DetailedResults {
		categoryErrors[result.Category] = append(categoryErrors[result.Category], result.Error)
	}

	for category, errors := range categoryErrors {
		var avgError float64
		for _, err := range errors {
			avgError += err
		}
		avgError /= float64(len(errors))
		fmt.Printf("%-15s: Avg Error = %.4f (%d cases)\n", category, avgError, len(errors))
	}

	fmt.Printf("\n--- TOP 5 WORST PERFORMING CASES ---\n")
	// Sort results by error (descending)
	worst := make([]CaseResult, len(result.DetailedResults))
	copy(worst, result.DetailedResults)
	for i := range len(worst) - 1 {
		for j := i + 1; j < len(worst); j++ {
			if worst[i].Error < worst[j].Error {
				worst[i], worst[j] = worst[j], worst[i]
			}
		}
	}

	for i := 0; i < 5 && i < len(worst); i++ {
		caseResult := worst[i]
		fmt.Printf("%d. %s (Category: %s)\n", i+1, caseResult.CaseName, caseResult.Category)
		fmt.Printf("   Expected: %.3f, Actual: %.3f, Error: %.3f\n",
			caseResult.Expected, caseResult.Actual, caseResult.Error)
	}
}
