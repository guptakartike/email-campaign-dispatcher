package dispatcher

import (
	"fmt"
	"sync"
	"time"

	"github.com/guptakartike/email-dispatcher/internal/models"
	"github.com/guptakartike/email-dispatcher/internal/sender"
)

// RetryConfig controls the retry behaviour for failed email sends.
type RetryConfig struct {
	MaxAttempts int           // Total attempts per email (including the first).
	Delay       time.Duration // Wait time between retry attempts.
}

// StartWorkers launches workerCount goroutines that each read EmailJob
// values from the jobs channel and send them via SMTP with retry logic.
// It blocks until all workers have finished.
func StartWorkers(jobs <-chan models.EmailJob, workerCount int, cfg sender.EmailConfig, retry RetryConfig) {
	var wg sync.WaitGroup

	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go worker(i, jobs, &wg, cfg, retry)
	}

	wg.Wait()
}

// worker is a single goroutine that processes jobs until the channel is closed.
// For each job it attempts to send via SMTP up to retry.MaxAttempts times.
// A permanently failed email is logged but does NOT stop the worker.
func worker(id int, jobs <-chan models.EmailJob, wg *sync.WaitGroup, cfg sender.EmailConfig, retry RetryConfig) {
	defer wg.Done()

	for job := range jobs {
		sendWithRetry(id, job, cfg, retry)
	}

	fmt.Printf("[Worker %d] Done — no more jobs.\n", id)
}

// sendWithRetry attempts to send a single email up to maxAttempts times.
// On success it returns immediately. On failure it waits retry.Delay before
// the next attempt. After all attempts are exhausted the failure is logged.
func sendWithRetry(workerID int, job models.EmailJob, cfg sender.EmailConfig, retry RetryConfig) {
	var err error

	for attempt := 1; attempt <= retry.MaxAttempts; attempt++ {
		err = sender.Send(
			cfg,
			job.Recipient.Email,
			job.Recipient.Name,
			job.Subject,
			job.Body,
		)

		if err == nil {
			fmt.Printf("[Worker %d] Sent to %s <%s>\n", workerID, job.Recipient.Name, job.Recipient.Email)
			return
		}

		if attempt < retry.MaxAttempts {
			fmt.Printf("[Worker %d] Attempt %d/%d failed for %s <%s>: %v — retrying in %v\n",
				workerID, attempt, retry.MaxAttempts, job.Recipient.Name, job.Recipient.Email, err, retry.Delay)
			time.Sleep(retry.Delay)
		}
	}

	// All attempts exhausted.
	fmt.Printf("[Worker %d] PERMANENTLY FAILED %s <%s> after %d attempts: %v\n",
		workerID, job.Recipient.Name, job.Recipient.Email, retry.MaxAttempts, err)
}
