package similarity

import (
	"math"
	"testing"

	"github.com/paveg/similarity-go/internal/config"
)

//nolint:gocognit // multiple subtests cover various scoring scenarios
func TestWeightOptimizer_EvaluateWeights(t *testing.T) {
	optimizer := NewWeightOptimizer()

	tests := []struct {
		name           string
		weights        config.SimilarityWeights
		expectedScore  float64 // approximate expected score
		scoreThreshold float64 // tolerance for score comparison
	}{
		{
			name: "perfect_weights_high_score",
			weights: config.SimilarityWeights{
				TreeEdit:           0.35,
				TokenSimilarity:    0.30,
				Structural:         0.25,
				Signature:          0.10,
				DifferentSignature: 0.3,
			},
			expectedScore:  0.79,
			scoreThreshold: 0.05,
		},
		{
			name: "unbalanced_weights_lower_score",
			weights: config.SimilarityWeights{
				TreeEdit:           0.9, // Too high
				TokenSimilarity:    0.05,
				Structural:         0.03,
				Signature:          0.02,
				DifferentSignature: 0.3,
			},
			expectedScore:  0.5, // Should be lower due to imbalance
			scoreThreshold: 0.3,
		},
		{
			name: "current_default_weights",
			weights: config.SimilarityWeights{
				TreeEdit:           config.TreeEditWeight,
				TokenSimilarity:    config.TokenSimilarityWeight,
				Structural:         config.StructuralWeight,
				Signature:          config.SignatureWeight,
				DifferentSignature: config.DifferentSignatureWeight,
			},
			expectedScore:  0.79,
			scoreThreshold: 0.05,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, results := optimizer.EvaluateWeights(t, tt.weights)

			// Check score is within expected range
			if math.Abs(score-tt.expectedScore) > tt.scoreThreshold {
				t.Errorf("Score %f not within threshold %f of expected %f",
					score, tt.scoreThreshold, tt.expectedScore)
			}

			// Verify results structure
			if len(results) == 0 {
				t.Error("Expected non-empty results")
			}

			// Check each result has required fields
			for i, result := range results {
				if result.CaseName == "" {
					t.Errorf("Result %d missing case name", i)
				}
				if result.Category == "" {
					t.Errorf("Result %d missing category", i)
				}
				if result.Expected < 0 || result.Expected > 1 {
					t.Errorf("Result %d expected similarity out of range: %f", i, result.Expected)
				}
				if result.Actual < 0 || result.Actual > 1 {
					t.Errorf("Result %d actual similarity out of range: %f", i, result.Actual)
				}
				if result.Error < 0 {
					t.Errorf("Result %d error cannot be negative: %f", i, result.Error)
				}
			}

			// Score should be between 0 and 1
			if score < 0 || score > 1 {
				t.Errorf("Score out of range [0,1]: %f", score)
			}
		})
	}
}

func TestWeightOptimizer_GridSearchOptimize(t *testing.T) {
	optimizer := NewWeightOptimizer()

	// Run optimization (limited search space for testing)
	result := optimizer.GridSearchOptimize(t)

	// Verify optimization result structure
	if result.BestScore < 0 || result.BestScore > 1 {
		t.Errorf("Best score out of range [0,1]: %f", result.BestScore)
	}

	if result.IterationCount <= 0 {
		t.Error("Expected positive iteration count")
	}

	// Verify weights sum approximately to 1
	weights := result.BestWeights
	total := weights.TreeEdit + weights.TokenSimilarity + weights.Structural + weights.Signature
	if math.Abs(total-1.0) > 0.02 {
		t.Errorf("Best weights don't sum to ~1.0: %f", total)
	}

	// Each weight should be positive
	if weights.TreeEdit <= 0 || weights.TokenSimilarity <= 0 ||
		weights.Structural <= 0 || weights.Signature <= 0 {
		t.Error("All weights should be positive")
	}

	// Detailed results should match dataset
	expectedCases := len(GetBenchmarkDataset())
	if len(result.DetailedResults) != expectedCases {
		t.Errorf("Expected %d detailed results, got %d", expectedCases, len(result.DetailedResults))
	}
}

func TestWeightOptimizer_AnalyzeCurrentWeights(t *testing.T) {
	optimizer := NewWeightOptimizer()

	score, results := optimizer.AnalyzeCurrentWeights(t)

	// Basic validation
	if score < 0 || score > 1 {
		t.Errorf("Score out of range [0,1]: %f", score)
	}

	if len(results) == 0 {
		t.Error("Expected non-empty results")
	}

	// Results should match benchmark dataset
	expectedCases := len(GetBenchmarkDataset())
	if len(results) != expectedCases {
		t.Errorf("Expected %d results, got %d", expectedCases, len(results))
	}

	// Verify each result
	for _, result := range results {
		if result.Error < 0 {
			t.Errorf("Negative error for case %s: %f", result.CaseName, result.Error)
		}
		if result.Expected < 0 || result.Expected > 1 {
			t.Errorf("Invalid expected similarity for case %s: %f", result.CaseName, result.Expected)
		}
	}
}

func TestWeightOptimizer_CategoryPerformance(t *testing.T) {
	optimizer := NewWeightOptimizer()

	// Test with current weights
	_, results := optimizer.AnalyzeCurrentWeights(t)

	// Group results by category
	categoryResults := make(map[string][]CaseResult)
	for _, result := range results {
		categoryResults[result.Category] = append(categoryResults[result.Category], result)
	}

	// Expected categories based on benchmark data
	expectedCategories := []string{"identical", "high_similar", "medium_similar", "low_similar", "different"}

	for _, category := range expectedCategories {
		if len(categoryResults[category]) == 0 {
			t.Errorf("No results found for category: %s", category)
			continue
		}

		// Calculate average error for category
		var totalError float64
		for _, result := range categoryResults[category] {
			totalError += result.Error
		}
		avgError := totalError / float64(len(categoryResults[category]))

		// Category-specific performance expectations
		switch category {
		case "identical":
			// Identical cases should have very low error
			if avgError > 0.1 {
				t.Errorf("High error for identical functions: %f", avgError)
			}
		case "high_similar":
			// High similarity cases should have moderate error
			if avgError > 0.3 {
				t.Errorf("High error for high similarity functions: %f", avgError)
			}
		case "different":
			// Different functions might have higher acceptable error
			if avgError > 0.55 {
				t.Errorf("Unexpectedly high error for different functions: %f", avgError)
			}
		}
	}
}

func TestWeightOptimizer_ConsistencyCheck(t *testing.T) {
	optimizer := NewWeightOptimizer()

	// Same weights should produce same results
	weights := config.SimilarityWeights{
		TreeEdit:           0.3,
		TokenSimilarity:    0.3,
		Structural:         0.25,
		Signature:          0.15,
		DifferentSignature: 0.3,
	}

	score1, results1 := optimizer.EvaluateWeights(t, weights)
	score2, results2 := optimizer.EvaluateWeights(t, weights)

	if math.Abs(score1-score2) > 1e-10 {
		t.Errorf("Inconsistent scores: %f vs %f", score1, score2)
	}

	if len(results1) != len(results2) {
		t.Errorf("Different result lengths: %d vs %d", len(results1), len(results2))
	}

	for i, r1 := range results1 {
		if i >= len(results2) {
			break
		}
		r2 := results2[i]

		if math.Abs(r1.Actual-r2.Actual) > 1e-10 {
			t.Errorf("Inconsistent actual similarity for case %s: %f vs %f",
				r1.CaseName, r1.Actual, r2.Actual)
		}
	}
}

//nolint:gocognit // edge case matrix requires multiple branches
func TestWeightOptimizer_EdgeCases(t *testing.T) {
	optimizer := NewWeightOptimizer()

	tests := []struct {
		name    string
		weights config.SimilarityWeights
		valid   bool
	}{
		{
			name: "zero_weights",
			weights: config.SimilarityWeights{
				TreeEdit:           0,
				TokenSimilarity:    0,
				Structural:         0,
				Signature:          0,
				DifferentSignature: 0.3,
			},
			valid: false, // Should handle gracefully but may produce poor results
		},
		{
			name: "single_weight_dominant",
			weights: config.SimilarityWeights{
				TreeEdit:           1.0,
				TokenSimilarity:    0,
				Structural:         0,
				Signature:          0,
				DifferentSignature: 0.3,
			},
			valid: true,
		},
		{
			name: "negative_weights",
			weights: config.SimilarityWeights{
				TreeEdit:           -0.1,
				TokenSimilarity:    0.5,
				Structural:         0.3,
				Signature:          0.3,
				DifferentSignature: 0.3,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic even with edge case weights
			defer func() {
				if r := recover(); r != nil {
					if tt.valid {
						t.Errorf("Unexpected panic for valid case: %v", r)
					}
				}
			}()

			score, results := optimizer.EvaluateWeights(t, tt.weights)

			if tt.valid {
				// For valid cases, check basic properties
				if score < 0 || score > 1 {
					t.Errorf("Score out of range for valid case: %f", score)
				}
				if len(results) == 0 {
					t.Error("Expected non-empty results for valid case")
				}
			}
		})
	}
}

func BenchmarkWeightOptimizer_EvaluateWeights(b *testing.B) {
	optimizer := NewWeightOptimizer()
	weights := config.SimilarityWeights{
		TreeEdit:           0.3,
		TokenSimilarity:    0.3,
		Structural:         0.25,
		Signature:          0.15,
		DifferentSignature: 0.3,
	}

	// Create a dummy test for the benchmark
	t := &testing.T{}

	b.ResetTimer()
	for range b.N {
		optimizer.EvaluateWeights(t, weights)
	}
}
