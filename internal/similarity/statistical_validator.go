package similarity

import (
	"fmt"
	"math"
	"sort"
	"testing"

	"github.com/paveg/similarity-go/internal/config"
)

// StatisticalValidator provides comprehensive statistical analysis for similarity algorithms.
type StatisticalValidator struct {
	dataset []BenchmarkCase
}

// ValidationResult contains comprehensive validation metrics.
type ValidationResult struct {
	// Basic metrics
	MAE         float64 // Mean Absolute Error
	MSE         float64 // Mean Squared Error
	RMSE        float64 // Root Mean Squared Error
	R2          float64 // Coefficient of Determination
	PearsonR    float64 // Pearson Correlation Coefficient
	SpearmanRho float64 // Spearman Rank Correlation

	// Classification metrics (treating as similarity classification problem)
	Precision float64 // Precision for high similarity detection
	Recall    float64 // Recall for high similarity detection
	F1Score   float64 // F1 Score
	Accuracy  float64 // Classification accuracy

	// Distribution analysis
	ErrorDistribution   ErrorDistributionStats
	CategoryPerformance map[string]CategoryStats

	// Robustness metrics
	RobustnessScore     float64 // Overall robustness measure
	ConsistencyScore    float64 // Consistency across similar cases
	DiscriminationScore float64 // Ability to distinguish different cases

	// Detailed results
	CaseResults []DetailedCaseResult
}

// ErrorDistributionStats contains statistical properties of prediction errors.
type ErrorDistributionStats struct {
	Mean     float64
	Median   float64
	StdDev   float64
	Skewness float64
	Kurtosis float64
	Q25      float64 // 25th percentile
	Q75      float64 // 75th percentile
	IQR      float64 // Interquartile Range
}

// CategoryStats contains performance metrics for each category.
type CategoryStats struct {
	Category     string
	Count        int
	MeanError    float64
	MedianError  float64
	StdDevError  float64
	MeanActual   float64
	MeanExpected float64
	WorstCase    DetailedCaseResult
	BestCase     DetailedCaseResult
}

// DetailedCaseResult extends CaseResult with additional analysis.
type DetailedCaseResult struct {
	CaseResult

	AbsoluteError       float64
	RelativeError       float64
	SquaredError        float64
	IsHighSimilarity    bool // Expected >= 0.7
	PredictedHigh       bool // Actual >= 0.7
	CorrectlyClassified bool
}

// NewStatisticalValidator creates a new statistical validator.
func NewStatisticalValidator() *StatisticalValidator {
	return &StatisticalValidator{
		dataset: GetBenchmarkDataset(),
	}
}

// ValidateWeights performs comprehensive statistical validation of weights.
func (sv *StatisticalValidator) ValidateWeights(t *testing.T, weights config.SimilarityWeights) ValidationResult {
	// Get basic evaluation results
	optimizer := &WeightOptimizer{dataset: sv.dataset}
	_, basicResults := optimizer.EvaluateWeights(t, weights)

	// Convert to detailed results
	detailedResults := sv.convertToDetailedResults(basicResults)

	// Calculate all metrics
	result := ValidationResult{
		CaseResults: detailedResults,
	}

	sv.calculateBasicMetrics(&result)
	sv.calculateClassificationMetrics(&result)
	sv.calculateDistributionStats(&result)
	sv.calculateCategoryStats(&result)
	sv.calculateRobustnessMetrics(&result)

	return result
}

// convertToDetailedResults converts basic results to detailed analysis.
func (sv *StatisticalValidator) convertToDetailedResults(basicResults []CaseResult) []DetailedCaseResult {
	detailed := make([]DetailedCaseResult, len(basicResults))

	for i, basic := range basicResults {
		detailed[i] = DetailedCaseResult{
			CaseResult:          basic,
			AbsoluteError:       basic.Error,
			RelativeError:       sv.calculateRelativeError(basic.Expected, basic.Actual),
			SquaredError:        basic.Error * basic.Error,
			IsHighSimilarity:    basic.Expected >= 0.7,
			PredictedHigh:       basic.Actual >= 0.7,
			CorrectlyClassified: sv.isCorrectlyClassified(basic.Expected, basic.Actual),
		}
	}

	return detailed
}

// calculateRelativeError computes relative error with safeguards.
func (sv *StatisticalValidator) calculateRelativeError(expected, actual float64) float64 {
	if expected == 0 {
		return math.Abs(actual) // Special case when expected is 0
	}
	return math.Abs(actual-expected) / expected
}

// isCorrectlyClassified determines if similarity is correctly classified as high/low.
func (sv *StatisticalValidator) isCorrectlyClassified(expected, actual float64) bool {
	threshold := 0.7
	expectedHigh := expected >= threshold
	actualHigh := actual >= threshold
	return expectedHigh == actualHigh
}

// calculateBasicMetrics computes fundamental statistical metrics.
func (sv *StatisticalValidator) calculateBasicMetrics(result *ValidationResult) {
	n := float64(len(result.CaseResults))
	if n == 0 {
		return
	}

	var sumAbsError, sumSquaredError, sumExpected, sumActual float64
	var sumExpectedSquared, sumActualSquared, sumProduct float64

	for _, r := range result.CaseResults {
		sumAbsError += r.AbsoluteError
		sumSquaredError += r.SquaredError
		sumExpected += r.Expected
		sumActual += r.Actual
		sumExpectedSquared += r.Expected * r.Expected
		sumActualSquared += r.Actual * r.Actual
		sumProduct += r.Expected * r.Actual
	}

	// Basic error metrics
	result.MAE = sumAbsError / n
	result.MSE = sumSquaredError / n
	result.RMSE = math.Sqrt(result.MSE)

	// Correlation metrics
	meanExpected := sumExpected / n
	meanActual := sumActual / n

	// Pearson correlation
	var numeratorPearson, denomExpectedPearson, denomActualPearson float64
	for _, r := range result.CaseResults {
		diffExpected := r.Expected - meanExpected
		diffActual := r.Actual - meanActual
		numeratorPearson += diffExpected * diffActual
		denomExpectedPearson += diffExpected * diffExpected
		denomActualPearson += diffActual * diffActual
	}

	if denomExpectedPearson > 0 && denomActualPearson > 0 {
		result.PearsonR = numeratorPearson / math.Sqrt(denomExpectedPearson*denomActualPearson)
	}

	// R² (coefficient of determination)
	var ssRes, ssTot float64
	for _, r := range result.CaseResults {
		ssRes += (r.Expected - r.Actual) * (r.Expected - r.Actual)
		ssTot += (r.Expected - meanExpected) * (r.Expected - meanExpected)
	}

	if ssTot > 0 {
		result.R2 = 1.0 - (ssRes / ssTot)
	}

	// Spearman rank correlation
	result.SpearmanRho = sv.calculateSpearmanRho(result.CaseResults)
}

// calculateSpearmanRho computes Spearman rank correlation coefficient.
func (sv *StatisticalValidator) calculateSpearmanRho(results []DetailedCaseResult) float64 {
	n := len(results)
	if n < 2 {
		return 0
	}

	// Create rank arrays
	expectedRanks := sv.getRanks(results, func(r DetailedCaseResult) float64 { return r.Expected })
	actualRanks := sv.getRanks(results, func(r DetailedCaseResult) float64 { return r.Actual })

	// Calculate Spearman's rho
	var sumSquaredDiffs float64
	for i := range n {
		diff := expectedRanks[i] - actualRanks[i]
		sumSquaredDiffs += diff * diff
	}

	rho := 1.0 - (6.0*sumSquaredDiffs)/(float64(n*(n*n-1)))
	return rho
}

// getRanks converts values to ranks (1-based, average ranks for ties).
func (sv *StatisticalValidator) getRanks(
	results []DetailedCaseResult,
	getValue func(DetailedCaseResult) float64,
) []float64 {
	n := len(results)
	ranks := make([]float64, n)

	// Create pairs of (value, index)
	pairs := make([]struct {
		value float64
		index int
	}, n)

	for i, result := range results {
		pairs[i] = struct {
			value float64
			index int
		}{getValue(result), i}
	}

	// Sort by value
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].value < pairs[j].value
	})

	// Assign ranks (handle ties by averaging)
	for i := 0; i < n; {
		j := i
		// Find end of tied group
		for j < n && pairs[j].value == pairs[i].value {
			j++
		}
		// Assign average rank to tied group
		avgRank := float64(i+j+1) / 2.0 // +1 for 1-based ranking
		for k := i; k < j; k++ {
			ranks[pairs[k].index] = avgRank
		}
		i = j
	}

	return ranks
}

// calculateClassificationMetrics computes precision, recall, F1, and accuracy.
func (sv *StatisticalValidator) calculateClassificationMetrics(result *ValidationResult) {
	var tp, fp, fn, tn int

	for _, r := range result.CaseResults {
		if r.IsHighSimilarity && r.PredictedHigh {
			tp++
		} else if !r.IsHighSimilarity && r.PredictedHigh {
			fp++
		} else if r.IsHighSimilarity && !r.PredictedHigh {
			fn++
		} else {
			tn++
		}
	}

	// Precision = TP / (TP + FP)
	if tp+fp > 0 {
		result.Precision = float64(tp) / float64(tp+fp)
	}

	// Recall = TP / (TP + FN)
	if tp+fn > 0 {
		result.Recall = float64(tp) / float64(tp+fn)
	}

	// F1 Score = 2 * (Precision * Recall) / (Precision + Recall)
	if result.Precision+result.Recall > 0 {
		result.F1Score = 2.0 * result.Precision * result.Recall / (result.Precision + result.Recall)
	}

	// Accuracy = (TP + TN) / (TP + FP + FN + TN)
	total := tp + fp + fn + tn
	if total > 0 {
		result.Accuracy = float64(tp+tn) / float64(total)
	}
}

// calculateDistributionStats analyzes error distribution properties.
func (sv *StatisticalValidator) calculateDistributionStats(result *ValidationResult) {
	errors := make([]float64, len(result.CaseResults))
	var sum float64

	for i, r := range result.CaseResults {
		errors[i] = r.AbsoluteError
		sum += r.AbsoluteError
	}

	if len(errors) == 0 {
		return
	}

	// Sort for percentile calculations
	sortedErrors := make([]float64, len(errors))
	copy(sortedErrors, errors)
	sort.Float64s(sortedErrors)

	n := float64(len(errors))
	mean := sum / n

	// Calculate variance and higher moments
	var variance, skewness, kurtosis float64
	for _, err := range errors {
		diff := err - mean
		variance += diff * diff
		skewness += diff * diff * diff
		kurtosis += diff * diff * diff * diff
	}

	variance /= n
	stdDev := math.Sqrt(variance)

	if stdDev > 0 {
		skewness = (skewness / n) / (stdDev * stdDev * stdDev)
		kurtosis = (kurtosis/n)/(variance*variance) - 3.0 // Excess kurtosis
	}

	result.ErrorDistribution = ErrorDistributionStats{
		Mean:     mean,
		Median:   sv.percentile(sortedErrors, 0.5),
		StdDev:   stdDev,
		Skewness: skewness,
		Kurtosis: kurtosis,
		Q25:      sv.percentile(sortedErrors, 0.25),
		Q75:      sv.percentile(sortedErrors, 0.75),
	}

	result.ErrorDistribution.IQR = result.ErrorDistribution.Q75 - result.ErrorDistribution.Q25
}

// percentile calculates the p-th percentile of sorted data.
func (sv *StatisticalValidator) percentile(sortedData []float64, p float64) float64 {
	if len(sortedData) == 0 {
		return 0
	}
	if len(sortedData) == 1 {
		return sortedData[0]
	}

	n := float64(len(sortedData))
	index := p * (n - 1)

	if index == math.Floor(index) {
		return sortedData[int(index)]
	}

	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	weight := index - math.Floor(index)

	return sortedData[lower]*(1-weight) + sortedData[upper]*weight
}

// calculateCategoryStats computes performance metrics by category.
func (sv *StatisticalValidator) calculateCategoryStats(result *ValidationResult) {
	categoryGroups := make(map[string][]DetailedCaseResult)

	for _, r := range result.CaseResults {
		categoryGroups[r.Category] = append(categoryGroups[r.Category], r)
	}

	result.CategoryPerformance = make(map[string]CategoryStats)

	for category, cases := range categoryGroups {
		stats := sv.calculateStatsForGroup(category, cases)
		result.CategoryPerformance[category] = stats
	}
}

// calculateStatsForGroup computes statistics for a group of cases.
func (sv *StatisticalValidator) calculateStatsForGroup(category string, cases []DetailedCaseResult) CategoryStats {
	if len(cases) == 0 {
		return CategoryStats{Category: category}
	}

	var sumError, sumActual, sumExpected float64
	errors := make([]float64, len(cases))

	bestCase := cases[0]
	worstCase := cases[0]

	for i, c := range cases {
		errors[i] = c.AbsoluteError
		sumError += c.AbsoluteError
		sumActual += c.Actual
		sumExpected += c.Expected

		if c.AbsoluteError < bestCase.AbsoluteError {
			bestCase = c
		}
		if c.AbsoluteError > worstCase.AbsoluteError {
			worstCase = c
		}
	}

	sort.Float64s(errors)

	// Calculate standard deviation
	meanError := sumError / float64(len(cases))
	var variance float64
	for _, err := range errors {
		diff := err - meanError
		variance += diff * diff
	}
	variance /= float64(len(cases))

	return CategoryStats{
		Category:     category,
		Count:        len(cases),
		MeanError:    meanError,
		MedianError:  sv.percentile(errors, 0.5),
		StdDevError:  math.Sqrt(variance),
		MeanActual:   sumActual / float64(len(cases)),
		MeanExpected: sumExpected / float64(len(cases)),
		WorstCase:    worstCase,
		BestCase:     bestCase,
	}
}

// calculateRobustnessMetrics computes advanced robustness metrics.
func (sv *StatisticalValidator) calculateRobustnessMetrics(result *ValidationResult) {
	// Robustness: combination of low error variance and high correlation
	errorVariance := result.ErrorDistribution.StdDev * result.ErrorDistribution.StdDev
	correlationComponent := math.Abs(result.PearsonR)
	result.RobustnessScore = correlationComponent * (1.0 / (1.0 + errorVariance))

	// Consistency: how consistent are predictions for similar expected values
	result.ConsistencyScore = sv.calculateConsistencyScore(result.CaseResults)

	// Discrimination: ability to distinguish between different similarity levels
	result.DiscriminationScore = sv.calculateDiscriminationScore(result.CaseResults)
}

// calculateConsistencyScore measures prediction consistency for similar expected values.
func (sv *StatisticalValidator) calculateConsistencyScore(results []DetailedCaseResult) float64 {
	// Group cases by expected similarity ranges
	ranges := []struct {
		min, max float64
		name     string
	}{
		{0.0, 0.3, "low"},
		{0.3, 0.7, "medium"},
		{0.7, 1.0, "high"},
	}

	var totalVariance, weightedSum float64

	for _, r := range ranges {
		var cases []DetailedCaseResult
		for _, result := range results {
			if result.Expected >= r.min && result.Expected <= r.max {
				cases = append(cases, result)
			}
		}

		if len(cases) >= 2 {
			// Calculate variance in this range
			var sum, sumSquared float64
			for _, c := range cases {
				sum += c.Actual
				sumSquared += c.Actual * c.Actual
			}

			mean := sum / float64(len(cases))
			variance := (sumSquared / float64(len(cases))) - (mean * mean)

			weight := float64(len(cases))
			totalVariance += variance * weight
			weightedSum += weight
		}
	}

	if weightedSum > 0 {
		avgVariance := totalVariance / weightedSum
		return 1.0 / (1.0 + avgVariance) // Higher score for lower variance
	}

	return 0.5 // Default score when insufficient data
}

// calculateDiscriminationScore measures ability to distinguish different similarity levels.
func (sv *StatisticalValidator) calculateDiscriminationScore(results []DetailedCaseResult) float64 {
	// Use ROC-AUC like measure for discrimination
	// Count how often higher expected similarity gets higher predicted similarity

	concordant := 0
	total := 0

	for i := range len(results) {
		for j := i + 1; j < len(results); j++ {
			r1, r2 := results[i], results[j]

			if r1.Expected != r2.Expected { // Only count pairs with different expected values
				total++
				if (r1.Expected > r2.Expected && r1.Actual > r2.Actual) ||
					(r1.Expected < r2.Expected && r1.Actual < r2.Actual) {
					concordant++
				} else if r1.Expected > r2.Expected && r1.Actual == r2.Actual {
					concordant++ // Give partial credit for ties
				}
			}
		}
	}

	if total > 0 {
		return float64(concordant) / float64(total)
	}

	return 0.5 // Random discrimination
}

// PrintValidationReport prints a comprehensive validation report.
func (sv *StatisticalValidator) PrintValidationReport(result ValidationResult) {
	fmt.Printf("\n=== STATISTICAL VALIDATION REPORT ===\n")

	// Basic metrics
	fmt.Printf("\n--- ERROR METRICS ---\n")
	fmt.Printf("Mean Absolute Error (MAE):    %.6f\n", result.MAE)
	fmt.Printf("Mean Squared Error (MSE):     %.6f\n", result.MSE)
	fmt.Printf("Root Mean Squared Error:      %.6f\n", result.RMSE)
	fmt.Printf("R² (Coefficient of Det.):     %.6f\n", result.R2)

	fmt.Printf("\n--- CORRELATION METRICS ---\n")
	fmt.Printf("Pearson Correlation (r):      %.6f\n", result.PearsonR)
	fmt.Printf("Spearman Rank Correlation:    %.6f\n", result.SpearmanRho)

	// Classification metrics
	fmt.Printf("\n--- CLASSIFICATION METRICS ---\n")
	fmt.Printf("Precision (High Sim):         %.6f\n", result.Precision)
	fmt.Printf("Recall (High Sim):            %.6f\n", result.Recall)
	fmt.Printf("F1 Score:                     %.6f\n", result.F1Score)
	fmt.Printf("Accuracy:                     %.6f\n", result.Accuracy)

	// Error distribution
	fmt.Printf("\n--- ERROR DISTRIBUTION ---\n")
	dist := result.ErrorDistribution
	fmt.Printf("Mean Error:                   %.6f\n", dist.Mean)
	fmt.Printf("Median Error:                 %.6f\n", dist.Median)
	fmt.Printf("Std Dev Error:                %.6f\n", dist.StdDev)
	fmt.Printf("Skewness:                     %.6f\n", dist.Skewness)
	fmt.Printf("Kurtosis:                     %.6f\n", dist.Kurtosis)
	fmt.Printf("25th Percentile:              %.6f\n", dist.Q25)
	fmt.Printf("75th Percentile:              %.6f\n", dist.Q75)
	fmt.Printf("IQR:                          %.6f\n", dist.IQR)

	// Robustness metrics
	fmt.Printf("\n--- ROBUSTNESS METRICS ---\n")
	fmt.Printf("Robustness Score:             %.6f\n", result.RobustnessScore)
	fmt.Printf("Consistency Score:            %.6f\n", result.ConsistencyScore)
	fmt.Printf("Discrimination Score:         %.6f\n", result.DiscriminationScore)

	// Category performance
	fmt.Printf("\n--- CATEGORY PERFORMANCE ---\n")
	fmt.Printf("Category        Count  Mean Err  Med Err  Std Err\n")
	for category, stats := range result.CategoryPerformance {
		fmt.Printf("%-14s  %5d  %8.4f  %7.4f  %7.4f\n",
			category, stats.Count, stats.MeanError, stats.MedianError, stats.StdDevError)
	}

	// Worst performing cases
	fmt.Printf("\n--- TOP 3 WORST CASES ---\n")
	sorted := make([]DetailedCaseResult, len(result.CaseResults))
	copy(sorted, result.CaseResults)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].AbsoluteError > sorted[j].AbsoluteError
	})

	for i := 0; i < 3 && i < len(sorted); i++ {
		caseResult := sorted[i]
		fmt.Printf("%d. %s (%s)\n", i+1, caseResult.CaseName, caseResult.Category)
		fmt.Printf("   Expected: %.3f, Actual: %.3f, Error: %.3f\n",
			caseResult.Expected, caseResult.Actual, caseResult.AbsoluteError)
	}
}
