# Architecture

## Overview

Email Dispatcher is a high-throughput, concurrent email campaign delivery system built in Go. It reads recipient data from a CSV file, constructs personalized email jobs, and dispatches them through a pool of concurrent workers using Go's native concurrency primitives (goroutines and channels).

The system follows a **Producer → Channel → Worker Pool → Rate Limiter & Retry → Sender** architecture, enabling efficient parallel delivery with configurable concurrency, a shared token-bucket rate limiter, independent retry backoff on failure, and a swappable delivery interface supporting both production SMTP and high-speed mock benchmarking.

---

## System Architecture Diagram

```mermaid
graph TB
    subgraph Input["Data Source"]
        CSV["📄 data/recipients.csv"]
    end

    subgraph Loader["internal/loader"]
        CSVLoader["CSV Loader<br/>LoadRecipients()<br/>(Dynamic Header Mapping)"]
    end

    subgraph Models["internal/models"]
        Recipient["Recipient{Name, Email}"]
        EmailJob["EmailJob{Recipient, Subject, Body}"]
    end

    subgraph Config["Configuration"]
        ENV[".env"]
        GoDotEnv["godotenv"]
    end

    subgraph Orchestrator["cmd/main.go (Orchestrator)"]
        JobsChan["jobs channel<br/>(chan models.EmailJob)"]
    end

    subgraph Dispatcher["internal/dispatcher"]
        Producer["Producer<br/>Produce()"]
        Limiter["RateLimiter<br/>(time.Ticker token bucket)"]
        
        subgraph WorkerPool["Worker Pool"]
            W1["Worker 1"]
            W2["Worker 2"]
            WN["Worker N"]
        end
        
        Retry["sendWithRetry()<br/>(MaxAttempts, Delay)"]
    end

    subgraph Sender["internal/sender (Sender Interface)"]
        SMTPSender["SMTPSender<br/>(net/smtp + PlainAuth)"]
        MockSender["MockSender<br/>(Benchmarking & Testing)"]
        SES["SESSender<br/>(Planned)"]
    end

    subgraph External["External Network"]
        Mailtrap["Mailtrap / SMTP Server"]
    end

    CSV --> CSVLoader
    CSVLoader --> Recipient
    ENV --> GoDotEnv --> Orchestrator
    Recipient --> Producer
    Producer -->|"EmailJob"| JobsChan

    JobsChan --> W1 & W2 & WN
    W1 & W2 & WN --> Retry
    Retry <--> Limiter
    Retry --> SMTPSender & MockSender
    SMTPSender --> Mailtrap
    SMTPSender -.->|"future"| SES

    style SES stroke-dasharray: 5 5
```

---

## Component Responsibilities

### 1. `cmd/main.go` — Campaign Pipeline Orchestrator
The CLI entry point responsible for:
- Loading environment configuration (`.env` via `godotenv`).
- Parsing retry parameters (`MAX_RETRIES`, `RETRY_DELAY_SECONDS`) and rate limits (`EMAILS_PER_SECOND`).
- Calling `loader.LoadRecipients()` to parse recipient records.
- Instantiating the shared `dispatcher.RateLimiter`.
- Allocating the buffered `jobs` channel.
- Spawning `dispatcher.Produce` in a background goroutine.
- Initializing `sender.NewSMTPSender(cfg)` and launching the worker pool via `dispatcher.StartWorkers`.

### 2. `cmd/benchmark/main.go` — Performance Benchmark Suite
A standalone runner for measuring dispatcher performance without network interference:
- Benchmarks 500 and 1,000 synthetic recipients across 1, 5, 10, and 20 workers.
- Measures raw in-memory throughput with the rate limiter bypassed.
- Simulates realistic I/O latency (e.g. 2ms per send) to evaluate horizontal scaling.
- Evaluates rate-limiting enforcement (`EMAILS_PER_SECOND=100`).
- Validates the retry mechanism under simulated failure rates.

### 3. `internal/models` — Shared Data Models
- [`Recipient`](file:///Users/kartikegupta/Documents/GitHub/email-dispatcher/internal/models/recipients.go): Encapsulates recipient `Name` and `Email`.
- [`EmailJob`](file:///Users/kartikegupta/Documents/GitHub/email-dispatcher/internal/models/job.go): Encapsulates a `Recipient`, `Subject`, and `Body` template.

### 4. `internal/loader` — Flexible CSV Parser
- Implements `LoadRecipients(filename) ([]models.Recipient, error)`.
- Inspects header columns dynamically to support both `name,email` and `email,name` layouts.
- Trims leading/trailing whitespace on all fields.
- Falls back to default indices (0 for email, 1 for name) if headers are not recognized.

### 5. `internal/dispatcher` — Concurrency & Regulation
- **Producer ([`producer.go`](file:///Users/kartikegupta/Documents/GitHub/email-dispatcher/internal/dispatcher/producer.go))**: Converts recipients into `EmailJob` structs, pushes them to `jobs chan<- models.EmailJob`, and closes the channel when done.
- **Worker Pool ([`worker.go`](file:///Users/kartikegupta/Documents/GitHub/email-dispatcher/internal/dispatcher/worker.go))**: Spawns N worker goroutines reading from the shared channel. Coordinates completion via `sync.WaitGroup` and collects atomic telemetry (`WorkerStats`).
- **Retry System ([`worker.go`](file:///Users/kartikegupta/Documents/GitHub/email-dispatcher/internal/dispatcher/worker.go))**: `sendWithRetry()` attempts delivery up to `MaxAttempts` with `Delay` between attempts. Each worker retries independently without blocking the pool.
- **Shared Rate Limiter ([`ratelimiter.go`](file:///Users/kartikegupta/Documents/GitHub/email-dispatcher/internal/dispatcher/ratelimiter.go))**: A thread-safe token-bucket built with `time.Ticker`. All workers call `limiter.Wait()` before every delivery attempt. Supports transparent bypass when `limiter` is `nil`.

### 6. `internal/sender` — Delivery Abstraction
- **`Sender` Interface ([`sender.go`](file:///Users/kartikegupta/Documents/GitHub/email-dispatcher/internal/sender/sender.go))**: Defines `Send(job models.EmailJob) error`, decoupling the delivery mechanism from the worker pool.
- **`SMTPSender` ([`sender.go`](file:///Users/kartikegupta/Documents/GitHub/email-dispatcher/internal/sender/sender.go))**: Wraps standard library `net/smtp` and `PlainAuth`. Substitutes `{{name}}` in the body and constructs RFC 5322 messages.
- **`MockSender` ([`mock.go`](file:///Users/kartikegupta/Documents/GitHub/email-dispatcher/internal/sender/mock.go))**: In-memory delivery simulator for benchmarks with configurable per-send latency and simulated failure rates.

---

## Data Flow & Lifecycle

```mermaid
sequenceDiagram
    autonumber
    participant Main as cmd/main.go
    participant Loader as internal/loader
    participant Producer as internal/dispatcher/producer
    participant Channel as jobs channel
    participant Worker as Worker Goroutine
    participant Limiter as Shared RateLimiter
    participant Sender as sender.Sender

    Main->>Loader: LoadRecipients("data/recipients.csv")
    Loader-->>Main: []models.Recipient
    Main->>Limiter: NewRateLimiter(emailsPerSecond)
    Main->>Channel: make(chan models.EmailJob, len(recipients))
    Main->>Producer: go Produce(recipients, jobs, subject, body)
    
    par Produce Jobs
        Producer->>Channel: push EmailJob
        Producer->>Channel: close(jobs)
    and Consume Jobs
        Main->>Worker: go worker(1..N, jobs, limiter, sender)
        loop Each EmailJob
            Channel-->>Worker: job
            Worker->>Limiter: limiter.Wait()
            Limiter-->>Worker: token ready
            Worker->>Sender: Send(job)
            alt Success
                Sender-->>Worker: nil
            else Failure
                Sender-->>Worker: error
                Worker->>Worker: sleep(retry.Delay)
                Worker->>Limiter: limiter.Wait()
                Worker->>Sender: Send(job) [retry]
            end
        end
    end
    Worker-->>Main: wg.Done()
```

---

## Concurrency & Synchronization Model

1. **Producer-Consumer via Go Channels**:
   - The channel buffer matches the total recipient count (`len(recipients)`), ensuring that the producer never blocks while feeding jobs.
   - When the producer finishes, it calls `close(jobs)`. In Go, reading from a closed channel yields remaining buffered items, then returns `false` on the ok-check (`for job := range jobs`), causing workers to terminate cleanly.

2. **Shutdown Coordination via `sync.WaitGroup`**:
   - `StartWorkers` initializes `sync.WaitGroup`, calls `wg.Add(1)` per worker, and defers `wg.Done()` inside each worker.
   - `wg.Wait()` guarantees all in-flight emails, retries, and cleanups finish before `main()` exits.

3. **Telemetry via `sync/atomic`**:
   - Success, failure, and retry counts in `WorkerStats` are incremented using lockless atomic primitives (`atomic.AddInt64`), eliminating mutex lock contention during high-speed benchmarks.
