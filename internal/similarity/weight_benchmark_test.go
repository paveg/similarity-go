package similarity

import (
	"testing"

	"github.com/paveg/similarity-go/internal/config"
)

// TestWeightOptimizationDemo demonstrates the weight optimization system.
func TestWeightOptimizationDemo(t *testing.T) {
	t.Log("🎯 Starting Weight Optimization Demo")

	// 1. Create optimizer and analyze current weights
	optimizer := NewWeightOptimizer()

	t.Log("📊 Analyzing current default weights...")
	currentScore, currentResults := optimizer.AnalyzeCurrentWeights(t)
	t.Logf("Current score: %.4f", currentScore)

	// Print category breakdown
	categoryErrors := make(map[string][]float64)
	for _, result := range currentResults {
		categoryErrors[result.Category] = append(categoryErrors[result.Category], result.Error)
	}

	for category, errors := range categoryErrors {
		var avgError float64
		for _, err := range errors {
			avgError += err
		}
		avgError /= float64(len(errors))
		t.Logf("  %s: Avg Error = %.4f (%d cases)", category, avgError, len(errors))
	}

	// 2. Test a small grid search with limited range
	t.Log("🔍 Running limited grid search...")

	// Test just a few weight combinations manually
	testWeights := []config.SimilarityWeights{
		// Current weights
		{TreeEdit: 0.3, TokenSimilarity: 0.3, Structural: 0.25, Signature: 0.15, DifferentSignature: 0.3},
		// More emphasis on tree edit
		{TreeEdit: 0.4, TokenSimilarity: 0.25, Structural: 0.25, Signature: 0.1, DifferentSignature: 0.3},
		// More emphasis on token similarity
		{TreeEdit: 0.25, TokenSimilarity: 0.4, Structural: 0.25, Signature: 0.1, DifferentSignature: 0.3},
		// Balanced approach
		{TreeEdit: 0.35, TokenSimilarity: 0.35, Structural: 0.2, Signature: 0.1, DifferentSignature: 0.3},
	}

	bestScore := currentScore
	var bestWeights config.SimilarityWeights

	for i, weights := range testWeights {
		score, _ := optimizer.EvaluateWeights(t, weights)
		t.Logf("  Configuration %d: Score = %.4f", i+1, score)
		t.Logf("    TreeEdit=%.2f, Token=%.2f, Struct=%.2f, Sig=%.2f",
			weights.TreeEdit, weights.TokenSimilarity, weights.Structural, weights.Signature)

		if score > bestScore {
			bestScore = score
			bestWeights = weights
		}
	}

	// 3. Show results
	if bestScore > currentScore {
		improvement := (bestScore - currentScore) / currentScore * 100
		t.Logf("🎉 Found improvement: %.4f → %.4f (%.2f%% better)", currentScore, bestScore, improvement)
		t.Logf("💡 Best weights: TreeEdit=%.3f, Token=%.3f, Struct=%.3f, Sig=%.3f",
			bestWeights.TreeEdit, bestWeights.TokenSimilarity, bestWeights.Structural, bestWeights.Signature)
	} else {
		t.Log("📝 Current weights appear to be well-optimized for this dataset")
	}

	// 4. Test individual cases to show what's happening
	t.Log("\n📋 Sample case analysis:")
	dataset := GetBenchmarkDataset()

	// Create detector with current weights
	cfg := config.Default()
	detector := NewDetector(0.8)
	detector.config = cfg

	// Test a few representative cases
	testCases := []int{0, 1, 3, 6} // Pick some interesting cases
	for _, idx := range testCases {
		if idx < len(dataset) {
			testCase := dataset[idx]
			func1, func2 := testCase.CreateFunctionPair(t)
			actual := detector.CalculateSimilarity(func1, func2)

			t.Logf("  %s (Category: %s)", testCase.Name, testCase.Category)
			t.Logf("    Expected: %.3f, Actual: %.3f, Error: %.3f",
				testCase.ExpectedSimilarity, actual, abs(actual-testCase.ExpectedSimilarity))
		}
	}

	t.Log("✅ Weight optimization demo completed")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
