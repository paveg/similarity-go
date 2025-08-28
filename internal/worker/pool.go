package worker

import (
	"context"
	"runtime"
	"sync"
)

// Task represents a unit of work to be processed by the worker pool.
type Task func() error

// Result represents the result of a task execution.
type Result struct {
	Error error
	Index int // Optional index for ordering results
}

// Pool represents a worker pool that processes tasks concurrently.
type Pool struct {
	workers  int
	taskCh   chan Task
	resultCh chan Result
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	started  bool
	mu       sync.RWMutex
}

// NewPool creates a new worker pool with the specified number of workers
// If workers <= 0, it defaults to runtime.NumCPU().
func NewPool(workers int) *Pool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		workers:  workers,
		taskCh:   make(chan Task, workers*2), // Buffer to prevent blocking
		resultCh: make(chan Result, workers*2),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start initializes and starts the worker goroutines.
func (p *Pool) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return
	}

	p.started = true

	// Start worker goroutines
	for range p.workers {
		p.wg.Add(1)
		go p.worker()
	}
}

// Stop gracefully shuts down the worker pool.
func (p *Pool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	// Signal workers to stop
	close(p.taskCh)

	// Wait for all workers to finish
	p.wg.Wait()

	// Cancel context and close result channel
	p.cancel()
	close(p.resultCh)

	p.started = false
}

// Submit adds a task to the worker pool queue.
func (p *Pool) Submit(task Task) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.started {
		return ErrPoolNotStarted
	}

	select {
	case p.taskCh <- task:
		return nil
	case <-p.ctx.Done():
		return p.ctx.Err()
	}
}

// Results returns a channel to receive task results.
func (p *Pool) Results() <-chan Result {
	return p.resultCh
}

// worker is the main worker goroutine function.
func (p *Pool) worker() {
	defer p.wg.Done()

	for {
		select {
		case task, ok := <-p.taskCh:
			if !ok {
				// Task channel closed, worker should exit
				return
			}

			// Execute task and send result
			err := task()

			select {
			case p.resultCh <- Result{Error: err}:
			case <-p.ctx.Done():
				return
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// WorkerCount returns the number of workers in the pool.
func (p *Pool) WorkerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.workers
}

// IsStarted returns whether the pool has been started.
func (p *Pool) IsStarted() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.started
}
