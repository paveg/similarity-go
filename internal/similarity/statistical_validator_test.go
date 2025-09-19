package similarity

import (
	"math"
	"testing"

	"github.com/paveg/similarity-go/internal/config"
)

func TestStatisticalValidator_NewStatisticalValidator(t *testing.T) {
	validator := NewStatisticalValidator()

	if validator == nil {
		t.Fatal("Expected non-nil validator")
	}

	if len(validator.dataset) == 0 {
		t.Error("Expected non-empty dataset")
	}
}

func TestStatisticalValidator_calculateRelativeError(t *testing.T) {
	validator := NewStatisticalValidator()

	tests := []struct {
		name     string
		expected float64
		actual   float64
		want     float64
	}{
		{"normal_case", 0.8, 0.6, 0.25},     // |0.6-0.8|/0.8 = 0.25
		{"zero_expected", 0.0, 0.5, 0.5},    // Special case: |0.5|
		{"zero_actual", 0.5, 0.0, 1.0},      // |0.0-0.5|/0.5 = 1.0
		{"identical_values", 0.7, 0.7, 0.0}, // Perfect match
		{"actual_higher", 0.6, 0.9, 0.5},    // |0.9-0.6|/0.6 = 0.5
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.calculateRelativeError(tt.expected, tt.actual)
			if math.Abs(got-tt.want) > 1e-10 {
				t.Errorf("calculateRelativeError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatisticalValidator_isCorrectlyClassified(t *testing.T) {
	validator := NewStatisticalValidator()

	tests := []struct {
		name     string
		expected float64
		actual   float64
		want     bool
	}{
		{"both_high", 0.8, 0.9, true},                 // Both >= 0.7
		{"both_low", 0.5, 0.3, true},                  // Both < 0.7
		{"expected_high_actual_low", 0.8, 0.6, false}, // Expected >= 0.7, actual < 0.7
		{"expected_low_actual_high", 0.5, 0.8, false}, // Expected < 0.7, actual >= 0.7
		{"boundary_expected", 0.7, 0.6, false},        // Expected exactly 0.7, actual < 0.7
		{"boundary_actual", 0.6, 0.7, false},          // Expected < 0.7, actual exactly 0.7
		{"both_boundary", 0.7, 0.7, true},             // Both exactly 0.7
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.isCorrectlyClassified(tt.expected, tt.actual)
			if got != tt.want {
				t.Errorf("isCorrectlyClassified(%v, %v) = %v, want %v",
					tt.expected, tt.actual, got, tt.want)
			}
		})
	}
}

func TestStatisticalValidator_percentile(t *testing.T) {
	validator := NewStatisticalValidator()

	tests := []struct {
		name     string
		data     []float64
		p        float64
		expected float64
	}{
		{"empty_data", []float64{}, 0.5, 0.0},
		{"single_element", []float64{5.0}, 0.5, 5.0},
		{"median_odd", []float64{1, 2, 3, 4, 5}, 0.5, 3.0},
		{"median_even", []float64{1, 2, 3, 4}, 0.5, 2.5},
		{"first_quartile", []float64{1, 2, 3, 4, 5}, 0.25, 2.0},
		{"third_quartile", []float64{1, 2, 3, 4, 5}, 0.75, 4.0},
		{"minimum", []float64{1, 2, 3, 4, 5}, 0.0, 1.0},
		{"maximum", []float64{1, 2, 3, 4, 5}, 1.0, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.percentile(tt.data, tt.p)
			if math.Abs(got-tt.expected) > 1e-10 {
				t.Errorf("percentile(%v, %v) = %v, want %v", tt.data, tt.p, got, tt.expected)
			}
		})
	}
}

func TestStatisticalValidator_getRanks(t *testing.T) {
	validator := NewStatisticalValidator()

	// Create test results with known values
	results := []DetailedCaseResult{
		{CaseResult: CaseResult{Expected: 0.9}}, // Rank 4 (highest)
		{CaseResult: CaseResult{Expected: 0.5}}, // Rank 2
		{CaseResult: CaseResult{Expected: 0.5}}, // Rank 2 (tied)
		{CaseResult: CaseResult{Expected: 0.1}}, // Rank 1 (lowest)
	}

	ranks := validator.getRanks(results, func(r DetailedCaseResult) float64 { return r.Expected })

	expectedRanks := []float64{4.0, 2.5, 2.5, 1.0} // Tied ranks averaged

	if len(ranks) != len(expectedRanks) {
		t.Fatalf("Expected %d ranks, got %d", len(expectedRanks), len(ranks))
	}

	for i, expected := range expectedRanks {
		if math.Abs(ranks[i]-expected) > 1e-10 {
			t.Errorf("Rank[%d] = %v, want %v", i, ranks[i], expected)
		}
	}
}

//nolint:gocognit,gocyclo,cyclop // comprehensive validation across multiple metrics
func TestStatisticalValidator_ValidateWeights(t *testing.T) {
	validator := NewStatisticalValidator()

	weights := config.SimilarityWeights{
		TreeEdit:           0.3,
		TokenSimilarity:    0.3,
		Structural:         0.25,
		Signature:          0.15,
		DifferentSignature: 0.3,
	}

	result := validator.ValidateWeights(t, weights)

	// Basic validation of result structure
	if len(result.CaseResults) == 0 {
		t.Error("Expected non-empty case results")
	}

	// Check metric ranges
	if result.MAE < 0 || result.MAE > 1 {
		t.Errorf("MAE out of expected range [0,1]: %f", result.MAE)
	}

	if result.MSE < 0 || result.MSE > 1 {
		t.Errorf("MSE out of expected range [0,1]: %f", result.MSE)
	}

	if result.RMSE < 0 || result.RMSE > 1 {
		t.Errorf("RMSE out of expected range [0,1]: %f", result.RMSE)
	}

	if result.PearsonR < -1 || result.PearsonR > 1 {
		t.Errorf("Pearson correlation out of range [-1,1]: %f", result.PearsonR)
	}

	if result.SpearmanRho < -1 || result.SpearmanRho > 1 {
		t.Errorf("Spearman correlation out of range [-1,1]: %f", result.SpearmanRho)
	}

	// Classification metrics should be between 0 and 1
	if result.Precision < 0 || result.Precision > 1 {
		t.Errorf("Precision out of range [0,1]: %f", result.Precision)
	}

	if result.Recall < 0 || result.Recall > 1 {
		t.Errorf("Recall out of range [0,1]: %f", result.Recall)
	}

	if result.F1Score < 0 || result.F1Score > 1 {
		t.Errorf("F1Score out of range [0,1]: %f", result.F1Score)
	}

	if result.Accuracy < 0 || result.Accuracy > 1 {
		t.Errorf("Accuracy out of range [0,1]: %f", result.Accuracy)
	}

	// Robustness metrics should be between 0 and 1
	if result.RobustnessScore < 0 || result.RobustnessScore > 1 {
		t.Errorf("RobustnessScore out of range [0,1]: %f", result.RobustnessScore)
	}

	if result.ConsistencyScore < 0 || result.ConsistencyScore > 1 {
		t.Errorf("ConsistencyScore out of range [0,1]: %f", result.ConsistencyScore)
	}

	if result.DiscriminationScore < 0 || result.DiscriminationScore > 1 {
		t.Errorf("DiscriminationScore out of range [0,1]: %f", result.DiscriminationScore)
	}

	// Error distribution validation
	dist := result.ErrorDistribution
	if dist.Mean < 0 {
		t.Error("Mean error should be non-negative")
	}

	if dist.Median < 0 {
		t.Error("Median error should be non-negative")
	}

	if dist.StdDev < 0 {
		t.Error("Standard deviation should be non-negative")
	}

	if dist.Q25 < 0 || dist.Q75 < 0 {
		t.Error("Quartiles should be non-negative")
	}

	if dist.IQR < 0 {
		t.Error("IQR should be non-negative")
	}

	// Category performance validation
	if len(result.CategoryPerformance) == 0 {
		t.Error("Expected category performance data")
	}

	for category, stats := range result.CategoryPerformance {
		if stats.Count <= 0 {
			t.Errorf("Category %s should have positive count", category)
		}

		if stats.MeanError < 0 {
			t.Errorf("Category %s mean error should be non-negative", category)
		}

		if stats.MedianError < 0 {
			t.Errorf("Category %s median error should be non-negative", category)
		}

		if stats.StdDevError < 0 {
			t.Errorf("Category %s std dev error should be non-negative", category)
		}
	}

	// Detailed results validation
	for i, caseResult := range result.CaseResults {
		if caseResult.AbsoluteError < 0 {
			t.Errorf("Case %d absolute error should be non-negative", i)
		}

		if caseResult.RelativeError < 0 {
			t.Errorf("Case %d relative error should be non-negative", i)
		}

		if caseResult.SquaredError < 0 {
			t.Errorf("Case %d squared error should be non-negative", i)
		}

		// Check classification consistency
		expectedHigh := caseResult.Expected >= 0.7
		actualHigh := caseResult.Actual >= 0.7
		correctClass := expectedHigh == actualHigh

		if caseResult.IsHighSimilarity != expectedHigh {
			t.Errorf("Case %d IsHighSimilarity inconsistent", i)
		}

		if caseResult.PredictedHigh != actualHigh {
			t.Errorf("Case %d PredictedHigh inconsistent", i)
		}

		if caseResult.CorrectlyClassified != correctClass {
			t.Errorf("Case %d CorrectlyClassified inconsistent", i)
		}
	}
}

func TestStatisticalValidator_calculateSpearmanRho(t *testing.T) {
	validator := NewStatisticalValidator()

	tests := []struct {
		name      string
		results   []DetailedCaseResult
		expected  float64
		tolerance float64
	}{
		{
			name: "perfect_positive_correlation",
			results: []DetailedCaseResult{
				{CaseResult: CaseResult{Expected: 0.1, Actual: 0.2}},
				{CaseResult: CaseResult{Expected: 0.5, Actual: 0.6}},
				{CaseResult: CaseResult{Expected: 0.9, Actual: 1.0}},
			},
			expected:  1.0,
			tolerance: 1e-10,
		},
		{
			name: "perfect_negative_correlation",
			results: []DetailedCaseResult{
				{CaseResult: CaseResult{Expected: 0.1, Actual: 0.9}},
				{CaseResult: CaseResult{Expected: 0.5, Actual: 0.5}},
				{CaseResult: CaseResult{Expected: 0.9, Actual: 0.1}},
			},
			expected:  -1.0,
			tolerance: 1e-10,
		},
		{
			name: "no_correlation",
			results: []DetailedCaseResult{
				{CaseResult: CaseResult{Expected: 0.1, Actual: 0.5}},
				{CaseResult: CaseResult{Expected: 0.5, Actual: 0.1}},
				{CaseResult: CaseResult{Expected: 0.9, Actual: 0.9}},
			},
			expected:  0.0,
			tolerance: 0.6, // More tolerance for this case
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validator.calculateSpearmanRho(tt.results)
			if math.Abs(got-tt.expected) > tt.tolerance {
				t.Errorf("calculateSpearmanRho() = %v, want %v (tolerance %v)",
					got, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestStatisticalValidator_calculateConsistencyScore(t *testing.T) {
	validator := NewStatisticalValidator()

	// Test with perfectly consistent results (no variance within ranges)
	consistentResults := []DetailedCaseResult{
		{CaseResult: CaseResult{Expected: 0.1, Actual: 0.1}}, // Low range
		{CaseResult: CaseResult{Expected: 0.2, Actual: 0.1}}, // Low range
		{CaseResult: CaseResult{Expected: 0.5, Actual: 0.5}}, // Medium range
		{CaseResult: CaseResult{Expected: 0.6, Actual: 0.5}}, // Medium range
		{CaseResult: CaseResult{Expected: 0.8, Actual: 0.8}}, // High range
		{CaseResult: CaseResult{Expected: 0.9, Actual: 0.8}}, // High range
	}

	score := validator.calculateConsistencyScore(consistentResults)
	if score <= 0.5 {
		t.Errorf("Expected high consistency score for consistent results, got %f", score)
	}

	// Test with inconsistent results (high variance within ranges)
	inconsistentResults := []DetailedCaseResult{
		{CaseResult: CaseResult{Expected: 0.1, Actual: 0.01}}, // Low range
		{CaseResult: CaseResult{Expected: 0.2, Actual: 0.29}}, // Low range - high variance
		{CaseResult: CaseResult{Expected: 0.5, Actual: 0.3}},  // Medium range
		{CaseResult: CaseResult{Expected: 0.6, Actual: 0.69}}, // Medium range - high variance
		{CaseResult: CaseResult{Expected: 0.8, Actual: 0.7}},  // High range
		{CaseResult: CaseResult{Expected: 0.9, Actual: 0.99}}, // High range - high variance
	}

	score2 := validator.calculateConsistencyScore(inconsistentResults)
	if score2 >= score {
		t.Errorf("Expected lower consistency score for inconsistent results, got %f vs %f", score2, score)
	}
}

func TestStatisticalValidator_calculateDiscriminationScore(t *testing.T) {
	validator := NewStatisticalValidator()

	// Test with perfect discrimination
	perfectResults := []DetailedCaseResult{
		{CaseResult: CaseResult{Expected: 0.1, Actual: 0.1}},
		{CaseResult: CaseResult{Expected: 0.5, Actual: 0.5}},
		{CaseResult: CaseResult{Expected: 0.9, Actual: 0.9}},
	}

	score := validator.calculateDiscriminationScore(perfectResults)
	if score != 1.0 {
		t.Errorf("Expected perfect discrimination score 1.0, got %f", score)
	}

	// Test with poor discrimination (reversed order)
	poorResults := []DetailedCaseResult{
		{CaseResult: CaseResult{Expected: 0.1, Actual: 0.9}},
		{CaseResult: CaseResult{Expected: 0.5, Actual: 0.5}},
		{CaseResult: CaseResult{Expected: 0.9, Actual: 0.1}},
	}

	score2 := validator.calculateDiscriminationScore(poorResults)
	if score2 >= 0.5 {
		t.Errorf("Expected poor discrimination score < 0.5, got %f", score2)
	}

	// Test with identical expected values (should not contribute to discrimination)
	identicalResults := []DetailedCaseResult{
		{CaseResult: CaseResult{Expected: 0.5, Actual: 0.1}},
		{CaseResult: CaseResult{Expected: 0.5, Actual: 0.9}},
	}

	score3 := validator.calculateDiscriminationScore(identicalResults)
	if score3 != 0.5 { // Default score when no discrimination possible
		t.Errorf("Expected default discrimination score 0.5 for identical expected values, got %f", score3)
	}
}

func TestStatisticalValidator_calculateStatsForGroup(t *testing.T) {
	validator := NewStatisticalValidator()

	cases := []DetailedCaseResult{
		{
			CaseResult: CaseResult{
				CaseName: "case1",
				Expected: 0.8,
				Actual:   0.7,
				Error:    0.1,
				Category: "test",
			},
			AbsoluteError: 0.1,
		},
		{
			CaseResult: CaseResult{
				CaseName: "case2",
				Expected: 0.9,
				Actual:   0.8,
				Error:    0.1,
				Category: "test",
			},
			AbsoluteError: 0.1,
		},
		{
			CaseResult: CaseResult{
				CaseName: "case3",
				Expected: 0.7,
				Actual:   0.5,
				Error:    0.2,
				Category: "test",
			},
			AbsoluteError: 0.2,
		},
	}

	stats := validator.calculateStatsForGroup("test", cases)

	if stats.Category != "test" {
		t.Errorf("Expected category 'test', got '%s'", stats.Category)
	}

	if stats.Count != 3 {
		t.Errorf("Expected count 3, got %d", stats.Count)
	}

	expectedMeanError := (0.1 + 0.1 + 0.2) / 3.0
	if math.Abs(stats.MeanError-expectedMeanError) > 1e-10 {
		t.Errorf("Expected mean error %f, got %f", expectedMeanError, stats.MeanError)
	}

	expectedMeanExpected := (0.8 + 0.9 + 0.7) / 3.0
	if math.Abs(stats.MeanExpected-expectedMeanExpected) > 1e-10 {
		t.Errorf("Expected mean expected %f, got %f", expectedMeanExpected, stats.MeanExpected)
	}

	expectedMeanActual := (0.7 + 0.8 + 0.5) / 3.0
	if math.Abs(stats.MeanActual-expectedMeanActual) > 1e-10 {
		t.Errorf("Expected mean actual %f, got %f", expectedMeanActual, stats.MeanActual)
	}

	// Best case should have lowest error (case1 or case2)
	if stats.BestCase.Error != 0.1 {
		t.Errorf("Expected best case error 0.1, got %f", stats.BestCase.Error)
	}

	// Worst case should have highest error (case3)
	if stats.WorstCase.Error != 0.2 {
		t.Errorf("Expected worst case error 0.2, got %f", stats.WorstCase.Error)
	}

	// Test empty group
	emptyStats := validator.calculateStatsForGroup("empty", []DetailedCaseResult{})
	if emptyStats.Count != 0 {
		t.Error("Expected zero count for empty group")
	}
}

func BenchmarkStatisticalValidator_ValidateWeights(b *testing.B) {
	validator := NewStatisticalValidator()
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
		validator.ValidateWeights(t, weights)
	}
}
