package main

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/paveg/similarity-go/internal/config"
	"github.com/paveg/similarity-go/internal/similarity"
)

const (
	percentageMultiplier = 100.0
	demoPopulationSize   = 30
	demoGenerations      = 50
	demoMutationRate     = 0.1
	demoCrossoverRate    = 0.8
	demoEliteSize        = 3
)

func main() {
	stdout := os.Stdout
	t := &testing.T{}
	optimizer := similarity.NewWeightOptimizer()
	validator := similarity.NewStatisticalValidator()

	printHeader(stdout)

	currentScore := analyzeCurrentWeights(stdout, t, optimizer)
	gridResult := runGridSearch(stdout, t, optimizer, currentScore)
	geneticResult := runGeneticOptimization(stdout, t)

	gridValidation := validator.ValidateWeights(t, gridResult.BestWeights)
	geneticValidation := validator.ValidateWeights(t, geneticResult.BestIndividual.Weights)
	currentValidation := validator.ValidateWeights(t, defaultWeights())

	printValidationSummaries(
		stdout,
		currentScore,
		gridResult,
		geneticResult,
		gridValidation,
		geneticValidation,
		currentValidation,
	)

	bestMethod, bestWeights, bestScore := pickBestWeights(currentScore, gridResult, geneticResult)
	printRecommendations(stdout, bestMethod, bestWeights, bestScore)
}

func printHeader(out io.Writer) {
	_, _ = fmt.Fprintln(out, "🎯 Similarity Algorithm Weight Optimization Benchmark")
	_, _ = fmt.Fprintln(out, "============================================================")
}

// analyzeCurrentWeights prints baseline performance and returns the aggregate score.
func analyzeCurrentWeights(out io.Writer, t *testing.T, optimizer *similarity.WeightOptimizer) float64 {
	_, _ = fmt.Fprintln(out, "\n📊 STEP 1: Analyzing Current Weights")
	currentScore, currentResults := optimizer.AnalyzeCurrentWeights(t)
	_, _ = fmt.Fprintf(out, "Current default weights performance: %.4f\n", currentScore)
	printCategoryBreakdown(out, currentResults)
	return currentScore
}

func runGridSearch(
	out io.Writer,
	t *testing.T,
	optimizer *similarity.WeightOptimizer,
	currentScore float64,
) similarity.OptimizationResult {
	_, _ = fmt.Fprintln(out, "\n🔍 STEP 2: Grid Search Optimization")
	_, _ = fmt.Fprintln(out, "Running grid search optimization (this may take a moment)...")
	gridResult := optimizer.GridSearchOptimize(t)
	_, _ = fmt.Fprintln(out, "Grid Search Results:")
	_, _ = fmt.Fprintf(out, "- Iterations: %d\n", gridResult.IterationCount)
	_, _ = fmt.Fprintf(out, "- Best Score: %.4f\n", gridResult.BestScore)
	improvement := gridResult.BestScore - currentScore
	_, _ = fmt.Fprintf(
		out,
		"- Improvement: %.4f (%.2f%%)\n",
		improvement,
		improvement/currentScore*percentageMultiplier,
	)
	return gridResult
}

func runGeneticOptimization(out io.Writer, t *testing.T) similarity.GeneticResult {
	_, _ = fmt.Fprintln(out, "\n🧬 STEP 3: Genetic Algorithm Optimization")
	_, _ = fmt.Fprintln(out, "Running genetic algorithm optimization...")
	genetic := similarity.NewGeneticOptimizer()
	genetic.SetParameters(demoPopulationSize, demoGenerations, demoMutationRate, demoCrossoverRate, demoEliteSize)
	result := genetic.OptimizeWeights(t)
	_, _ = fmt.Fprintln(out, "Genetic Algorithm Results:")
	_, _ = fmt.Fprintf(out, "- Generations: %d\n", len(result.GenerationHistory))
	_, _ = fmt.Fprintf(out, "- Best Score: %.4f\n", result.BestIndividual.Fitness)
	_, _ = fmt.Fprintf(out, "- Convergence Generation: %d\n", result.ConvergenceGen)
	return result
}

func printValidationSummaries(
	out io.Writer,
	currentScore float64,
	gridResult similarity.OptimizationResult,
	geneticResult similarity.GeneticResult,
	gridValidation similarity.ValidationResult,
	geneticValidation similarity.ValidationResult,
	currentValidation similarity.ValidationResult,
) {
	_, _ = fmt.Fprintln(out, "\n📈 STEP 4: Statistical Validation")
	_, _ = fmt.Fprintln(out, "Grid Search Validation:")
	printValidationSummary(out, gridValidation)
	_, _ = fmt.Fprintln(out, "\nGenetic Algorithm Validation:")
	printValidationSummary(out, geneticValidation)

	_, _ = fmt.Fprintln(out, "\n🏆 STEP 5: Final Comparison")
	_, _ = fmt.Fprintf(out, "%-20s | Score  | MAE    | R²     | F1     \n", "Method")
	_, _ = fmt.Fprintf(out, "%-20s-|--------|--------|--------|--------\n", "")
	printComparisonRow(out, "Current Default", currentScore, currentValidation)
	printComparisonRow(out, "Grid Search", gridResult.BestScore, gridValidation)
	printComparisonRow(out, "Genetic Algorithm", geneticResult.BestIndividual.Fitness, geneticValidation)
}

func pickBestWeights(
	currentScore float64,
	gridResult similarity.OptimizationResult,
	geneticResult similarity.GeneticResult,
) (string, config.SimilarityWeights, float64) {
	bestMethod := "Current Default"
	bestWeights := defaultWeights()
	bestScore := currentScore

	if gridResult.BestScore > bestScore {
		bestMethod = "Grid Search"
		bestWeights = gridResult.BestWeights
		bestScore = gridResult.BestScore
	}

	if geneticResult.BestIndividual.Fitness > bestScore {
		bestMethod = "Genetic Algorithm"
		bestWeights = geneticResult.BestIndividual.Weights
		bestScore = geneticResult.BestIndividual.Fitness
	}

	return bestMethod, bestWeights, bestScore
}

func printRecommendations(out io.Writer, method string, bestWeights config.SimilarityWeights, bestScore float64) {
	_, _ = fmt.Fprintln(out, "\n💡 STEP 6: Weight Recommendations")
	_, _ = fmt.Fprintf(out, "🎖️  Best performing method: %s (Score: %.4f)\n", method, bestScore)
	_, _ = fmt.Fprintln(out, "\n📋 Recommended Weight Configuration:")
	_, _ = fmt.Fprintln(out, "const (")
	_, _ = fmt.Fprintf(out, "    TreeEditWeight        = %.3f\n", bestWeights.TreeEdit)
	_, _ = fmt.Fprintf(out, "    TokenSimilarityWeight = %.3f\n", bestWeights.TokenSimilarity)
	_, _ = fmt.Fprintf(out, "    StructuralWeight      = %.3f\n", bestWeights.Structural)
	_, _ = fmt.Fprintf(out, "    SignatureWeight       = %.3f\n", bestWeights.Signature)
	_, _ = fmt.Fprintln(out, ")")

	_, _ = fmt.Fprintln(out, "\n📄 YAML Configuration:")
	_, _ = fmt.Fprintln(out, "similarity:")
	_, _ = fmt.Fprintln(out, "  weights:")
	_, _ = fmt.Fprintf(out, "    tree_edit: %.3f\n", bestWeights.TreeEdit)
	_, _ = fmt.Fprintf(out, "    token_similarity: %.3f\n", bestWeights.TokenSimilarity)
	_, _ = fmt.Fprintf(out, "    structural: %.3f\n", bestWeights.Structural)
	_, _ = fmt.Fprintf(out, "    signature: %.3f\n", bestWeights.Signature)

	_, _ = fmt.Fprintln(out, "\n✅ Weight optimization benchmark completed!")
	_, _ = fmt.Fprintln(out, "Recommendation: Review the results above and consider updating")
	_, _ = fmt.Fprintln(out, "the default weights if significant improvements are found.")
}

func printCategoryBreakdown(out io.Writer, results []similarity.CaseResult) {
	categoryErrors := make(map[string][]float64)
	for _, result := range results {
		categoryErrors[result.Category] = append(categoryErrors[result.Category], result.Error)
	}

	_, _ = fmt.Fprintln(out, "Category Performance:")
	for category, errors := range categoryErrors {
		var avgError float64
		for _, err := range errors {
			avgError += err
		}
		avgError /= float64(len(errors))
		_, _ = fmt.Fprintf(out, "  %-15s: Avg Error = %.4f (%d cases)\n", category, avgError, len(errors))
	}
}

func printValidationSummary(out io.Writer, validation similarity.ValidationResult) {
	_, _ = fmt.Fprintf(out, "  MAE: %.4f | MSE: %.4f | R²: %.4f\n", validation.MAE, validation.MSE, validation.R2)
	_, _ = fmt.Fprintf(
		out,
		"  F1-Score: %.4f | Accuracy: %.4f | Robustness: %.4f\n",
		validation.F1Score,
		validation.Accuracy,
		validation.RobustnessScore,
	)
}

func printComparisonRow(out io.Writer, label string, score float64, validation similarity.ValidationResult) {
	_, _ = fmt.Fprintf(out, "%-20s | %.4f | %.4f | %.4f | %.4f\n",
		label,
		score,
		validation.MAE,
		validation.R2,
		validation.F1Score,
	)
}

func defaultWeights() config.SimilarityWeights {
	return config.SimilarityWeights{
		TreeEdit:           config.TreeEditWeight,
		TokenSimilarity:    config.TokenSimilarityWeight,
		Structural:         config.StructuralWeight,
		Signature:          config.SignatureWeight,
		DifferentSignature: config.DifferentSignatureWeight,
	}
}
