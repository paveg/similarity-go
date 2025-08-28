package worker

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/similarity"
)

const (
	// MinFunctionCountForComparison is the minimum number of functions needed for comparison.
	MinFunctionCountForComparison = 2
)

// ComparisonJob represents a pair of functions to compare.
type ComparisonJob struct {
	Function1 *ast.Function
	Function2 *ast.Function
	Index1    int
	Index2    int
}

// ComparisonResult represents the result of a similarity comparison.
type ComparisonResult struct {
	Match     *similarity.Match // Pointer to match, nil if below threshold
	Error     error
	Index1    int
	Index2    int
	Completed bool // Indicates completion of a comparison
}

// SimilarityWorker handles parallel similarity calculations.
type SimilarityWorker struct {
	detector  *similarity.Detector
	workers   int
	threshold float64
	matchesCh chan *similarity.Match
	resultsCh chan ComparisonResult
}

// NewSimilarityWorker creates a new similarity worker with the specified detector and worker count.
func NewSimilarityWorker(detector *similarity.Detector, workers int, threshold float64) *SimilarityWorker {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	return &SimilarityWorker{
		detector:  detector,
		workers:   workers,
		threshold: threshold,
		matchesCh: make(chan *similarity.Match, workers*ChannelBufferMultiplier),
		resultsCh: make(chan ComparisonResult, workers*ChannelBufferMultiplier),
	}
}

// FindSimilarFunctions finds similar functions using parallel processing.
func (sw *SimilarityWorker) FindSimilarFunctions(
	functions []*ast.Function,
	progressCallback func(completed, total int),
) ([]similarity.Match, error) {
	if len(functions) < MinFunctionCountForComparison {
		return nil, nil
	}

	// Calculate total number of comparisons
	totalComparisons := len(functions) * (len(functions) - 1) / MinFunctionCountForComparison

	// Create jobs for all function pairs
	jobs := make(chan ComparisonJob, totalComparisons)
	for i := range functions {
		for j := i + 1; j < len(functions); j++ {
			jobs <- ComparisonJob{
				Function1: functions[i],
				Function2: functions[j],
				Index1:    i,
				Index2:    j,
			}
		}
	}
	close(jobs) // No more jobs will be sent

	// Start workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start worker goroutines
	for range sw.workers {
		wg.Add(1)
		go sw.worker(ctx, jobs, &wg)
	}

	// Collect results
	var matches []similarity.Match
	var errors []error
	completed := 0

	// Start a goroutine to close results channel when all workers are done
	go func() {
		wg.Wait()
		close(sw.resultsCh)
	}()

	// Collect results as they come in
	for result := range sw.resultsCh {
		completed++

		if progressCallback != nil {
			progressCallback(completed, totalComparisons)
		}

		if result.Error != nil {
			errors = append(errors, result.Error)
		} else if result.Match != nil {
			matches = append(matches, *result.Match)
		}
	}

	// Return any errors that occurred
	if len(errors) > 0 {
		return matches, fmt.Errorf("encountered %d errors during similarity calculation", len(errors))
	}

	return matches, nil
}

// worker is the main worker goroutine that processes comparison jobs.
func (sw *SimilarityWorker) worker(ctx context.Context, jobs <-chan ComparisonJob, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			// Context cancelled, send error and return
			sw.resultsCh <- ComparisonResult{
				Error:     ctx.Err(),
				Index1:    job.Index1,
				Index2:    job.Index2,
				Completed: false,
			}
			return
		default:
		}

		// Process the comparison
		result := sw.processComparison(job)

		// Send result
		select {
		case sw.resultsCh <- result:
		case <-ctx.Done():
			return
		}
	}
}

// processComparison processes a single function comparison.
func (sw *SimilarityWorker) processComparison(job ComparisonJob) ComparisonResult {
	// Calculate similarity
	similarityScore := sw.detector.CalculateSimilarity(job.Function1, job.Function2)

	result := ComparisonResult{
		Index1:    job.Index1,
		Index2:    job.Index2,
		Completed: true,
	}

	// If above threshold, create match
	if sw.detector.IsAboveThreshold(similarityScore) {
		match := &similarity.Match{
			Function1:  job.Function1,
			Function2:  job.Function2,
			Similarity: similarityScore,
		}
		result.Match = match
	}

	return result
}

// WorkerCount returns the number of workers in the pool.
func (sw *SimilarityWorker) WorkerCount() int {
	return sw.workers
}
