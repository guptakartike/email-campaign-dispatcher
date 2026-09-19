package sender

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/guptakartike/email-dispatcher/internal/models"
)

// MockSender simulates email delivery without making network requests.
// It is thread-safe and implements the Sender interface.
type MockSender struct {
	Latency   time.Duration // Simulated delay per send (0 for instant).
	FailEvery int           // If > 0, every Nth attempt fails.
	callCount int64
	sentCount int64
	failCount int64
}

// NewMockSender creates a MockSender with the specified simulated latency.
func NewMockSender(latency time.Duration) *MockSender {
	return &MockSender{Latency: latency}
}

// Send implements the Sender interface for MockSender.
func (m *MockSender) Send(job models.EmailJob) error {
	if m.Latency > 0 {
		time.Sleep(m.Latency)
	}

	count := atomic.AddInt64(&m.callCount, 1)
	if m.FailEvery > 0 && count%int64(m.FailEvery) == 0 {
		atomic.AddInt64(&m.failCount, 1)
		return fmt.Errorf("mock error delivering email to %s", job.Recipient.Email)
	}

	atomic.AddInt64(&m.sentCount, 1)
	return nil
}

// Stats returns total calls, successfully sent emails, and failed attempts.
func (m *MockSender) Stats() (sent, failed, total int64) {
	return atomic.LoadInt64(&m.sentCount), atomic.LoadInt64(&m.failCount), atomic.LoadInt64(&m.callCount)
}

// Reset clears all counters.
func (m *MockSender) Reset() {
	atomic.StoreInt64(&m.callCount, 0)
	atomic.StoreInt64(&m.sentCount, 0)
	atomic.StoreInt64(&m.failCount, 0)
}
