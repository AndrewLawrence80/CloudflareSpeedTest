package concurrency

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AndrewLawrence80/CloudflareSpeedTest/pkg/log"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

/*
Executor is an interface that defines methods for submitting tasks, waiting for their completion, and closing the executor.

Methods:

- Submit(task func()) error: Submits a task to be executed. The task is a function that takes no arguments and returns no value. The method returns an error if the task cannot be submitted.

- Wait(): Waits for all submitted tasks to complete. This method blocks until all tasks have finished executing.

- Close(): Closes the executor, preventing any new tasks from being submitted. This method should be called when the executor is no longer needed to release any resources it may be holding.
*/
type Executor interface {
	Submit(ctx context.Context, task func()) error
	Wait()
	Close()
}

/*
SimpleExecutor is a basic implementation of the Executor interface that limits the number of concurrent tasks
and the number of tasks that can be submitted per minute.

Use [NewSimpleExecutor] to create a new instance of SimpleExecutor with the desired concurrency and rate limits.
*/
type SimpleExecutor struct {
	sem         *semaphore.Weighted
	rateLimiter *rate.Limiter
	wg          sync.WaitGroup
	closed      atomic.Bool
}

/*
NewSimpleExecutor creates a new SimpleExecutor with the specified number of routines and queries per minute (qpm).

Parameters:

- nRoutines: The maximum number of concurrent goroutines, 0 means no limit.

- qpm: The maximum number of tasks submitted per minute, 0 means no limit.

Returns:

- A pointer to a new SimpleExecutor instance.
*/
func NewSimpleExecutor(nRoutines uint, qpm uint) *SimpleExecutor {
	executor := &SimpleExecutor{}
	if nRoutines > 0 {
		executor.sem = semaphore.NewWeighted(int64(nRoutines))
	}
	if qpm > 0 {
		// Burst of 1 ensures tasks are spread evenly over the minute
		// rather than allowing all tokens to be consumed at once.
		executor.rateLimiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(qpm)), 1)
	}
	return executor
}

// Submit enqueues a task for concurrent execution.
// Returns an error if the executor has been closed or if waiting for a
// concurrency slot or rate-limiter token fails.
func (e *SimpleExecutor) Submit(ctx context.Context, task func()) error {
	if e.closed.Load() {
		return fmt.Errorf("concurrency: executor is closed")
	}

	if e.sem != nil {
		if err := e.sem.Acquire(ctx, 1); err != nil {
			return fmt.Errorf("concurrency: semaphore acquire: %w", err)
		}
	}

	if e.rateLimiter != nil {
		if err := e.rateLimiter.Wait(ctx); err != nil {
			// Release the slot acquired above before returning.
			if e.sem != nil {
				e.sem.Release(1)
			}
			return fmt.Errorf("concurrency: rate limiter wait: %w", err)
		}
	}

	e.wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				stackInfo := make([]byte, 1024)
				n := runtime.Stack(stackInfo, false)
				log.GetLogger().Error("task panicked", "error", r, "stack", string(stackInfo[:n]))
			}
			if e.sem != nil {
				e.sem.Release(1)
			}
			e.wg.Done()
		}()
		task()
	}()
	return nil
}

// Wait blocks until all submitted tasks have completed.
func (e *SimpleExecutor) Wait() {
	e.wg.Wait()
}

// Close marks the executor as closed. Subsequent calls to Submit will return
// an error. Already-running tasks are not cancelled; call Wait after Close to
// drain them.
func (e *SimpleExecutor) Close() {
	e.closed.Store(true)
}

// exampleUsage demonstrates how to use SimpleExecutor with a cancellable
// context captured via closure, without changing the task func() signature.
func exampleUsage() {
	executor := NewSimpleExecutor(5, 10) // 5 concurrent goroutines, 10 tasks/minute
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ctx is captured by the closure; the task respects cancellation without
	// needing a context parameter in the func() signature.
	if err := executor.Submit(ctx, func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		fmt.Println("Task 1 is running")
	}); err != nil {
		log.GetLogger().Error("failed to submit task", "error", err)
	}

	executor.Close() // stop accepting new tasks
	executor.Wait()  // drain in-flight tasks
}
