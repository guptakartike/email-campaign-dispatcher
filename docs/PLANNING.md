# Development Plan

A phased development plan and progress record for **MailPunk** — from initial CSV loading to full concurrent dispatch, rate limiting, and performance benchmarking.

---

## Completed Phases

### Phase 1: CSV Recipient Loader ✅

**Goal:** Parse recipient records from CSV into typed Go structs.

- Implemented `models.Recipient` in `internal/models/recipients.go`.
- Implemented `LoadRecipients(filename)` in `internal/loader/csv.go`.
- Added dynamic column position detection (`name,email` vs `email,name`).
- Added row and header validation with descriptive errors.

---

### Phase 2: Email Job Model ✅

**Goal:** Define the data structures flowing through the system channels.

- Implemented `models.EmailJob` in `internal/models/job.go`.
- Encapsulates `Recipient`, `Subject`, and `Body`.
- Exported and decoupled for use across loader, dispatcher, and sender packages.

---

### Phase 3: Producer Stage ✅

**Goal:** Decouple job generation from the main orchestrator.

- Created `dispatcher.Produce(recipients, jobsChan, subject, body)` in `internal/dispatcher/producer.go`.
- Iterates over recipients, constructs `EmailJob` structs, and pushes them to the channel.
- Safely closes the channel after the last recipient to signal consumer termination.

---

### Phase 4: Channels & Pipeline Orchestration ✅

**Goal:** Establish non-blocking buffered pipeline connecting producer to workers.

- Configured buffered `jobs := make(chan models.EmailJob, len(recipients))` in `cmd/main.go`.
- Producer runs concurrently as a goroutine: `go dispatcher.Produce(...)`.
- Workers drain jobs until the channel closes.

---

### Phase 5: Concurrent Worker Pool ✅

**Goal:** Execute concurrent email sending across multiple worker goroutines.

- Implemented `dispatcher.StartWorkers()` in `internal/dispatcher/worker.go`.
- Spawns `workerCount` goroutines reading from the shared `jobs` channel.
- Synchronized completion using `sync.WaitGroup`.
- Tracks atomic execution metrics (`WorkerStats`).

---

### Phase 6: SMTP Sender Integration ✅

**Goal:** Send RFC 5322 plain text emails via SMTP.

- Implemented `sender.Send()` in `internal/sender/smtp.go` using standard library `net/smtp`.
- Configured authentication via `smtp.PlainAuth`.
- Supports dynamic `{{name}}` personalization in the email body.
- Verified delivery against Mailtrap SMTP sandbox.

---

### Phase 7: Swappable Sender Interface & Mocking ✅

**Goal:** Decouple delivery implementation from the worker pool.

- Defined `sender.Sender` interface in `internal/sender/sender.go`.
- Implemented `sender.SMTPSender` adapter wrapping `net/smtp`.
- Created `sender.MockSender` in `internal/sender/mock.go` for zero-network benchmarking and simulated latency/error injection.

---

### Phase 8: Resilient Retry Mechanism ✅

**Goal:** Handle transient network or rate-limit failures without dropping jobs.

- Implemented `dispatcher.RetryConfig` with `MaxAttempts` and `Delay`.
- Configured via `MAX_RETRIES` (default 3) and `RETRY_DELAY_SECONDS` (default 2).
- Per-worker independent backoff: failing attempts sleep and retry without blocking other workers.
- Logs permanent failures only after all attempts are exhausted.

---

### Phase 9: Shared Concurrency Rate Limiter ✅

**Goal:** Ensure multiple workers collectively respect external SMTP sending quotas.

- Implemented `dispatcher.RateLimiter` in `internal/dispatcher/ratelimiter.go` using `time.Ticker`.
- Configured via `EMAILS_PER_SECOND` (default 1.0).
- Shared across all workers: every send attempt (initial and retried) calls `limiter.Wait()`.
- Supports nil receiver for transparent rate limiter bypassing.

---

### Phase 10: Performance Benchmark Suite ✅

**Goal:** Accurately measure dispatcher throughput and worker pool scaling.

- Implemented standalone CLI benchmark runner in `cmd/benchmark/main.go`.
- Benchmarks 500 and 1,000 recipients across 1, 5, 10, and 20 workers.
- Verified raw in-memory throughput (up to 13.1M emails/sec).
- Verified concurrent I/O scaling under 2ms simulated latency (near-linear 19.9x speedup with 20 workers).
- Verified exact 100 emails/sec constraint under rate limiting.

---

## Planned Future Phases

### Phase 11: AWS SES Integration 📋

**Goal:** Add Amazon Simple Email Service (SES) as an alternative production backend.

- Implement `SESSender` conforming to `sender.Sender`.
- Integrate `aws-sdk-go-v2/service/sesv2`.
- Add backend selection via `EMAIL_BACKEND=ses|smtp`.

---

### Phase 12: Production Readiness & Operations 📋

**Goal:** Enterprise resilience, graceful shutdowns, and formatting.

- Signal handling (`SIGINT`, `SIGTERM`) for graceful in-flight job completion.
- Context cancellation propagation (`context.Context`).
- Multipart MIME for HTML email support.
- Structured logging with Go's `log/slog`.
