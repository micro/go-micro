// Package jobs provides background job processing for go-micro.
// Similar to Sidekiq (Rails) or Spring Batch.
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	log "go-micro.dev/v5/logger"
)

// Job represents a background job.
type Job struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Status    JobStatus              `json:"status"`
	Attempts  int                    `json:"attempts"`
	MaxRetry  int                    `json:"max_retry"`
	Error     string                 `json:"error,omitempty"`
	RunAt     time.Time              `json:"run_at"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// JobStatus represents the status of a job.
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
	StatusRetrying  JobStatus = "retrying"
)

// Handler processes a job.
type Handler func(ctx context.Context, job *Job) error

// Queue is the job queue interface.
type Queue interface {
	// Enqueue adds a job to the queue.
	Enqueue(ctx context.Context, job *Job) error
	// Dequeue retrieves the next job from the queue.
	Dequeue(ctx context.Context) (*Job, error)
	// Complete marks a job as completed.
	Complete(ctx context.Context, job *Job) error
	// Fail marks a job as failed.
	Fail(ctx context.Context, job *Job, err error) error
	// Retry re-queues a job for retry.
	Retry(ctx context.Context, job *Job) error
}

// Worker processes jobs from a queue.
type Worker struct {
	queue      Queue
	handlers   map[string]Handler
	concurrency int
	pollInterval time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
}

// Option configures a worker.
type Option func(*Worker)

// WithConcurrency sets the number of concurrent workers.
func WithConcurrency(n int) Option {
	return func(w *Worker) { w.concurrency = n }
}

// WithPollInterval sets the queue poll interval.
func WithPollInterval(d time.Duration) Option {
	return func(w *Worker) { w.pollInterval = d }
}

// NewWorker creates a new job worker.
func NewWorker(queue Queue, opts ...Option) *Worker {
	w := &Worker{
		queue:        queue,
		handlers:     make(map[string]Handler),
		concurrency:  1,
		pollInterval: time.Second,
		stopCh:       make(chan struct{}),
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Register registers a handler for a job type.
func (w *Worker) Register(jobType string, handler Handler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[jobType] = handler
}

// Start starts the worker.
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("worker already running")
	}
	w.running = true
	w.mu.Unlock()

	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.run(ctx, i)
	}

	log.Infof("Job worker started with %d workers", w.concurrency)
	return nil
}

// Stop stops the worker gracefully.
func (w *Worker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()
	log.Info("Job worker stopped")
}

func (w *Worker) run(ctx context.Context, id int) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processNext(ctx)
		}
	}
}

func (w *Worker) processNext(ctx context.Context) {
	job, err := w.queue.Dequeue(ctx)
	if err != nil {
		return
	}
	if job == nil {
		return
	}

	w.mu.RLock()
	handler, ok := w.handlers[job.Type]
	w.mu.RUnlock()

	if !ok {
		log.Warnf("No handler for job type: %s", job.Type)
		w.queue.Fail(ctx, job, fmt.Errorf("no handler for job type: %s", job.Type))
		return
	}

	log.Infof("Processing job %s (type: %s, attempt: %d)", job.ID, job.Type, job.Attempts+1)
	job.Attempts++
	job.Status = StatusRunning

	err = handler(ctx, job)
	if err != nil {
		log.Errorf("Job %s failed: %v", job.ID, err)
		if job.Attempts < job.MaxRetry {
			log.Infof("Retrying job %s (attempt %d/%d)", job.ID, job.Attempts, job.MaxRetry)
			w.queue.Retry(ctx, job)
		} else {
			w.queue.Fail(ctx, job, err)
		}
		return
	}

	log.Infof("Job %s completed", job.ID)
	w.queue.Complete(ctx, job)
}

// NewJob creates a new job.
func NewJob(jobType string, payload map[string]interface{}) *Job {
	now := time.Now()
	return &Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Payload:   payload,
		Status:    StatusPending,
		Attempts:  0,
		MaxRetry:  3,
		RunAt:     now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Delay sets the job to run after a delay.
func (j *Job) Delay(d time.Duration) *Job {
	j.RunAt = time.Now().Add(d)
	return j
}

// Retries sets the max retry count.
func (j *Job) Retries(n int) *Job {
	j.MaxRetry = n
	return j
}

// MemoryQueue is an in-memory job queue for development/testing.
type MemoryQueue struct {
	jobs []*Job
	mu   sync.Mutex
}

// NewMemoryQueue creates a new in-memory queue.
func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{}
}

func (q *MemoryQueue) Enqueue(ctx context.Context, job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = append(q.jobs, job)
	return nil
}

func (q *MemoryQueue) Dequeue(ctx context.Context) (*Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	for i, job := range q.jobs {
		if job.Status == StatusPending && !job.RunAt.After(now) {
			q.jobs = append(q.jobs[:i], q.jobs[i+1:]...)
			return job, nil
		}
	}
	return nil, nil
}

func (q *MemoryQueue) Complete(ctx context.Context, job *Job) error {
	job.Status = StatusCompleted
	job.UpdatedAt = time.Now()
	return nil
}

func (q *MemoryQueue) Fail(ctx context.Context, job *Job, err error) error {
	job.Status = StatusFailed
	job.Error = err.Error()
	job.UpdatedAt = time.Now()
	return nil
}

func (q *MemoryQueue) Retry(ctx context.Context, job *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	job.Status = StatusPending
	job.RunAt = time.Now().Add(time.Second * time.Duration(job.Attempts*job.Attempts)) // Exponential backoff
	job.UpdatedAt = time.Now()
	q.jobs = append(q.jobs, job)
	return nil
}

// Scheduler for recurring jobs.
type Scheduler struct {
	queue   Queue
	entries []scheduleEntry
	stopCh  chan struct{}
	running bool
	mu      sync.Mutex
}

type scheduleEntry struct {
	interval time.Duration
	jobFunc  func() *Job
	lastRun  time.Time
}

// NewScheduler creates a new job scheduler.
func NewScheduler(queue Queue) *Scheduler {
	return &Scheduler{
		queue:  queue,
		stopCh: make(chan struct{}),
	}
}

// Every schedules a job to run at an interval.
func (s *Scheduler) Every(interval time.Duration, jobFunc func() *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, scheduleEntry{
		interval: interval,
		jobFunc:  jobFunc,
	})
}

// Start starts the scheduler.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.mu.Lock()
			for i := range s.entries {
				entry := &s.entries[i]
				if now.Sub(entry.lastRun) >= entry.interval {
					job := entry.jobFunc()
					s.queue.Enqueue(ctx, job)
					entry.lastRun = now
				}
			}
			s.mu.Unlock()
		}
	}
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()
	close(s.stopCh)
}

// JSON helpers for job payloads

// GetString gets a string from job payload.
func (j *Job) GetString(key string) string {
	if v, ok := j.Payload[key].(string); ok {
		return v
	}
	return ""
}

// GetInt gets an int from job payload.
func (j *Job) GetInt(key string) int {
	switch v := j.Payload[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}

// GetBool gets a bool from job payload.
func (j *Job) GetBool(key string) bool {
	if v, ok := j.Payload[key].(bool); ok {
		return v
	}
	return false
}
