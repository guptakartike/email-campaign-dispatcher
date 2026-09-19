package models

// EmailJob represents a unit of work to be dispatched to a worker.
// It bundles a Recipient with the email content to send.
type EmailJob struct {
	Recipient Recipient
	Subject   string
	Body      string
}
