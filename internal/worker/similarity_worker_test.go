package worker

import (
	"runtime"
	"testing"

	"github.com/paveg/similarity-go/internal/ast"
	"github.com/paveg/similarity-go/internal/config"
	"github.com/paveg/similarity-go/internal/similarity"
)

// Create test functions for similarity testing.
func createTestFunction(name, body string) *ast.Function {
	_ = body // body parameter used for test readability but not needed in implementation
	return &ast.Function{
		Name:      name,
		File:      "test.go",
		StartLine: 1,
		EndLine:   10,
		LineCount: 10,
		// Note: In real implementation, AST would be populated
		// For tests, we'll rely on the similarity detector's logic
	}
}

func TestNewSimilarityWorker(t *testing.T) {
	cfg := config.Default()
	detector := similarity.NewDetectorWithConfig(0.8, cfg)

	tests := []struct {
		name            string
		workers         int
		expectedWorkers int
		threshold       float64
	}{
		{
			name:            "positive workers",
			workers:         4,
			expectedWorkers: 4,
			threshold:       0.8,
		},
		{
			name:            "zero workers defaults to NumCPU",
			workers:         0,
			expectedWorkers: runtime.NumCPU(),
			threshold:       0.7,
		},
		{
			name:            "negative workers defaults to NumCPU",
			workers:         -1,
			expectedWorkers: runtime.NumCPU(),
			threshold:       0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewSimilarityWorker(detector, tt.workers, tt.threshold)

			if worker.WorkerCount() != tt.expectedWorkers {
				t.Errorf("expected %d workers, got %d", tt.expectedWorkers, worker.WorkerCount())
			}

			if worker.threshold != tt.threshold {
				t.Errorf("expected threshold %f, got %f", tt.threshold, worker.threshold)
			}

			if worker.detector != detector {
				t.Error("detector not set correctly")
			}
		})
	}
}

func TestSimilarityWorkerEmptyInput(t *testing.T) {
	cfg := config.Default()
	detector := similarity.NewDetectorWithConfig(0.8, cfg)
	worker := NewSimilarityWorker(detector, 2, 0.8)

	tests := []struct {
		name      string
		functions []*ast.Function
	}{
		{
			name:      "nil functions",
			functions: nil,
		},
		{
			name:      "empty functions",
			functions: []*ast.Function{},
		},
		{
			name:      "single function",
			functions: []*ast.Function{createTestFunction("func1", "return 1")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := worker.FindSimilarFunctions(tt.functions, nil)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(matches) != 0 {
				t.Errorf("expected no matches, got %d", len(matches))
			}
		})
	}
}

func TestSimilarityWorkerProgressCallback(t *testing.T) {
	cfg := config.Default()
	detector := similarity.NewDetectorWithConfig(0.8, cfg)
	worker := NewSimilarityWorker(detector, 2, 0.8)

	// Create multiple test functions
	functions := []*ast.Function{
		createTestFunction("func1", "return 1"),
		createTestFunction("func2", "return 2"),
		createTestFunction("func3", "return 3"),
		createTestFunction("func4", "return 4"),
	}

	// Track progress callback calls
	var progressCalls []struct {
		completed int
		total     int
	}

	progressCallback := func(completed, total int) {
		progressCalls = append(progressCalls, struct {
			completed int
			total     int
		}{completed, total})
	}

	_, err := worker.FindSimilarFunctions(functions, progressCallback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have progress callbacks
	if len(progressCalls) == 0 {
		t.Error("expected progress callbacks, got none")
	}

	// Verify total is consistent
	expectedTotal := len(functions) * (len(functions) - 1) / 2 // n*(n-1)/2 comparisons
	for _, call := range progressCalls {
		if call.total != expectedTotal {
			t.Errorf("expected total %d, got %d", expectedTotal, call.total)
		}

		if call.completed < 0 || call.completed > call.total {
			t.Errorf("invalid progress: completed %d, total %d", call.completed, call.total)
		}
	}

	// Last call should be complete
	if len(progressCalls) > 0 {
		lastCall := progressCalls[len(progressCalls)-1]
		if lastCall.completed != lastCall.total {
			t.Errorf("last progress call should be complete: completed %d, total %d",
				lastCall.completed, lastCall.total)
		}
	}
}

func TestSimilarityWorkerConcurrency(t *testing.T) {
	cfg := config.Default()
	detector := similarity.NewDetectorWithConfig(0.1, cfg) // Low threshold to get matches

	tests := []struct {
		name    string
		workers int
	}{
		{
			name:    "single worker",
			workers: 1,
		},
		{
			name:    "multiple workers",
			workers: 4,
		},
		{
			name:    "many workers",
			workers: 8,
		},
	}

	// Create many test functions for meaningful parallelization
	functions := make([]*ast.Function, 20)
	for i := range 20 {
		functions[i] = createTestFunction(
			"func"+string(rune('A'+i%26)),
			"return "+string(rune('0'+i%10)),
		)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewSimilarityWorker(detector, tt.workers, 0.1)

			matches, err := worker.FindSimilarFunctions(functions, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify all matches are valid
			for _, match := range matches {
				if match.Function1 == nil || match.Function2 == nil {
					t.Error("match contains nil function")
				}

				if match.Similarity < 0 || match.Similarity > 1 {
					t.Errorf("invalid similarity score: %f", match.Similarity)
				}
			}

			// Verify no duplicate matches
			seen := make(map[string]bool)
			for _, match := range matches {
				key1 := match.Function1.Name + "-" + match.Function2.Name
				key2 := match.Function2.Name + "-" + match.Function1.Name

				if seen[key1] || seen[key2] {
					t.Errorf("duplicate match found: %s", key1)
				}

				seen[key1] = true
				seen[key2] = true
			}
		})
	}
}

func TestComparisonResult(t *testing.T) {
	cfg := config.Default()
	detector := similarity.NewDetectorWithConfig(0.8, cfg)
	worker := NewSimilarityWorker(detector, 2, 0.8)

	func1 := createTestFunction("func1", "return 1")
	func2 := createTestFunction("func2", "return 2")

	job := ComparisonJob{
		Function1: func1,
		Function2: func2,
		Index1:    0,
		Index2:    1,
	}

	result := worker.processComparison(job)

	if result.Index1 != 0 {
		t.Errorf("expected Index1 = 0, got %d", result.Index1)
	}

	if result.Index2 != 1 {
		t.Errorf("expected Index2 = 1, got %d", result.Index2)
	}

	if !result.Completed {
		t.Error("result should be marked as completed")
	}

	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
}
