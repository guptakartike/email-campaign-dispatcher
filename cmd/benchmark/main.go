package main

import (
	"fmt"
	"time"

	"github.com/guptakartike/email-dispatcher/internal/dispatcher"
	"github.com/guptakartike/email-dispatcher/internal/models"
	"github.com/guptakartike/email-dispatcher/internal/sender"
)

func main() {
	// Disable per-email console logs so terminal I/O does not distort benchmark measurements.
	dispatcher.Verbose = false

	fmt.Println("========================================")
	fmt.Println("       MailPunk Performance Benchmark")
	fmt.Println("========================================")
	fmt.Printf("Started at: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	workerCounts := []int{1, 5, 10, 20}
	defaultRetry := dispatcher.RetryConfig{
		MaxAttempts: 3,
		Delay:       0,
	}

	// -------------------------------------------------------------
	// PART 1: Raw Throughput (500 Recipients, Rate Limiter Bypassed)
	// -------------------------------------------------------------
	runSuite("RAW THROUGHPUT (500 Recipients, Rate Limiter: BYPASSED, Latency: 0ms)", 500, workerCounts, nil, 0, defaultRetry)

	// -------------------------------------------------------------
	// PART 2: Raw Throughput (1000 Recipients, Rate Limiter Bypassed)
	// -------------------------------------------------------------
	runSuite("RAW THROUGHPUT (1000 Recipients, Rate Limiter: BYPASSED, Latency: 0ms)", 1000, workerCounts, nil, 0, defaultRetry)

	// -------------------------------------------------------------
	// PART 3: Rate-Limited Throughput (EMAILS_PER_SECOND = 100)
	// -------------------------------------------------------------
	limiterRate := 100.0
	makeLimiter := func() *dispatcher.RateLimiter {
		return dispatcher.NewRateLimiter(limiterRate)
	}
	runSuite(fmt.Sprintf("RATE-LIMITED THROUGHPUT (500 Recipients, EMAILS_PER_SECOND = %.0f)", limiterRate), 500, workerCounts, makeLimiter, 0, defaultRetry)

	// -------------------------------------------------------------
	// PART 4: Realistic SMTP Latency Simulation (2ms network RTT)
	// -------------------------------------------------------------
	runSuite("I/O SIMULATION (500 Recipients, Rate Limiter: BYPASSED, Latency: 2ms)", 500, workerCounts, nil, 2*time.Millisecond, defaultRetry)

	// -------------------------------------------------------------
	// PART 5: Retry System Benchmark (Simulated 20% Initial Failures)
	// -------------------------------------------------------------
	runRetryBenchmark(500, 5)
}

// generateRecipients creates N synthetic recipient models.
func generateRecipients(count int) []models.Recipient {
	recipients := make([]models.Recipient, count)
	for i := 0; i < count; i++ {
		recipients[i] = models.Recipient{
			Name:  fmt.Sprintf("Test User %d", i+1),
			Email: fmt.Sprintf("testuser%d@example.com", i+1),
		}
	}
	return recipients
}

// runSuite executes the benchmark for a given recipient count across worker counts.
func runSuite(title string, count int, workerCounts []int, makeLimiter func() *dispatcher.RateLimiter, latency time.Duration, retry dispatcher.RetryConfig) {
	fmt.Println("========================================")
	fmt.Printf("%s\n", title)
	fmt.Println("========================================")
	fmt.Printf("Recipients: %d\n\n", count)
	fmt.Printf("%-8s | %-10s | %s\n", "Workers", "Time", "Emails/sec")
	fmt.Println("--------------------------------")

	recipients := generateRecipients(count)
	var lastStats dispatcher.WorkerStats

	for _, w := range workerCounts {
		mock := sender.NewMockSender(latency)
		jobs := make(chan models.EmailJob, count)

		var limiter *dispatcher.RateLimiter
		if makeLimiter != nil {
			limiter = makeLimiter()
		}

		go dispatcher.Produce(recipients, jobs, "Benchmark Subject", "Hello {{name}}")

		start := time.Now()
		stats := dispatcher.StartWorkers(jobs, w, mock, retry, limiter)
		duration := time.Since(start)

		if limiter != nil {
			limiter.Stop()
		}

		lastStats = stats
		throughput := float64(stats.Successful+stats.Failed) / duration.Seconds()

		fmt.Printf("%-8d | %-10s | %.0f\n", w, formatDuration(duration), throughput)
	}

	fmt.Println("--------------------------------")
	fmt.Printf("Success:  %d\n", lastStats.Successful)
	fmt.Printf("Failures: %d\n", lastStats.Failed)
	if lastStats.Retries > 0 {
		fmt.Printf("Retries:  %d\n", lastStats.Retries)
	}
	fmt.Println("========================================")
	fmt.Println()
}

// runRetryBenchmark tests the retry system with intentional mock failures.
func runRetryBenchmark(count int, workers int) {
	fmt.Println("========================================")
	fmt.Println("RETRY SYSTEM BENCHMARK (Simulated Failures)")
	fmt.Println("========================================")
	fmt.Printf("Recipients: %d | Workers: %d | Simulated Fail Every: 5th send\n\n", count, workers)

	recipients := generateRecipients(count)
	mock := sender.NewMockSender(0)
	mock.FailEvery = 5 // every 5th send attempt fails

	retryCfg := dispatcher.RetryConfig{
		MaxAttempts: 3,
		Delay:       1 * time.Millisecond,
	}

	jobs := make(chan models.EmailJob, count)
	go dispatcher.Produce(recipients, jobs, "Retry Test", "Hello {{name}}")

	start := time.Now()
	stats := dispatcher.StartWorkers(jobs, workers, mock, retryCfg, nil)
	duration := time.Since(start)

	throughput := float64(stats.Successful+stats.Failed) / duration.Seconds()

	fmt.Printf("%-8s | %-10s | %s\n", "Workers", "Time", "Emails/sec")
	fmt.Println("--------------------------------")
	fmt.Printf("%-8d | %-10s | %.0f\n", workers, formatDuration(duration), throughput)
	fmt.Println("--------------------------------")
	fmt.Printf("Success:  %d\n", stats.Successful)
	fmt.Printf("Failures: %d\n", stats.Failed)
	fmt.Printf("Retries:  %d\n", stats.Retries)
	fmt.Println("========================================")
	fmt.Println()
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Microseconds())/1000.0)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
