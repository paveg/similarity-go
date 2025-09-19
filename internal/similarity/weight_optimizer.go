package similarity

import (
	"fmt"
	"io"
	"math"
	"os"
	"testing"

	"github.com/paveg/similarity-go/internal/config"
)

const (
	gridSearchStep          = 0.05
	treeEditMin             = 0.1
	treeEditMax             = 0.5
	tokenSimilarityMin      = 0.1
	tokenSimilarityMax      = 0.5
	structuralMin           = 0.1
	structuralMax           = 0.4
	signatureMin            = 0.05
	signatureMax            = 0.3
	weightSumTolerance      = 0.01
	percentageMultiplier100 = 100.0
	maxWorstCasesToReport   = 5
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
	detector := NewDetector(config.DefaultThreshold)
	detector.config = cfg

	var totalError float64
	var results []CaseResult

	for _, testCase := range wo.dataset {
		func1, func2 := testCase.CreateFunctionPair(t)

		actualSimilarity := detector.CalculateSimilarity(func1, func2)
		absError := math.Abs(actualSimilarity - testCase.ExpectedSimilarity)
		totalError += absError

		results = append(results, CaseResult{
			CaseName: testCase.Name,
			Expected: testCase.ExpectedSimilarity,
			Actual:   actualSimilarity,
			Error:    absError,
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

	for treeEdit := treeEditMin; treeEdit <= treeEditMax; treeEdit += gridSearchStep {
		for tokenSim := tokenSimilarityMin; tokenSim <= tokenSimilarityMax; tokenSim += gridSearchStep {
			for structural := structuralMin; structural <= structuralMax; structural += gridSearchStep {
				for signature := signatureMin; signature <= signatureMax; signature += gridSearchStep {
					// Ensure weights sum to approximately 1.0
					total := treeEdit + tokenSim + structural + signature
					if math.Abs(total-1.0) > weightSumTolerance {
						continue
					}

					weights := config.SimilarityWeights{
						TreeEdit:           treeEdit,
						TokenSimilarity:    tokenSim,
						Structural:         structural,
						Signature:          signature,
						DifferentSignature: config.DifferentSignatureWeight, // Keep penalty constant
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
	wo.printOptimizationReport(os.Stdout, result, currentScore)
}

func (wo *WeightOptimizer) printOptimizationReport(out io.Writer, result OptimizationResult, currentScore float64) {
	write := func(format string, args ...any) {
		_, _ = fmt.Fprintf(out, format, args...)
	}

	write("\n=== WEIGHT OPTIMIZATION REPORT ===\n")
	write("Total iterations: %d\n", result.IterationCount)
	write("Current weights score: %.4f\n", currentScore)
	write("Best optimized score: %.4f\n", result.BestScore)
	improvement := result.BestScore - currentScore
	write("Improvement: %.4f (%.2f%%)\n", improvement, improvement/currentScore*percentageMultiplier100)

	write("\n--- CURRENT vs OPTIMIZED WEIGHTS ---\n")
	write("Algorithm        Current  Optimized  Change\n")
	write("TreeEdit         %.3f    %.3f      %+.3f\n",
		config.TreeEditWeight, result.BestWeights.TreeEdit,
		result.BestWeights.TreeEdit-config.TreeEditWeight)
	write("TokenSimilarity  %.3f    %.3f      %+.3f\n",
		config.TokenSimilarityWeight, result.BestWeights.TokenSimilarity,
		result.BestWeights.TokenSimilarity-config.TokenSimilarityWeight)
	write("Structural       %.3f    %.3f      %+.3f\n",
		config.StructuralWeight, result.BestWeights.Structural,
		result.BestWeights.Structural-config.StructuralWeight)
	write("Signature        %.3f    %.3f      %+.3f\n",
		config.SignatureWeight, result.BestWeights.Signature,
		result.BestWeights.Signature-config.SignatureWeight)

	write("\n--- PERFORMANCE BY CATEGORY ---\n")
	categoryErrors := make(map[string][]float64)
	for _, detailed := range result.DetailedResults {
		categoryErrors[detailed.Category] = append(categoryErrors[detailed.Category], detailed.Error)
	}

	for category, errors := range categoryErrors {
		var avgError float64
		for _, err := range errors {
			avgError += err
		}
		avgError /= float64(len(errors))
		write("%-15s: Avg Error = %.4f (%d cases)\n", category, avgError, len(errors))
	}

	write("\n--- TOP 5 WORST PERFORMING CASES ---\n")
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

	for i := 0; i < maxWorstCasesToReport && i < len(worst); i++ {
		caseResult := worst[i]
		write("%d. %s (Category: %s)\n", i+1, caseResult.CaseName, caseResult.Category)
		write("   Expected: %.3f, Actual: %.3f, Error: %.3f\n",
			caseResult.Expected, caseResult.Actual, caseResult.Error)
	}
}
