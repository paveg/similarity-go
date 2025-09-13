package main

import (
	"fmt"
	"testing"

	"github.com/paveg/similarity-go/internal/config"
	"github.com/paveg/similarity-go/internal/similarity"
)

func main() {
	fmt.Println("🎯 Similarity Algorithm Weight Optimization Benchmark")
	fmt.Println("============================================================")

	// Create a test instance for the benchmark functions
	t := &testing.T{}

	// 1. Analyze current weights
	fmt.Println("\n📊 STEP 1: Analyzing Current Weights")
	optimizer := similarity.NewWeightOptimizer()
	currentScore, currentResults := optimizer.AnalyzeCurrentWeights(t)

	fmt.Printf("Current default weights performance: %.4f\n", currentScore)
	printCategoryBreakdown(currentResults)

	// 2. Run grid search optimization
	fmt.Println("\n🔍 STEP 2: Grid Search Optimization")
	fmt.Println("Running grid search optimization (this may take a moment)...")
	gridResult := optimizer.GridSearchOptimize(t)

	fmt.Printf("Grid Search Results:\n")
	fmt.Printf("- Iterations: %d\n", gridResult.IterationCount)
	fmt.Printf("- Best Score: %.4f\n", gridResult.BestScore)
	fmt.Printf("- Improvement: %.4f (%.2f%%)\n",
		gridResult.BestScore-currentScore,
		(gridResult.BestScore-currentScore)/currentScore*100)

	// 3. Run genetic algorithm optimization
	fmt.Println("\n🧬 STEP 3: Genetic Algorithm Optimization")
	fmt.Println("Running genetic algorithm optimization...")
	genetic := similarity.NewGeneticOptimizer()
	genetic.SetParameters(30, 50, 0.1, 0.8, 3) // Smaller params for demo
	geneticResult := genetic.OptimizeWeights(t)

	fmt.Printf("Genetic Algorithm Results:\n")
	fmt.Printf("- Generations: %d\n", len(geneticResult.GenerationHistory))
	fmt.Printf("- Best Score: %.4f\n", geneticResult.BestIndividual.Fitness)
	fmt.Printf("- Convergence Generation: %d\n", geneticResult.ConvergenceGen)

	// 4. Statistical validation
	fmt.Println("\n📈 STEP 4: Statistical Validation")
	validator := similarity.NewStatisticalValidator()

	// Validate grid search result
	gridValidation := validator.ValidateWeights(t, gridResult.BestWeights)
	fmt.Printf("Grid Search Validation:\n")
	printValidationSummary(gridValidation)

	// Validate genetic algorithm result
	geneticValidation := validator.ValidateWeights(t, geneticResult.BestIndividual.Weights)
	fmt.Printf("\nGenetic Algorithm Validation:\n")
	printValidationSummary(geneticValidation)

	// 5. Compare all approaches
	fmt.Println("\n🏆 STEP 5: Final Comparison")
	fmt.Printf("%-20s | Score  | MAE    | R²     | F1     \n", "Method")
	fmt.Printf("%-20s-|--------|--------|--------|--------\n", "")

	// Calculate basic validation for current weights
	currentWeights := config.SimilarityWeights{
		TreeEdit:           config.TreeEditWeight,
		TokenSimilarity:    config.TokenSimilarityWeight,
		Structural:         config.StructuralWeight,
		Signature:          config.SignatureWeight,
		DifferentSignature: config.DifferentSignatureWeight,
	}
	currentValidation := validator.ValidateWeights(t, currentWeights)

	fmt.Printf("%-20s | %.4f | %.4f | %.4f | %.4f\n", "Current Default", currentScore,
		currentValidation.MAE, currentValidation.R2, currentValidation.F1Score)

	fmt.Printf("%-20s | %.4f | %.4f | %.4f | %.4f\n", "Grid Search", gridResult.BestScore,
		gridValidation.MAE, gridValidation.R2, gridValidation.F1Score)

	fmt.Printf("%-20s | %.4f | %.4f | %.4f | %.4f\n", "Genetic Algorithm", geneticResult.BestIndividual.Fitness,
		geneticValidation.MAE, geneticValidation.R2, geneticValidation.F1Score)

	// 6. Generate recommended weights
	fmt.Println("\n💡 STEP 6: Weight Recommendations")

	bestMethod := "Current Default"
	bestWeights := currentWeights
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

	fmt.Printf("🎖️  Best performing method: %s (Score: %.4f)\n", bestMethod, bestScore)
	fmt.Printf("\n📋 Recommended Weight Configuration:\n")
	fmt.Printf("const (\n")
	fmt.Printf("    TreeEditWeight        = %.3f\n", bestWeights.TreeEdit)
	fmt.Printf("    TokenSimilarityWeight = %.3f\n", bestWeights.TokenSimilarity)
	fmt.Printf("    StructuralWeight      = %.3f\n", bestWeights.Structural)
	fmt.Printf("    SignatureWeight       = %.3f\n", bestWeights.Signature)
	fmt.Printf(")\n")

	fmt.Printf("\n📄 YAML Configuration:\n")
	fmt.Printf("similarity:\n")
	fmt.Printf("  weights:\n")
	fmt.Printf("    tree_edit: %.3f\n", bestWeights.TreeEdit)
	fmt.Printf("    token_similarity: %.3f\n", bestWeights.TokenSimilarity)
	fmt.Printf("    structural: %.3f\n", bestWeights.Structural)
	fmt.Printf("    signature: %.3f\n", bestWeights.Signature)

	fmt.Println("\n✅ Weight optimization benchmark completed!")
	fmt.Println("Recommendation: Review the results above and consider updating")
	fmt.Println("the default weights if significant improvements are found.")
}

func printCategoryBreakdown(results []similarity.CaseResult) {
	categoryErrors := make(map[string][]float64)
	for _, result := range results {
		categoryErrors[result.Category] = append(categoryErrors[result.Category], result.Error)
	}

	fmt.Println("Category Performance:")
	for category, errors := range categoryErrors {
		var avgError float64
		for _, err := range errors {
			avgError += err
		}
		avgError /= float64(len(errors))
		fmt.Printf("  %-15s: Avg Error = %.4f (%d cases)\n", category, avgError, len(errors))
	}
}

func printValidationSummary(validation similarity.ValidationResult) {
	fmt.Printf("  MAE: %.4f | MSE: %.4f | R²: %.4f\n",
		validation.MAE,
		validation.MSE,
		validation.R2)
	fmt.Printf("  F1-Score: %.4f | Accuracy: %.4f | Robustness: %.4f\n",
		validation.F1Score,
		validation.Accuracy,
		validation.RobustnessScore)
}
