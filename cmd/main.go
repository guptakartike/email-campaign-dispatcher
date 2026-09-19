package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/guptakartike/email-dispatcher/internal/dispatcher"
	"github.com/guptakartike/email-dispatcher/internal/loader"
	"github.com/guptakartike/email-dispatcher/internal/models"
	"github.com/guptakartike/email-dispatcher/internal/sender"
	"github.com/joho/godotenv"
)

const (
	workerCount = 5
	subject     = "Welcome to MailPunk!"
	body        = "Hi {{name}},\n\nWelcome to our email campaign service.\n\nBest regards,\nMailPunk Team"
)

func main() {
	// Step 1: Load environment variables.
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load .env: %v", err)
	}

	cfg := sender.EmailConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("FROM_EMAIL"),
	}

	retryCfg := loadRetryConfig()

	// Step 2: Load recipients from CSV.
	recipients, err := loader.LoadRecipients("data/recipients.csv")
	if err != nil {
		log.Fatalf("Failed to load recipients: %v", err)
	}

	fmt.Printf("Loaded %d recipient(s).\n", len(recipients))
	fmt.Printf("Retry config: max %d attempts, %v delay.\n", retryCfg.MaxAttempts, retryCfg.Delay)

	// Step 3: Create channel and produce jobs.
	jobs := make(chan models.EmailJob, len(recipients))
	go dispatcher.Produce(recipients, jobs, subject, body)

	// Step 4: Start worker pool and wait for completion.
	fmt.Printf("Starting %d workers...\n\n", workerCount)
	dispatcher.StartWorkers(jobs, workerCount, cfg, retryCfg)

	fmt.Println("\nAll jobs processed.")
}

// loadRetryConfig reads retry settings from environment variables
// with sensible defaults (3 attempts, 2 second delay).
func loadRetryConfig() dispatcher.RetryConfig {
	maxAttempts := 3
	delaySec := 2

	if v := os.Getenv("MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxAttempts = n
		}
	}

	if v := os.Getenv("RETRY_DELAY_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			delaySec = n
		}
	}

	return dispatcher.RetryConfig{
		MaxAttempts: maxAttempts,
		Delay:       time.Duration(delaySec) * time.Second,
	}
}
