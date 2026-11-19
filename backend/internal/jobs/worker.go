package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	ledgersync "github.com/portfolio/ledgerops/backend/internal/features/sync"
)

type Repository interface {
	ClaimJob(context.Context) (*ledgersync.Job, error)
	CompleteJob(context.Context, ledgersync.Job, int) error
	FailJob(context.Context, ledgersync.Job, error, bool, time.Duration) error
}

type Processor interface {
	Synchronize(context.Context, ledgersync.Job) (int, error)
}

type Worker struct {
	repository   Repository
	processors   map[string]Processor
	logger       *slog.Logger
	pollInterval time.Duration
}

func NewWorker(repository Repository, processors map[string]Processor, logger *slog.Logger, pollInterval time.Duration) *Worker {
	return &Worker{repository: repository, processors: processors, logger: logger, pollInterval: pollInterval}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.processOne(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processOne(ctx)
		}
	}
}

func (w *Worker) processOne(ctx context.Context) {
	job, err := w.repository.ClaimJob(ctx)
	if err != nil {
		w.logger.Error("claim sync job", "error", err)
		return
	}
	if job == nil {
		return
	}
	w.logger.Info("processing sync job", "job_id", job.ID, "provider", job.Provider, "attempt", job.Attempts)

	processor, exists := w.processors[job.Provider]
	if !exists {
		w.fail(ctx, *job, fmt.Errorf("sync provider %q is not configured", job.Provider), false, 0)
		return
	}
	recordsProcessed, err := processor.Synchronize(ctx, *job)
	if err != nil {
		retryable, retryAfter := retryPolicy(err)
		w.fail(ctx, *job, err, retryable, retryAfter)
		return
	}
	if err := w.repository.CompleteJob(ctx, *job, recordsProcessed); err != nil {
		w.logger.Error("complete sync job", "job_id", job.ID, "error", err)
		if failErr := w.repository.FailJob(ctx, *job, err, true, 0); failErr != nil {
			w.logger.Error("reschedule sync job", "job_id", job.ID, "error", failErr)
		}
		return
	}
	w.logger.Info("sync job complete", "job_id", job.ID, "records_processed", recordsProcessed)
}

func (w *Worker) fail(ctx context.Context, job ledgersync.Job, cause error, retryable bool, retryAfter time.Duration) {
	w.logger.Error("process sync job", "job_id", job.ID, "error", cause, "retryable", retryable)
	if err := w.repository.FailJob(ctx, job, cause, retryable, retryAfter); err != nil {
		w.logger.Error("reschedule sync job", "job_id", job.ID, "error", err)
	}
}

func retryPolicy(err error) (bool, time.Duration) {
	var policy interface {
		Retryable() bool
		RetryAfter() time.Duration
	}
	if errors.As(err, &policy) {
		return policy.Retryable(), policy.RetryAfter()
	}
	return true, 0
}
