package similarity

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/paveg/similarity-go/internal/config"
)

const (
	percentageMultiplier     = 100.0
	defaultPopulationSize    = 50
	defaultGenerations       = 100
	defaultMutationRate      = 0.1
	defaultCrossoverRate     = 0.8
	defaultEliteSize         = 5
	stagnationEpsilon        = 1e-6
	stagnationLimit          = 20
	treeEditRange            = 0.4
	treeEditOffset           = 0.1
	tokenSimilarityRange     = 0.4
	tokenSimilarityOffset    = 0.1
	structuralRange          = 0.3
	structuralOffset         = 0.1
	signatureRange           = 0.25
	signatureOffset          = 0.05
	mutationSigmaBase        = 0.05
	mutationAgeFactor        = 0.1
	mutationProbability      = 0.3
	minWeightThreshold       = 0.01
	tournamentSize           = 3
	minimumPopulationForEval = 2
	crossoverMixThreshold    = 0.5
)

// GeneticOptimizer implements genetic algorithm for weight optimization.
type GeneticOptimizer struct {
	dataset        []BenchmarkCase
	populationSize int
	generations    int
	mutationRate   float64
	crossoverRate  float64
	eliteSize      int
	random         *rand.Rand
}

// Individual represents a candidate solution in the genetic algorithm.
type Individual struct {
	Weights config.SimilarityWeights
	Fitness float64
	Age     int // Number of generations survived
}

// GeneticResult contains the results of genetic algorithm optimization.
type GeneticResult struct {
	BestIndividual    Individual
	FinalPopulation   []Individual
	GenerationHistory []GenerationStats
	TotalEvaluations  int
	ConvergenceGen    int // Generation where algorithm converged
}

// GenerationStats tracks statistics for each generation.
type GenerationStats struct {
	Generation   int
	BestFitness  float64
	AvgFitness   float64
	WorstFitness float64
	Diversity    float64 // Population diversity metric
}

// NewGeneticOptimizer creates a new genetic algorithm optimizer.
func NewGeneticOptimizer() *GeneticOptimizer {
	return &GeneticOptimizer{
		dataset:        GetBenchmarkDataset(),
		populationSize: defaultPopulationSize,
		generations:    defaultGenerations,
		mutationRate:   defaultMutationRate,
		crossoverRate:  defaultCrossoverRate,
		eliteSize:      defaultEliteSize,
		random:         newPseudoRandomGenerator(),
	}
}

//nolint:gosec // pseudo-random generator is sufficient for heuristic optimization.
func newPseudoRandomGenerator() *rand.Rand {
	seed := time.Now().UnixNano()
	return rand.New(rand.NewPCG(uint64(seed), uint64(seed>>1)))
}

// SetParameters allows customizing genetic algorithm parameters.
func (g *GeneticOptimizer) SetParameters(
	populationSize, generations int,
	mutationRate, crossoverRate float64,
	eliteSize int,
) {
	g.populationSize = populationSize
	g.generations = generations
	g.mutationRate = mutationRate
	g.crossoverRate = crossoverRate
	g.eliteSize = eliteSize
}

// OptimizeWeights runs the genetic algorithm to find optimal weights.
func (g *GeneticOptimizer) OptimizeWeights(t *testing.T) GeneticResult {
	// Initialize population
	population := g.initializePopulation()

	// Evaluate initial population
	g.evaluatePopulation(t, population)

	var history []GenerationStats
	totalEvaluations := len(population)
	convergenceGen := -1
	lastBestFitness := -1.0
	stagnationCount := 0

	for generation := range g.generations {
		// Record generation statistics
		stats := g.calculateGenerationStats(generation, population)
		history = append(history, stats)

		// Check for convergence (no improvement for 10 generations)
		if math.Abs(stats.BestFitness-lastBestFitness) < stagnationEpsilon {
			stagnationCount++
		} else {
			stagnationCount = 0
			lastBestFitness = stats.BestFitness
		}

		if stagnationCount >= 10 && convergenceGen == -1 {
			convergenceGen = generation
		}

		// Early termination if converged for too long
		if stagnationCount >= stagnationLimit {
			break
		}

		// Create next generation
		newPopulation := g.evolvePopulation(t, population)
		totalEvaluations += len(newPopulation) - g.eliteSize // Elite carry over without re-evaluation
		population = newPopulation

		// Age individuals
		for i := range population {
			population[i].Age++
		}
	}

	// Sort final population by fitness
	sort.Slice(population, func(i, j int) bool {
		return population[i].Fitness > population[j].Fitness
	})

	return GeneticResult{
		BestIndividual:    population[0],
		FinalPopulation:   population,
		GenerationHistory: history,
		TotalEvaluations:  totalEvaluations,
		ConvergenceGen:    convergenceGen,
	}
}

// initializePopulation creates the initial random population.
func (g *GeneticOptimizer) initializePopulation() []Individual {
	population := make([]Individual, g.populationSize)

	for i := range population {
		population[i] = Individual{
			Weights: g.generateRandomWeights(),
			Age:     0,
		}
	}

	return population
}

// generateRandomWeights creates random weights that sum to approximately 1.0.
func (g *GeneticOptimizer) generateRandomWeights() config.SimilarityWeights {
	// Generate random values
	treeEdit := g.random.Float64()*treeEditRange + treeEditOffset
	tokenSim := g.random.Float64()*tokenSimilarityRange + tokenSimilarityOffset
	structural := g.random.Float64()*structuralRange + structuralOffset
	signature := g.random.Float64()*signatureRange + signatureOffset

	// Normalize to sum to 1.0
	total := treeEdit + tokenSim + structural + signature
	treeEdit /= total
	tokenSim /= total
	structural /= total
	signature /= total

	return config.SimilarityWeights{
		TreeEdit:           treeEdit,
		TokenSimilarity:    tokenSim,
		Structural:         structural,
		Signature:          signature,
		DifferentSignature: config.DifferentSignatureWeight,
	}
}

// evaluatePopulation calculates fitness for all individuals in the population.
func (g *GeneticOptimizer) evaluatePopulation(t *testing.T, population []Individual) {
	optimizer := &WeightOptimizer{dataset: g.dataset}

	for i := range population {
		score, _ := optimizer.EvaluateWeights(t, population[i].Weights)
		population[i].Fitness = score
	}
}

// calculateGenerationStats computes statistics for the current generation.
func (g *GeneticOptimizer) calculateGenerationStats(generation int, population []Individual) GenerationStats {
	if len(population) == 0 {
		return GenerationStats{}
	}

	var totalFitness float64
	bestFitness := population[0].Fitness
	worstFitness := population[0].Fitness

	for _, individual := range population {
		totalFitness += individual.Fitness
		if individual.Fitness > bestFitness {
			bestFitness = individual.Fitness
		}
		if individual.Fitness < worstFitness {
			worstFitness = individual.Fitness
		}
	}

	avgFitness := totalFitness / float64(len(population))
	diversity := g.calculatePopulationDiversity(population)

	return GenerationStats{
		Generation:   generation,
		BestFitness:  bestFitness,
		AvgFitness:   avgFitness,
		WorstFitness: worstFitness,
		Diversity:    diversity,
	}
}

// calculatePopulationDiversity measures genetic diversity in the population.
func (g *GeneticOptimizer) calculatePopulationDiversity(population []Individual) float64 {
	if len(population) < minimumPopulationForEval {
		return 0.0
	}

	var totalDistance float64
	comparisons := 0

	for i := range population {
		for j := i + 1; j < len(population); j++ {
			distance := g.calculateWeightDistance(population[i].Weights, population[j].Weights)
			totalDistance += distance
			comparisons++
		}
	}

	return totalDistance / float64(comparisons)
}

// calculateWeightDistance computes Euclidean distance between two weight vectors.
func (g *GeneticOptimizer) calculateWeightDistance(w1, w2 config.SimilarityWeights) float64 {
	diff1 := w1.TreeEdit - w2.TreeEdit
	diff2 := w1.TokenSimilarity - w2.TokenSimilarity
	diff3 := w1.Structural - w2.Structural
	diff4 := w1.Signature - w2.Signature

	return math.Sqrt(diff1*diff1 + diff2*diff2 + diff3*diff3 + diff4*diff4)
}

// evolvePopulation creates the next generation through selection, crossover, and mutation.
func (g *GeneticOptimizer) evolvePopulation(t *testing.T, population []Individual) []Individual {
	// Sort by fitness (descending)
	sort.Slice(population, func(i, j int) bool {
		return population[i].Fitness > population[j].Fitness
	})

	nextGeneration := make([]Individual, 0, g.populationSize)

	// Elite selection - keep best individuals
	for i := 0; i < g.eliteSize && i < len(population); i++ {
		nextGeneration = append(nextGeneration, population[i])
	}

	// Generate offspring through crossover and mutation
	for len(nextGeneration) < g.populationSize {
		// Tournament selection
		parent1 := g.tournamentSelection(population, tournamentSize)
		parent2 := g.tournamentSelection(population, tournamentSize)

		var child Individual

		// Crossover
		if g.random.Float64() < g.crossoverRate {
			child = g.crossover(parent1, parent2)
		} else {
			// If no crossover, randomly pick one parent
			if g.random.Float64() < crossoverMixThreshold {
				child = parent1
			} else {
				child = parent2
			}
		}

		// Mutation
		if g.random.Float64() < g.mutationRate {
			child = g.mutate(child)
		}

		// Reset age for new individuals
		child.Age = 0
		nextGeneration = append(nextGeneration, child)
	}

	// Evaluate new individuals (skip elite)
	optimizer := &WeightOptimizer{dataset: g.dataset}
	for i := g.eliteSize; i < len(nextGeneration); i++ {
		score, _ := optimizer.EvaluateWeights(t, nextGeneration[i].Weights)
		nextGeneration[i].Fitness = score
	}

	return nextGeneration
}

// tournamentSelection selects an individual using tournament selection.
func (g *GeneticOptimizer) tournamentSelection(population []Individual, tournamentSize int) Individual {
	if tournamentSize > len(population) {
		tournamentSize = len(population)
	}

	if tournamentSize == len(population) {
		best := population[0]
		for _, candidate := range population[1:] {
			if candidate.Fitness > best.Fitness {
				best = candidate
			}
		}
		return best
	}

	best := population[g.random.IntN(len(population))]

	for i := 1; i < tournamentSize; i++ {
		candidate := population[g.random.IntN(len(population))]
		if candidate.Fitness > best.Fitness {
			best = candidate
		}
	}

	return best
}

// crossover combines two parents to create an offspring.
func (g *GeneticOptimizer) crossover(parent1, parent2 Individual) Individual {
	// Arithmetic crossover with random weight
	alpha := g.random.Float64()

	child := Individual{
		Weights: config.SimilarityWeights{
			TreeEdit:           alpha*parent1.Weights.TreeEdit + (1-alpha)*parent2.Weights.TreeEdit,
			TokenSimilarity:    alpha*parent1.Weights.TokenSimilarity + (1-alpha)*parent2.Weights.TokenSimilarity,
			Structural:         alpha*parent1.Weights.Structural + (1-alpha)*parent2.Weights.Structural,
			Signature:          alpha*parent1.Weights.Signature + (1-alpha)*parent2.Weights.Signature,
			DifferentSignature: config.DifferentSignatureWeight, // Keep penalty constant
		},
		Age: 0,
	}

	// Normalize weights to sum to 1.0
	total := child.Weights.TreeEdit + child.Weights.TokenSimilarity +
		child.Weights.Structural + child.Weights.Signature

	if total > 0 {
		child.Weights.TreeEdit /= total
		child.Weights.TokenSimilarity /= total
		child.Weights.Structural /= total
		child.Weights.Signature /= total
	}

	return child
}

// mutate applies random mutations to an individual.
func (g *GeneticOptimizer) mutate(individual Individual) Individual {
	mutated := individual

	// Gaussian mutation with adaptive step size
	sigma := mutationSigmaBase * (1.0 + mutationAgeFactor*float64(individual.Age)) // Smaller mutations for older individuals

	// Apply mutations to each weight
	weights := []float64{
		mutated.Weights.TreeEdit,
		mutated.Weights.TokenSimilarity,
		mutated.Weights.Structural,
		mutated.Weights.Signature,
	}

	for i := range weights {
		if g.random.Float64() < mutationProbability {
			mutation := g.random.NormFloat64() * sigma
			weights[i] += mutation

			// Ensure positive weights
			if weights[i] < minWeightThreshold {
				weights[i] = minWeightThreshold
			}
		}
	}

	// Normalize to sum to 1.0
	total := weights[0] + weights[1] + weights[2] + weights[3]
	if total > 0 {
		for i := range weights {
			weights[i] /= total
		}
	}

	mutated.Weights = config.SimilarityWeights{
		TreeEdit:           weights[0],
		TokenSimilarity:    weights[1],
		Structural:         weights[2],
		Signature:          weights[3],
		DifferentSignature: config.DifferentSignatureWeight,
	}

	return mutated
}

// PrintGeneticReport prints a detailed report of genetic algorithm results.
func (g *GeneticOptimizer) PrintGeneticReport(result GeneticResult, baselineScore float64) {
	out := os.Stdout
	write := func(format string, args ...any) {
		_, _ = fmt.Fprintf(out, format, args...)
	}

	write("\n=== GENETIC ALGORITHM OPTIMIZATION REPORT ===\n")
	write("Population Size: %d\n", g.populationSize)
	write("Generations Run: %d\n", len(result.GenerationHistory))
	write("Total Evaluations: %d\n", result.TotalEvaluations)
	write("Convergence Generation: %d\n", result.ConvergenceGen)

	best := result.BestIndividual
	write("\nBaseline Score: %.6f\n", baselineScore)
	write("Best GA Score: %.6f\n", best.Fitness)
	improvement := best.Fitness - baselineScore
	write("Improvement: %.6f (%.2f%%)\n", improvement, improvement/baselineScore*percentageMultiplier)

	write("\n--- OPTIMIZED WEIGHTS ---\n")
	write("TreeEdit:        %.4f\n", best.Weights.TreeEdit)
	write("TokenSimilarity: %.4f\n", best.Weights.TokenSimilarity)
	write("Structural:      %.4f\n", best.Weights.Structural)
	write("Signature:       %.4f\n", best.Weights.Signature)
	write("Weight Sum:      %.4f\n",
		best.Weights.TreeEdit+best.Weights.TokenSimilarity+best.Weights.Structural+best.Weights.Signature)

	write("\n--- EVOLUTION PROGRESS ---\n")
	write("Generation  Best Score  Avg Score   Diversity\n")

	for i, gen := range result.GenerationHistory {
		if i%10 == 0 || i == len(result.GenerationHistory)-1 {
			write("%9d   %9.6f   %8.6f   %8.6f\n",
				gen.Generation, gen.BestFitness, gen.AvgFitness, gen.Diversity)
		}
	}

	// Population diversity analysis
	if len(result.FinalPopulation) > 1 {
		finalDiversity := g.calculatePopulationDiversity(result.FinalPopulation)
		write("\nFinal Population Diversity: %.6f\n", finalDiversity)

		// Show top 5 individuals
		write("\n--- TOP 5 SOLUTIONS ---\n")
		sort.Slice(result.FinalPopulation, func(i, j int) bool {
			return result.FinalPopulation[i].Fitness > result.FinalPopulation[j].Fitness
		})

		for i := 0; i < 5 && i < len(result.FinalPopulation); i++ {
			ind := result.FinalPopulation[i]
			write("%d. Score: %.6f, Age: %d\n", i+1, ind.Fitness, ind.Age)
			write("   Weights: [%.3f, %.3f, %.3f, %.3f]\n",
				ind.Weights.TreeEdit, ind.Weights.TokenSimilarity,
				ind.Weights.Structural, ind.Weights.Signature)
		}
	}
}
