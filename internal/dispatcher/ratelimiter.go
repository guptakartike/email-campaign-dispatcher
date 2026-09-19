package dispatcher

import (
	"time"
)

// RateLimiter controls the global sending rate across all workers.
// It uses a time.Ticker to emit tokens at a fixed interval, implementing
// a simple token-bucket with bucket size of 1.
//
// All workers share a single RateLimiter instance. Calling Wait() blocks
// until a token is available, ensuring the collective send rate never
// exceeds the configured limit.
type RateLimiter struct {
	ticker *time.Ticker
}

// NewRateLimiter creates a rate limiter that allows emailsPerSecond sends
// per second. The limiter must be stopped with Stop() when no longer needed.
func NewRateLimiter(emailsPerSecond float64) *RateLimiter {
	interval := time.Duration(float64(time.Second) / emailsPerSecond)
	return &RateLimiter{
		ticker: time.NewTicker(interval),
	}
}

// Wait blocks until the rate limiter allows the next send.
// If rl is nil, it returns immediately without waiting (bypassed).
// It is safe to call from multiple goroutines.
func (rl *RateLimiter) Wait() {
	if rl == nil || rl.ticker == nil {
		return
	}
	<-rl.ticker.C
}

// Stop releases the underlying ticker resources.
func (rl *RateLimiter) Stop() {
	if rl == nil || rl.ticker == nil {
		return
	}
	rl.ticker.Stop()
}
