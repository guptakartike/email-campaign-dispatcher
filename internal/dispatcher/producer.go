package dispatcher

import (
	"github.com/guptakartike/email-dispatcher/internal/models"
)

// Produce converts a slice of recipients into EmailJob values and sends
// them onto the provided channel. It closes the channel after all jobs
// have been produced, signalling consumers that no more work is coming.
//
// Ownership contract:
//   - The caller creates the channel.
//   - Produce writes to and closes the channel.
//   - Consumers read from the channel.
func Produce(recipients []models.Recipient, jobs chan<- models.EmailJob, subject, body string) {
	for _, r := range recipients {
		jobs <- models.EmailJob{
			Recipient: r,
			Subject:   subject,
			Body:      body,
		}
	}
	close(jobs)
}
