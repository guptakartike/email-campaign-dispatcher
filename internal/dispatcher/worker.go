package dispatcher

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/guptakartike/email-dispatcher/internal/models"
	"github.com/guptakartike/email-dispatcher/internal/sender"
)

// Verbose controls whether individual email send / retry logs are printed.
// Defaults to true for normal CLI execution; set to false in benchmarks.
var Verbose = true

// RetryConfig controls the retry behaviour for failed email sends.
type RetryConfig struct {
	MaxAttempts int           // Total attempts per email (including the first).
	Delay       time.Duration // Wait time between retry attempts.
}

// WorkerStats tracks execution metrics across all workers in a pool.
type WorkerStats struct {
	Successful int64
	Failed     int64
	Retries    int64
}

// StartWorkers launches workerCount goroutines that each read EmailJob
// values from the jobs channel and send them using the provided Sender.
// All workers share the provided RateLimiter to collectively respect
// the global sending rate (pass nil to bypass rate limiting).
// It blocks until all workers have finished and returns execution stats.
func StartWorkers(jobs <-chan models.EmailJob, workerCount int, s sender.Sender, retry RetryConfig, limiter *RateLimiter) WorkerStats {
	var wg sync.WaitGroup
	var stats WorkerStats

	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go worker(i, jobs, &wg, s, retry, limiter, &stats)
	}

	wg.Wait()
	return stats
}

// worker is a single goroutine that processes jobs until the channel is closed.
// For each job it attempts to send via the Sender up to retry.MaxAttempts times.
// A permanently failed email is logged but does NOT stop the worker.
func worker(id int, jobs <-chan models.EmailJob, wg *sync.WaitGroup, s sender.Sender, retry RetryConfig, limiter *RateLimiter, stats *WorkerStats) {
	defer wg.Done()

	for job := range jobs {
		sendWithRetry(id, job, s, retry, limiter, stats)
	}

	if Verbose {
		fmt.Printf("[Worker %d] Done — no more jobs.\n", id)
	}
}

// sendWithRetry attempts to send a single email up to maxAttempts times.
// Every attempt (including retries) waits for the shared rate limiter (if configured)
// before calling the Sender. On success it returns immediately.
// After all attempts are exhausted the failure is logged.
func sendWithRetry(workerID int, job models.EmailJob, s sender.Sender, retry RetryConfig, limiter *RateLimiter, stats *WorkerStats) {
	var err error

	for attempt := 1; attempt <= retry.MaxAttempts; attempt++ {
		// Wait for rate limiter before every send attempt (no-op if limiter is nil).
		if limiter != nil {
			limiter.Wait()
		}

		err = s.Send(job)

		if err == nil {
			if stats != nil {
				atomic.AddInt64(&stats.Successful, 1)
			}
			if Verbose {
				fmt.Printf("[Worker %d] Sent to %s <%s>\n", workerID, job.Recipient.Name, job.Recipient.Email)
			}
			return
		}

		if attempt < retry.MaxAttempts {
			if stats != nil {
				atomic.AddInt64(&stats.Retries, 1)
			}
			if Verbose {
				fmt.Printf("[Worker %d] Attempt %d/%d failed for %s <%s>: %v — retrying in %v\n",
					workerID, attempt, retry.MaxAttempts, job.Recipient.Name, job.Recipient.Email, err, retry.Delay)
			}
			if retry.Delay > 0 {
				time.Sleep(retry.Delay)
			}
		}
	}

	// All attempts exhausted.
	if stats != nil {
		atomic.AddInt64(&stats.Failed, 1)
	}
	if Verbose {
		fmt.Printf("[Worker %d] PERMANENTLY FAILED %s <%s> after %d attempts: %v\n",
			workerID, job.Recipient.Name, job.Recipient.Email, retry.MaxAttempts, err)
	}
}
