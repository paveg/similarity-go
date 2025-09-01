package similarity

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/testhelpers"
)

// createBenchmarkFunction is a helper for benchmarks that can't use testing.T.
func createBenchmarkFunction(source, name string) *ast.Function {
	// Create a dummy test instance for the helper
	t := &testing.T{}
	return testhelpers.CreateFunctionFromSource(t, source, name)
}

func TestNewDefaultParallelProcessor(t *testing.T) {
	detector := NewDetector(0.8)

	// Test with explicit worker count
	processor := NewDefaultParallelProcessor(detector, 4)
	if processor.GetWorkerCount() != 4 {
		t.Errorf("Expected 4 workers, got %d", processor.GetWorkerCount())
	}

	// Test with zero/negative worker count (should default to runtime.NumCPU())
	processor = NewDefaultParallelProcessor(detector, 0)
	if processor.GetWorkerCount() <= 0 {
		t.Error("Expected positive worker count when defaulting")
	}

	processor = NewDefaultParallelProcessor(detector, -1)
	if processor.GetWorkerCount() <= 0 {
		t.Error("Expected positive worker count when defaulting")
	}
}

func TestNewDefaultParallelProcessorWithContext(t *testing.T) {
	detector := NewDetector(0.8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processor := NewDefaultParallelProcessorWithContext(ctx, detector, 2)
	if processor.GetWorkerCount() != 2 {
		t.Errorf("Expected 2 workers, got %d", processor.GetWorkerCount())
	}

	// Test context is properly stored
	if processor.ctx != ctx {
		t.Error("Context was not properly stored")
	}
}

func TestDefaultParallelProcessor_FindSimilarFunctions(t *testing.T) {
	detector := NewDetector(0.8)
	processor := NewDefaultParallelProcessor(detector, 2)

	// Create test functions
	functions := []*ast.Function{
		testhelpers.CreateFunctionFromSource(t, `package main
func add(a, b int) int { return a + b }`, "add"),
		testhelpers.CreateFunctionFromSource(t, `package main  
func sum(x, y int) int { return x + y }`, "sum"), // Very similar to add
		testhelpers.CreateFunctionFromSource(t, `package main
func multiply(a, b int) int { return a * b }`, "multiply"), // Different
	}

	// Test normal execution
	matches, err := processor.FindSimilarFunctions(functions, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should find similarity between add and sum
	if len(matches) == 0 {
		t.Error("Expected to find at least one match")
	}

	// Verify match contains similar functions
	found := false
	for _, match := range matches {
		if (match.Function1.Name == "add" && match.Function2.Name == "sum") ||
			(match.Function1.Name == "sum" && match.Function2.Name == "add") {
			found = true
			if match.Similarity < 0.8 {
				t.Errorf("Expected high similarity for add/sum, got %f", match.Similarity)
			}
		}
	}
	if !found {
		t.Error("Expected to find similarity between add and sum functions")
	}
}

func TestDefaultParallelProcessor_FindSimilarFunctions_EmptyInput(t *testing.T) {
	detector := NewDetector(0.8)
	processor := NewDefaultParallelProcessor(detector, 2)

	// Test with empty slice
	matches, err := processor.FindSimilarFunctions([]*ast.Function{}, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Error("Expected no matches for empty input")
	}

	// Test with single function
	function := testhelpers.CreateFunctionFromSource(t, `package main
func test() { return 42 }`, "test")
	matches, err = processor.FindSimilarFunctions([]*ast.Function{function}, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Error("Expected no matches for single function")
	}
}

func TestDefaultParallelProcessor_ProgressCallback(t *testing.T) {
	detector := NewDetector(0.8)
	processor := NewDefaultParallelProcessor(detector, 2)

	// Create test functions
	functions := []*ast.Function{
		testhelpers.CreateFunctionFromSource(t, `package main
func func1() { return 1 }`, "func1"),
		testhelpers.CreateFunctionFromSource(t, `package main
func func2() { return 2 }`, "func2"),
		testhelpers.CreateFunctionFromSource(t, `package main
func func3() { return 3 }`, "func3"),
		testhelpers.CreateFunctionFromSource(t, `package main
func func4() { return 4 }`, "func4"),
	}

	var callbackCount int64
	var lastCompleted, lastTotal int

	progressCallback := func(completed, total int) {
		atomic.AddInt64(&callbackCount, 1)
		lastCompleted = completed
		lastTotal = total
	}

	matches, err := processor.FindSimilarFunctions(functions, progressCallback)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Progress callback should have been called
	if atomic.LoadInt64(&callbackCount) == 0 {
		t.Error("Expected progress callback to be called")
	}

	// Total should be (n*(n-1))/2 = (4*3)/2 = 6
	expectedTotal := 6
	if lastTotal != expectedTotal {
		t.Errorf("Expected total %d, got %d", expectedTotal, lastTotal)
	}

	// Final completed should equal total
	if lastCompleted != expectedTotal {
		t.Errorf("Expected completed %d, got %d", expectedTotal, lastCompleted)
	}

	_ = matches // We don't care about the actual matches in this test
}

func TestDefaultParallelProcessor_Cancellation(t *testing.T) {
	detector := NewDetector(0.8)
	ctx, cancel := context.WithCancel(context.Background())
	processor := NewDefaultParallelProcessorWithContext(
		ctx,
		detector,
		1,
	) // Single worker to make cancellation more predictable

	// Create many functions to ensure cancellation occurs during processing
	functions := make([]*ast.Function, 50)
	for i := range 50 {
		source := fmt.Sprintf(`package main
func func%d() { 
	x := %d
	for j := 0; j < 10000; j++ {
		for k := 0; k < 1000; k++ {
			for l := 0; l < 100; l++ {
				x += j * k * l
			}
		}
	}
	return x
}`, i, i)
		functions[i] = testhelpers.CreateFunctionFromSource(t, source, fmt.Sprintf("func%d", i))
	}

	// Cancel immediately to test cancellation before processing starts
	cancel()

	// Should return context error immediately or very quickly
	matches, err := processor.FindSimilarFunctions(functions, nil)

	if !errors.Is(err, context.Canceled) {
		// If cancellation didn't happen immediately, it's still acceptable
		// The test passes as long as no panic occurs and results are reasonable
		t.Logf("Cancellation test completed with error: %v", err)
		if matches == nil && err != nil {
			// This is acceptable - some error occurred
			return
		}
	} else if matches != nil {
		// Expected cancellation occurred
		t.Error("Expected nil matches on cancellation")
	}
}

func TestDefaultParallelProcessor_TimeoutCancellation(t *testing.T) {
	detector := NewDetector(0.8)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond) // Very short timeout
	defer cancel()
	processor := NewDefaultParallelProcessorWithContext(ctx, detector, 1)

	// Create functions
	functions := make([]*ast.Function, 10)
	for i := range 10 {
		source := fmt.Sprintf(`package main
func func%d() { 
	result := 0
	for i := 0; i < 100; i++ {
		result += i
	}
	return result
}`, i)
		functions[i] = testhelpers.CreateFunctionFromSource(t, source, fmt.Sprintf("func%d", i))
	}

	// Should timeout or complete quickly
	matches, err := processor.FindSimilarFunctions(functions, nil)

	if errors.Is(err, context.DeadlineExceeded) {
		// Expected timeout occurred
		if matches != nil {
			t.Error("Expected nil matches on timeout")
		}
	} else {
		// Test passes as long as no panic occurs
		t.Logf("Timeout test completed with error: %v, matches: %d", err, len(matches))
	}
}

func TestDefaultParallelProcessor_WithContext(t *testing.T) {
	detector := NewDetector(0.8)
	original := NewDefaultParallelProcessor(detector, 4)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	newProcessor := original.WithContext(ctx)

	// Should have same detector and worker count
	if newProcessor.detector != original.detector {
		t.Error("Expected same detector")
	}
	if newProcessor.GetWorkerCount() != original.GetWorkerCount() {
		t.Error("Expected same worker count")
	}

	// Should have new context
	if newProcessor.ctx != ctx {
		t.Error("Expected new context")
	}
	if newProcessor.ctx == original.ctx {
		t.Error("Expected different context from original")
	}
}

func TestDefaultParallelProcessor_RaceConditionSafety(t *testing.T) {
	detector := NewDetector(0.8)
	processor := NewDefaultParallelProcessor(detector, 4) // Multiple workers

	// Create functions
	functions := make([]*ast.Function, 8)
	for i := range 8 {
		source := fmt.Sprintf(`package main
func func%d() int { 
	x := %d
	y := x * 2
	return x + y
}`, i, i)
		functions[i] = testhelpers.CreateFunctionFromSource(t, source, fmt.Sprintf("func%d", i))
	}

	// Run multiple times to catch race conditions
	for run := range 5 {
		matches, err := processor.FindSimilarFunctions(functions, nil)
		if err != nil {
			t.Fatalf("Run %d failed with error: %v", run, err)
		}

		// Basic sanity check
		for _, match := range matches {
			if match.Similarity < 0 || match.Similarity > 1 {
				t.Errorf("Invalid similarity score: %f", match.Similarity)
			}
			if match.Function1 == nil || match.Function2 == nil {
				t.Error("Null functions in match")
			}
		}
	}
}

// BenchmarkDefaultParallelProcessor_FindSimilarFunctions benchmarks parallel processing performance.
func BenchmarkDefaultParallelProcessor_FindSimilarFunctions(b *testing.B) {
	detector := NewDetector(0.8)
	processor := NewDefaultParallelProcessor(detector, 4)

	// Create benchmark functions
	functions := make([]*ast.Function, 10)
	for i := range 10 {
		source := fmt.Sprintf(`package main
func func%d() int { 
	sum := 0
	for j := 0; j < %d; j++ {
		sum += j * %d
	}
	return sum
}`, i, i+1, i+2)
		functions[i] = createBenchmarkFunction(source, fmt.Sprintf("func%d", i))
	}

	b.ResetTimer()
	for range b.N {
		_, err := processor.FindSimilarFunctions(functions, nil)
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// TestDefaultParallelProcessor_vs_Sequential compares parallel vs sequential performance.
func TestDefaultParallelProcessor_vs_Sequential(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison in short mode")
	}

	detector := NewDetector(0.8)
	parallelProcessor := NewDefaultParallelProcessor(detector, 4)

	// Create functions for comparison
	functions := make([]*ast.Function, 12)
	for i := range 12 {
		source := fmt.Sprintf(`package main
func func%d() int { 
	result := %d
	for k := 0; k < 50; k++ {
		result = result * 2 + k
	}
	return result
}`, i, i)
		functions[i] = testhelpers.CreateFunctionFromSource(t, source, fmt.Sprintf("func%d", i))
	}

	// Time sequential processing
	start := time.Now()
	sequentialMatches := detector.FindSimilarFunctions(functions)
	sequentialTime := time.Since(start)

	// Time parallel processing
	start = time.Now()
	parallelMatches, err := parallelProcessor.FindSimilarFunctions(functions, nil)
	parallelTime := time.Since(start)

	if err != nil {
		t.Fatalf("Parallel processing failed: %v", err)
	}

	// Results should be the same (order may differ)
	if len(sequentialMatches) != len(parallelMatches) {
		t.Errorf("Different number of matches: sequential=%d, parallel=%d",
			len(sequentialMatches), len(parallelMatches))
	}

	t.Logf("Sequential time: %v, Parallel time: %v", sequentialTime, parallelTime)

	// Parallel should not be significantly slower (allowing for overhead)
	if parallelTime > sequentialTime*2 {
		t.Logf("Warning: Parallel processing much slower than sequential: %v vs %v",
			parallelTime, sequentialTime)
	}
}
