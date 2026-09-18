package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/guptakartike/email-dispatcher/internal/loader"
	"github.com/guptakartike/email-dispatcher/internal/sender"
	"github.com/guptakartike/email-dispatcher/internal/dispatcher"
	"github.com/joho/godotenv"
)

const (
	subject = "Welcome!"
	body    = "Hi! {{name}}, \n\n Welcome to our service!"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load .env")

	}

	cfg := sender.EmailConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		Username: os.Getenv("SMTP_USER"),
		Password: os.Getenv("SMTP_PASS"),
		From:     os.Getenv("SMTP_FROM"),
	}

	recipients, err := loader.LoadRecipient("data/recipients.csv")
	if err != nil{
		log.Fatal(err)
	}

	jobs := make(chan dispatcher.Job,len(recipients))
	results:= make(chan dispatcher.Result, len(recipients))

	var wg sync.WaitGroup
	workerCount :=5

	for i:=0; i<workerCount; i++{
		wg.Add(1)
		go dispatcher.StartWorker(i, jobs, results, &wg, cfg)
	}

	for _,r:= range recipients{
		jobs <- dispatcher.Job{
			Recipient: r,
			Subject: subject,
			Body: body,
		}
	}
	close(jobs)

	wg.Wait()
	close(results)
	sent,failed :=0,0
	for res := range results{
		if res.Success{
			sent++
		} else{
			failed++
		}
	}

	fmt.Printf("Summary: Sent=%d Failed=%d", sent, failed)
}
