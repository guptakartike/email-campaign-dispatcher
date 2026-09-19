package sender

import (
	"github.com/guptakartike/email-dispatcher/internal/models"
)

// Sender defines the interface for delivering email jobs.
type Sender interface {
	Send(job models.EmailJob) error
}

// SMTPSender delivers emails using an SMTP server configuration.
type SMTPSender struct {
	cfg EmailConfig
}

// NewSMTPSender creates a new SMTPSender with the provided configuration.
func NewSMTPSender(cfg EmailConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

// Send implements the Sender interface for SMTPSender.
func (s *SMTPSender) Send(job models.EmailJob) error {
	return Send(s.cfg, job.Recipient.Email, job.Recipient.Name, job.Subject, job.Body)
}
