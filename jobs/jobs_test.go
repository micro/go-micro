package jobs

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestMemoryQueue(t *testing.T) {
	ctx := context.Background()
	queue := NewMemoryQueue()

	job := NewJob("test", map[string]interface{}{"key": "value"})
	
	if err := queue.Enqueue(ctx, job); err != nil {
		t.Fatalf("failed to enqueue: %v", err)
	}

	dequeued, err := queue.Dequeue(ctx)
	if err != nil {
		t.Fatalf("failed to dequeue: %v", err)
	}

	if dequeued.ID != job.ID {
		t.Error("dequeued job ID mismatch")
	}

	if dequeued.GetString("key") != "value" {
		t.Error("payload mismatch")
	}
}

func TestWorker(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queue := NewMemoryQueue()
	worker := NewWorker(queue, WithConcurrency(1), WithPollInterval(100*time.Millisecond))

	var processed int32
	worker.Register("test", func(ctx context.Context, job *Job) error {
		atomic.AddInt32(&processed, 1)
		return nil
	})

	worker.Start(ctx)
	defer worker.Stop()

	// Enqueue a job
	job := NewJob("test", nil)
	queue.Enqueue(ctx, job)

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	if atomic.LoadInt32(&processed) != 1 {
		t.Errorf("expected 1 processed job, got %d", processed)
	}
}

func TestJobDelay(t *testing.T) {
	ctx := context.Background()
	queue := NewMemoryQueue()

	job := NewJob("test", nil).Delay(time.Hour)
	queue.Enqueue(ctx, job)

	// Should not dequeue a delayed job
	dequeued, _ := queue.Dequeue(ctx)
	if dequeued != nil {
		t.Error("should not dequeue delayed job")
	}
}

func TestJobRetries(t *testing.T) {
	job := NewJob("test", nil).Retries(5)
	if job.MaxRetry != 5 {
		t.Errorf("expected MaxRetry=5, got %d", job.MaxRetry)
	}
}
