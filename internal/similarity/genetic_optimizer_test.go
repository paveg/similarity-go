package similarity

import (
	"math"
	"testing"

	"github.com/paveg/similarity-go/internal/config"
)

func TestGeneticOptimizer_NewGeneticOptimizer(t *testing.T) {
	optimizer := NewGeneticOptimizer()

	if optimizer == nil {
		t.Fatal("Expected non-nil optimizer")
	}

	// Check default parameters
	if optimizer.populationSize <= 0 {
		t.Error("Expected positive population size")
	}
	if optimizer.generations <= 0 {
		t.Error("Expected positive generations")
	}
	if optimizer.mutationRate <= 0 || optimizer.mutationRate >= 1 {
		t.Error("Expected mutation rate in (0, 1)")
	}
	if optimizer.crossoverRate <= 0 || optimizer.crossoverRate >= 1 {
		t.Error("Expected crossover rate in (0, 1)")
	}
	if optimizer.eliteSize <= 0 {
		t.Error("Expected positive elite size")
	}
	if len(optimizer.dataset) == 0 {
		t.Error("Expected non-empty dataset")
	}
}

func TestGeneticOptimizer_SetParameters(t *testing.T) {
	optimizer := NewGeneticOptimizer()

	// Test parameter setting
	optimizer.SetParameters(100, 200, 0.15, 0.85, 10)

	if optimizer.populationSize != 100 {
		t.Errorf("Expected population size 100, got %d", optimizer.populationSize)
	}
	if optimizer.generations != 200 {
		t.Errorf("Expected generations 200, got %d", optimizer.generations)
	}
	if optimizer.mutationRate != 0.15 {
		t.Errorf("Expected mutation rate 0.15, got %f", optimizer.mutationRate)
	}
	if optimizer.crossoverRate != 0.85 {
		t.Errorf("Expected crossover rate 0.85, got %f", optimizer.crossoverRate)
	}
	if optimizer.eliteSize != 10 {
		t.Errorf("Expected elite size 10, got %d", optimizer.eliteSize)
	}
}

func TestGeneticOptimizer_generateRandomWeights(t *testing.T) {
	optimizer := NewGeneticOptimizer()

	// Generate multiple random weights to test consistency
	for range 10 {
		weights := optimizer.generateRandomWeights()

		// Check weights are positive
		if weights.TreeEdit <= 0 || weights.TokenSimilarity <= 0 ||
			weights.Structural <= 0 || weights.Signature <= 0 {
			t.Error("All weights should be positive")
		}

		// Check weights sum approximately to 1.0
		total := weights.TreeEdit + weights.TokenSimilarity + weights.Structural + weights.Signature
		if math.Abs(total-1.0) > 0.001 {
			t.Errorf("Weights don't sum to 1.0: %f", total)
		}

		// Check penalty weight is constant
		if weights.DifferentSignature != 0.3 {
			t.Errorf("Expected penalty weight 0.3, got %f", weights.DifferentSignature)
		}

		// Check weights are within expected ranges
		if weights.TreeEdit > 0.5 || weights.TreeEdit < 0.1 {
			t.Errorf("TreeEdit weight out of expected range: %f", weights.TreeEdit)
		}
		if weights.TokenSimilarity > 0.5 || weights.TokenSimilarity < 0.1 {
			t.Errorf("TokenSimilarity weight out of expected range: %f", weights.TokenSimilarity)
		}
		if weights.Structural > 0.4 || weights.Structural < 0.1 {
			t.Errorf("Structural weight out of expected range: %f", weights.Structural)
		}
		if weights.Signature > 0.3 || weights.Signature < 0.05 {
			t.Errorf("Signature weight out of expected range: %f", weights.Signature)
		}
	}
}

func TestGeneticOptimizer_initializePopulation(t *testing.T) {
	optimizer := NewGeneticOptimizer()
	optimizer.SetParameters(20, 50, 0.1, 0.8, 3)

	population := optimizer.initializePopulation()

	if len(population) != optimizer.populationSize {
		t.Errorf("Expected population size %d, got %d", optimizer.populationSize, len(population))
	}

	// Check each individual
	for i, individual := range population {
		if individual.Age != 0 {
			t.Errorf("Individual %d should start with age 0, got %d", i, individual.Age)
		}

		// Check weights are valid
		weights := individual.Weights
		if weights.TreeEdit <= 0 || weights.TokenSimilarity <= 0 ||
			weights.Structural <= 0 || weights.Signature <= 0 {
			t.Errorf("Individual %d has invalid weights", i)
		}

		total := weights.TreeEdit + weights.TokenSimilarity + weights.Structural + weights.Signature
		if math.Abs(total-1.0) > 0.001 {
			t.Errorf("Individual %d weights don't sum to 1.0: %f", i, total)
		}
	}
}

func TestGeneticOptimizer_calculateWeightDistance(t *testing.T) {
	optimizer := NewGeneticOptimizer()

	w1 := config.SimilarityWeights{
		TreeEdit:        0.3,
		TokenSimilarity: 0.3,
		Structural:      0.25,
		Signature:       0.15,
	}

	w2 := config.SimilarityWeights{
		TreeEdit:        0.35,
		TokenSimilarity: 0.25,
		Structural:      0.25,
		Signature:       0.15,
	}

	distance := optimizer.calculateWeightDistance(w1, w2)

	// Should be > 0 for different weights
	if distance <= 0 {
		t.Error("Distance should be positive for different weights")
	}

	// Distance to self should be 0
	selfDistance := optimizer.calculateWeightDistance(w1, w1)
	if math.Abs(selfDistance) > 1e-10 {
		t.Errorf("Self-distance should be 0, got %f", selfDistance)
	}

	// Distance should be symmetric
	reverseDistance := optimizer.calculateWeightDistance(w2, w1)
	if math.Abs(distance-reverseDistance) > 1e-10 {
		t.Error("Distance should be symmetric")
	}
}

func TestGeneticOptimizer_crossover(t *testing.T) {
	optimizer := NewGeneticOptimizer()

	parent1 := Individual{
		Weights: config.SimilarityWeights{
			TreeEdit:        0.4,
			TokenSimilarity: 0.3,
			Structural:      0.2,
			Signature:       0.1,
		},
		Age: 5,
	}

	parent2 := Individual{
		Weights: config.SimilarityWeights{
			TreeEdit:        0.2,
			TokenSimilarity: 0.4,
			Structural:      0.25,
			Signature:       0.15,
		},
		Age: 3,
	}

	// Perform multiple crossovers to test consistency
	for range 10 {
		child := optimizer.crossover(parent1, parent2)

		// Check child age is reset
		if child.Age != 0 {
			t.Error("Child age should be 0")
		}

		// Check weights are positive
		if child.Weights.TreeEdit <= 0 || child.Weights.TokenSimilarity <= 0 ||
			child.Weights.Structural <= 0 || child.Weights.Signature <= 0 {
			t.Error("Child weights should be positive")
		}

		// Check weights sum to 1.0
		total := child.Weights.TreeEdit + child.Weights.TokenSimilarity +
			child.Weights.Structural + child.Weights.Signature
		if math.Abs(total-1.0) > 0.001 {
			t.Errorf("Child weights don't sum to 1.0: %f", total)
		}

		// Check penalty weight is preserved
		if child.Weights.DifferentSignature != 0.3 {
			t.Error("Child should preserve penalty weight")
		}

		// Child weights should be between parent weights
		for j := range 4 {
			var childWeight, parent1Weight, parent2Weight float64
			switch j {
			case 0:
				childWeight = child.Weights.TreeEdit
				parent1Weight = parent1.Weights.TreeEdit
				parent2Weight = parent2.Weights.TreeEdit
			case 1:
				childWeight = child.Weights.TokenSimilarity
				parent1Weight = parent1.Weights.TokenSimilarity
				parent2Weight = parent2.Weights.TokenSimilarity
			case 2:
				childWeight = child.Weights.Structural
				parent1Weight = parent1.Weights.Structural
				parent2Weight = parent2.Weights.Structural
			case 3:
				childWeight = child.Weights.Signature
				parent1Weight = parent1.Weights.Signature
				parent2Weight = parent2.Weights.Signature
			}

			minParent := math.Min(parent1Weight, parent2Weight)
			maxParent := math.Max(parent1Weight, parent2Weight)

			// Allow some tolerance for normalization effects
			if childWeight < minParent*0.8 || childWeight > maxParent*1.2 {
				t.Errorf("Child weight %d (%f) not reasonably between parents [%f, %f]",
					j, childWeight, minParent, maxParent)
			}
		}
	}
}

func TestGeneticOptimizer_mutate(t *testing.T) {
	optimizer := NewGeneticOptimizer()

	individual := Individual{
		Weights: config.SimilarityWeights{
			TreeEdit:        0.3,
			TokenSimilarity: 0.3,
			Structural:      0.25,
			Signature:       0.15,
		},
		Age: 2,
	}

	// Perform multiple mutations to test consistency
	for range 10 {
		mutated := optimizer.mutate(individual)

		// Check weights are positive
		if mutated.Weights.TreeEdit <= 0 || mutated.Weights.TokenSimilarity <= 0 ||
			mutated.Weights.Structural <= 0 || mutated.Weights.Signature <= 0 {
			t.Error("Mutated weights should be positive")
		}

		// Check weights sum to 1.0
		total := mutated.Weights.TreeEdit + mutated.Weights.TokenSimilarity +
			mutated.Weights.Structural + mutated.Weights.Signature
		if math.Abs(total-1.0) > 0.001 {
			t.Errorf("Mutated weights don't sum to 1.0: %f", total)
		}

		// Check penalty weight is preserved
		if mutated.Weights.DifferentSignature != 0.3 {
			t.Error("Mutated individual should preserve penalty weight")
		}

		// Age should be preserved
		if mutated.Age != individual.Age {
			t.Error("Mutation should preserve age")
		}
	}
}

func TestGeneticOptimizer_tournamentSelection(t *testing.T) {
	optimizer := NewGeneticOptimizer()

	// Create population with known fitness values
	population := []Individual{
		{Fitness: 0.9, Age: 0},
		{Fitness: 0.8, Age: 1},
		{Fitness: 0.7, Age: 2},
		{Fitness: 0.6, Age: 3},
		{Fitness: 0.5, Age: 4},
	}

	// Run tournament selection multiple times
	bestCount := 0
	totalSelections := 100

	for range totalSelections {
		selected := optimizer.tournamentSelection(population, 3)
		if selected.Fitness == 0.9 {
			bestCount++
		}
	}

	// Best individual should be selected more often
	if bestCount < totalSelections/3 {
		t.Errorf("Best individual selected too rarely: %d/%d", bestCount, totalSelections)
	}

	// Test with tournament size larger than population
	selected := optimizer.tournamentSelection(population, 10)
	if selected.Fitness != 0.9 {
		t.Error("Should always select best with large tournament size")
	}
}

func TestGeneticOptimizer_OptimizeWeights_SmallScale(t *testing.T) {
	optimizer := NewGeneticOptimizer()

	// Use small parameters for faster testing
	optimizer.SetParameters(10, 5, 0.2, 0.8, 2)

	result := optimizer.OptimizeWeights(t)

	// Validate result structure
	if result.BestIndividual.Fitness < 0 || result.BestIndividual.Fitness > 1 {
		t.Errorf("Best fitness out of range: %f", result.BestIndividual.Fitness)
	}

	if len(result.FinalPopulation) != 10 {
		t.Errorf("Expected final population size 10, got %d", len(result.FinalPopulation))
	}

	if len(result.GenerationHistory) == 0 {
		t.Error("Expected non-empty generation history")
	}

	if result.TotalEvaluations <= 0 {
		t.Error("Expected positive evaluation count")
	}

	// Validate best weights
	weights := result.BestIndividual.Weights
	if weights.TreeEdit <= 0 || weights.TokenSimilarity <= 0 ||
		weights.Structural <= 0 || weights.Signature <= 0 {
		t.Error("Best weights should be positive")
	}

	total := weights.TreeEdit + weights.TokenSimilarity + weights.Structural + weights.Signature
	if math.Abs(total-1.0) > 0.01 {
		t.Errorf("Best weights don't sum to 1.0: %f", total)
	}

	// Check generation history
	for i, gen := range result.GenerationHistory {
		if gen.Generation != i {
			t.Errorf("Generation %d has wrong generation number: %d", i, gen.Generation)
		}
		if gen.BestFitness < 0 || gen.BestFitness > 1 {
			t.Errorf("Generation %d best fitness out of range: %f", i, gen.BestFitness)
		}
		if gen.AvgFitness < 0 || gen.AvgFitness > 1 {
			t.Errorf("Generation %d avg fitness out of range: %f", i, gen.AvgFitness)
		}
	}
}

func TestGeneticOptimizer_calculatePopulationDiversity(t *testing.T) {
	optimizer := NewGeneticOptimizer()

	// Test with identical population (zero diversity)
	identicalPop := []Individual{
		{Weights: config.SimilarityWeights{TreeEdit: 0.25, TokenSimilarity: 0.25, Structural: 0.25, Signature: 0.25}},
		{Weights: config.SimilarityWeights{TreeEdit: 0.25, TokenSimilarity: 0.25, Structural: 0.25, Signature: 0.25}},
	}

	diversity := optimizer.calculatePopulationDiversity(identicalPop)
	if diversity != 0.0 {
		t.Errorf("Expected zero diversity for identical population, got %f", diversity)
	}

	// Test with diverse population
	diversePop := []Individual{
		{Weights: config.SimilarityWeights{TreeEdit: 0.5, TokenSimilarity: 0.2, Structural: 0.2, Signature: 0.1}},
		{Weights: config.SimilarityWeights{TreeEdit: 0.1, TokenSimilarity: 0.5, Structural: 0.2, Signature: 0.2}},
		{Weights: config.SimilarityWeights{TreeEdit: 0.2, TokenSimilarity: 0.2, Structural: 0.5, Signature: 0.1}},
	}

	diversity = optimizer.calculatePopulationDiversity(diversePop)
	if diversity <= 0.0 {
		t.Error("Expected positive diversity for diverse population")
	}

	// Test edge cases
	emptyPop := []Individual{}
	diversity = optimizer.calculatePopulationDiversity(emptyPop)
	if diversity != 0.0 {
		t.Error("Expected zero diversity for empty population")
	}

	singlePop := []Individual{{}}
	diversity = optimizer.calculatePopulationDiversity(singlePop)
	if diversity != 0.0 {
		t.Error("Expected zero diversity for single individual")
	}
}

func BenchmarkGeneticOptimizer_OptimizeWeights(b *testing.B) {
	optimizer := NewGeneticOptimizer()
	optimizer.SetParameters(20, 10, 0.1, 0.8, 3) // Small parameters for benchmarking

	// Create a dummy test for the benchmark
	t := &testing.T{}

	b.ResetTimer()
	for range b.N {
		optimizer.OptimizeWeights(t)
	}
}

func TestGeneticOptimizer_EvolutionProgress(t *testing.T) {
	optimizer := NewGeneticOptimizer()
	optimizer.SetParameters(15, 8, 0.15, 0.85, 3)

	result := optimizer.OptimizeWeights(t)

	// Check that fitness generally improves over generations
	if len(result.GenerationHistory) < 2 {
		t.Skip("Need at least 2 generations to test progress")
	}

	firstGenBest := result.GenerationHistory[0].BestFitness
	lastGenBest := result.GenerationHistory[len(result.GenerationHistory)-1].BestFitness

	// Fitness should not significantly decrease (allowing for some stochastic variation)
	if lastGenBest < firstGenBest-0.1 {
		t.Errorf("Fitness significantly decreased from %f to %f", firstGenBest, lastGenBest)
	}

	// Check that generation numbers are sequential
	for i, gen := range result.GenerationHistory {
		if gen.Generation != i {
			t.Errorf("Generation %d has incorrect generation number: %d", i, gen.Generation)
		}
	}
}
