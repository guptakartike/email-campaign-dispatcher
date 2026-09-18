package sender

import (
	"fmt"
	"net/smtp"
	"strings"
)

type EmailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func Send(cfg EmailConfig, to, name, subject, body string) error {
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	personalisedBody := strings.ReplaceAll(body, "{{name}}", name)

	msg := []byte(
		fmt.Sprintf(
			"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
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
