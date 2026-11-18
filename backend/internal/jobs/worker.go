package jobs

import (
	"context"
	"log/slog"
	"time"

	ledgersync "github.com/portfolio/ledgerops/backend/internal/features/sync"
)

type Repository interface {
	ClaimJob(context.Context) (*ledgersync.Job, error)
	CompleteJob(context.Context, ledgersync.Job) error
	FailJob(context.Context, ledgersync.Job, error) error
}

type Worker struct {
	repository   Repository
	logger       *slog.Logger
	pollInterval time.Duration
}

func NewWorker(repository Repository, logger *slog.Logger, pollInterval time.Duration) *Worker {
	return &Worker{repository: repository, logger: logger, pollInterval: pollInterval}
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

	// The portfolio demo synchronizes its seeded QuickBooks-shaped snapshot. A real
	// QBO client can implement the same boundary without changing queue semantics.
	if err := w.repository.CompleteJob(ctx, *job); err != nil {
		w.logger.Error("complete sync job", "job_id", job.ID, "error", err)
		if failErr := w.repository.FailJob(ctx, *job, err); failErr != nil {
			w.logger.Error("reschedule sync job", "job_id", job.ID, "error", failErr)
		}
		return
	}
	w.logger.Info("sync job complete", "job_id", job.ID)
}
