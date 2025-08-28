package worker

import "errors"

var (
	// ErrPoolNotStarted is returned when trying to submit tasks to a pool that hasn't been started.
	ErrPoolNotStarted = errors.New("worker pool not started")

	// ErrPoolStopped is returned when trying to submit tasks to a stopped pool.
	ErrPoolStopped = errors.New("worker pool stopped")
)
