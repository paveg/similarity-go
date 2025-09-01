package similarity

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/paveg/similarity-go/internal/ast"
)

// DefaultParallelProcessor implements ParallelProcessor with goroutine pool and context cancellation.
type DefaultParallelProcessor struct {
	detector    *Detector
	workerCount int
	ctx         context.Context
}

// NewDefaultParallelProcessor creates a new parallel processor with the given detector and worker count.
func NewDefaultParallelProcessor(detector *Detector, workerCount int) *DefaultParallelProcessor {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	return &DefaultParallelProcessor{
		detector:    detector,
		workerCount: workerCount,
		ctx:         context.Background(),
	}
}

// NewDefaultParallelProcessorWithContext creates a processor with custom context for cancellation.
func NewDefaultParallelProcessorWithContext(
	ctx context.Context,
	detector *Detector,
	workerCount int,
) *DefaultParallelProcessor {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
	}

	return &DefaultParallelProcessor{
		detector:    detector,
		workerCount: workerCount,
		ctx:         ctx,
	}
}

// WorkItem represents a pair of functions to compare.
type WorkItem struct {
	Index1 int
	Index2 int
	Func1  *ast.Function
	Func2  *ast.Function
}

// WorkResult represents the result of a similarity comparison.
type WorkResult struct {
	WorkItem

	Similarity float64
	Error      error
}

// FindSimilarFunctions implements ParallelProcessor interface with goroutine pool and progress tracking.
func (p *DefaultParallelProcessor) FindSimilarFunctions(
	functions []*ast.Function,
	progressCallback func(completed, total int),
) ([]Match, error) {
	const minFunctionCount = 2
	if len(functions) < minFunctionCount {
		return []Match{}, nil
	}

	// Pre-normalize all functions to avoid race conditions during parallel processing
	normalizedFunctions := make([]*ast.Function, len(functions))
	for i, fn := range functions {
		normalizedFunctions[i] = fn.Normalize()
	}

	// Calculate total number of comparisons (n*(n-1)/2)
	const divisor = 2
	totalComparisons := (len(normalizedFunctions) * (len(normalizedFunctions) - 1)) / divisor

	// Create work items using pre-normalized functions
	workItems := make(chan WorkItem, totalComparisons)
	results := make(chan WorkResult, totalComparisons)

	// Fill work items
	go func() {
		defer close(workItems)
		for i := range normalizedFunctions {
			for j := i + 1; j < len(normalizedFunctions); j++ {
				select {
				case workItems <- WorkItem{
					Index1: i,
					Index2: j,
					Func1:  normalizedFunctions[i],
					Func2:  normalizedFunctions[j],
				}:
				case <-p.ctx.Done():
					return
				}
			}
		}
	}()

	// Start worker goroutines
	var wg sync.WaitGroup
	for range p.workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.worker(workItems, results)
		}()
	}

	// Close results channel when all workers finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results with progress tracking
	var matches []Match
	var matchesMutex sync.Mutex
	var completed int64

	for result := range results {
		if result.Error != nil {
			return nil, result.Error
		}

		if p.detector.IsAboveThreshold(result.Similarity) {
			matchesMutex.Lock()
			matches = append(matches, Match{
				Function1:  functions[result.Index1], // Use original functions in result
				Function2:  functions[result.Index2], // Use original functions in result
				Similarity: result.Similarity,
			})
			matchesMutex.Unlock()
		}

		// Update progress atomically
		newCompleted := atomic.AddInt64(&completed, 1)
		if progressCallback != nil {
			progressCallback(int(newCompleted), totalComparisons)
		}

		// Check for cancellation
		select {
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		default:
		}
	}

	return matches, nil
}

// worker processes work items and sends results.
func (p *DefaultParallelProcessor) worker(workItems <-chan WorkItem, results chan<- WorkResult) {
	for item := range workItems {
		// Check for cancellation before processing
		select {
		case <-p.ctx.Done():
			results <- WorkResult{WorkItem: item, Error: p.ctx.Err()}
			return
		default:
		}

		// Calculate similarity using pre-normalized functions
		// No need for deep copying since functions are already normalized
		similarity := p.detector.CalculateSimilarity(item.Func1, item.Func2)

		// Send result
		select {
		case results <- WorkResult{
			WorkItem:   item,
			Similarity: similarity,
		}:
		case <-p.ctx.Done():
			results <- WorkResult{WorkItem: item, Error: p.ctx.Err()}
			return
		}
	}
}

// GetWorkerCount returns the number of worker goroutines.
func (p *DefaultParallelProcessor) GetWorkerCount() int {
	return p.workerCount
}

// WithContext returns a new processor with the given context.
func (p *DefaultParallelProcessor) WithContext(ctx context.Context) *DefaultParallelProcessor {
	return &DefaultParallelProcessor{
		detector:    p.detector,
		workerCount: p.workerCount,
		ctx:         ctx,
	}
}
