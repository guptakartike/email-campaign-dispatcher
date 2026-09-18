package dispatcher

import (
	"log"
	"sync"
	"time"

	"github.com/guptakartike/email-dispatcher/internal/loader"
	"github.com/guptakartike/email-dispatcher/internal/sender"
)

type Job struct {
	Recipient loader.Recipient
	Subject   string
	Body      string
}

type Result struct {
	Success bool
}

func StartWorker(
	id int,
	jobs <-chan Job,
	results chan<- Result,
	wg *sync.WaitGroup,
	cfg sender.EmailConfig,
) {
	defer wg.Done()
	for job := range jobs {
		var err error

		for attempt := 1; attempt <= 3; attempt++ {
			err = sender.Send(
				cfg,
				job.Recipient.Email,
				job.Recipient.Name,
				job.Subject,
				job.Body,
			)

			if err == nil {
				log.Printf("[Worker %d] Sent to %s", id, job.Recipient.Email)
				results <- Result{Success: true}
				break
			}
			sleep := time.Duration(1<<attempt) * time.Second
			time.Sleep(sleep)
		}

		if err!= nil{
			log.Printf("[Worker %d] FAILED %s", id, job.Recipient.Email)
			results<-Result{Success: false}
		}
	}
}
