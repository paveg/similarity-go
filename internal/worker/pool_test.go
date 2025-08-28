package worker

import (
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	tests := []struct {
		name            string
		workers         int
		expectedWorkers int
	}{
		{
			name:            "positive workers",
			workers:         4,
			expectedWorkers: 4,
		},
		{
			name:            "zero workers defaults to NumCPU",
			workers:         0,
			expectedWorkers: runtime.NumCPU(),
		},
		{
			name:            "negative workers defaults to NumCPU",
			workers:         -1,
			expectedWorkers: runtime.NumCPU(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewPool(tt.workers)
			if pool.WorkerCount() != tt.expectedWorkers {
				t.Errorf("expected %d workers, got %d", tt.expectedWorkers, pool.WorkerCount())
			}
			if pool.IsStarted() {
				t.Error("pool should not be started initially")
			}
		})
	}
}

func TestPoolStartStop(t *testing.T) {
	pool := NewPool(2)

	// Test initial state
	if pool.IsStarted() {
		t.Error("pool should not be started initially")
	}

	// Test start
	pool.Start()
	if !pool.IsStarted() {
		t.Error("pool should be started after Start()")
	}

	// Test double start (should be safe)
	pool.Start()
	if !pool.IsStarted() {
		t.Error("pool should still be started after double Start()")
	}

	// Test stop
	pool.Stop()
	if pool.IsStarted() {
		t.Error("pool should not be started after Stop()")
	}

	// Test double stop (should be safe)
	pool.Stop()
	if pool.IsStarted() {
		t.Error("pool should still not be started after double Stop()")
	}
}

func TestPoolSubmitAndExecute(t *testing.T) {
	pool := NewPool(2)
	pool.Start()
	defer pool.Stop()

	// Test successful task execution
	executed := make(chan bool, 1)

	task := func() error {
		executed <- true
		return nil
	}

	err := pool.Submit(task)
	if err != nil {
		t.Fatalf("unexpected error submitting task: %v", err)
	}

	// Wait for task to execute with timeout
	select {
	case <-executed:
		// Task completed successfully
	case <-time.After(100 * time.Millisecond):
		t.Fatal("task did not execute within timeout")
	}
}

func TestPoolTaskResults(t *testing.T) {
	pool := NewPool(2)
	pool.Start()
	defer pool.Stop()

	// Submit tasks and check results
	numTasks := 5
	for i := range numTasks {
		task := func() error {
			// Simulate work without sleep - just return
			return nil
		}

		err := pool.Submit(task)
		if err != nil {
			t.Fatalf("unexpected error submitting task %d: %v", i, err)
		}
	}

	// Collect results
	results := 0
	timeout := time.After(1 * time.Second)

	for results < numTasks {
		select {
		case result := <-pool.Results():
			if result.Error != nil {
				t.Errorf("unexpected task error: %v", result.Error)
			}
			results++
		case <-timeout:
			t.Fatalf("did not receive all results, got %d out of %d", results, numTasks)
		}
	}
}

func TestPoolSubmitToStoppedPool(t *testing.T) {
	pool := NewPool(2)

	// Try to submit to unstarted pool
	task := func() error { return nil }
	err := pool.Submit(task)

	if !errors.Is(err, ErrPoolNotStarted) {
		t.Errorf("expected ErrPoolNotStarted, got %v", err)
	}
}

func TestPoolConcurrentAccess(t *testing.T) {
	pool := NewPool(4)
	pool.Start()
	defer pool.Stop()

	// Submit fewer tasks concurrently for faster testing
	numGoroutines := 3
	tasksPerGoroutine := 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()
			for range tasksPerGoroutine {
				task := func() error {
					// Simulate work without sleep - just return
					return nil
				}

				err := pool.Submit(task)
				if err != nil {
					t.Errorf("unexpected error submitting task: %v", err)
				}
			}
		}()
	}

	// Wait for all submissions
	wg.Wait()

	// Collect all results with shorter timeout
	totalTasks := numGoroutines * tasksPerGoroutine
	results := 0
	timeout := time.After(500 * time.Millisecond)

	for results < totalTasks {
		select {
		case result := <-pool.Results():
			if result.Error != nil {
				t.Errorf("unexpected task error: %v", result.Error)
			}
			results++
		case <-timeout:
			t.Fatalf("did not receive all results, got %d out of %d", results, totalTasks)
		}
	}
}
