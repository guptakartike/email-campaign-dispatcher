package sender

import (
	"fmt"
	"net/smtp"
	"strings"
)

// EmailConfig holds the SMTP connection parameters.
type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// Send delivers a single email via SMTP.
// It performs {{name}} template substitution in the body and constructs
// an RFC 5322–compliant message.
func Send(cfg EmailConfig, to, name, subject, body string) error {
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	personalisedBody := strings.ReplaceAll(body, "{{name}}", name)

	msg := []byte(
		fmt.Sprintf(
			"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s",
			cfg.From,
			to,
			subject,
			personalisedBody,
		),
	)

	return smtp.SendMail(
		cfg.Host+":"+cfg.Port,
		auth,
		cfg.From,
		[]string{to},
		msg,
	)
}
