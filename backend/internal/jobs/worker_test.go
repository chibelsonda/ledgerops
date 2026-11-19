package jobs

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	ledgersync "github.com/portfolio/ledgerops/backend/internal/features/sync"
)

type workerRepositoryStub struct {
	job              *ledgersync.Job
	completedRecords int
	failed           bool
	retryable        bool
	retryAfter       time.Duration
}

func (r *workerRepositoryStub) ClaimJob(context.Context) (*ledgersync.Job, error) { return r.job, nil }
func (r *workerRepositoryStub) CompleteJob(_ context.Context, _ ledgersync.Job, records int) error {
	r.completedRecords = records
	return nil
}
func (r *workerRepositoryStub) FailJob(_ context.Context, _ ledgersync.Job, _ error, retryable bool, retryAfter time.Duration) error {
	r.failed, r.retryable, r.retryAfter = true, retryable, retryAfter
	return nil
}

type processorStub struct {
	records int
	err     error
}

func (p processorStub) Synchronize(context.Context, ledgersync.Job) (int, error) {
	return p.records, p.err
}

type policyError struct{}

func (policyError) Error() string             { return "rate limited" }
func (policyError) Retryable() bool           { return true }
func (policyError) RetryAfter() time.Duration { return 12 * time.Second }

func TestWorkerCompletesWithImportedRecordCount(t *testing.T) {
	repository := &workerRepositoryStub{job: &ledgersync.Job{ID: "job", Provider: "quickbooks-sandbox", OrganizationID: "organization"}}
	worker := NewWorker(repository, map[string]Processor{"quickbooks-sandbox": processorStub{records: 8}}, testLogger(), time.Second)
	worker.processOne(context.Background())
	if repository.completedRecords != 8 || repository.failed {
		t.Fatalf("repository = %#v", repository)
	}
}

func TestWorkerHonorsProviderRetryPolicy(t *testing.T) {
	repository := &workerRepositoryStub{job: &ledgersync.Job{ID: "job", Provider: "quickbooks-sandbox", OrganizationID: "organization"}}
	worker := NewWorker(repository, map[string]Processor{"quickbooks-sandbox": processorStub{err: policyError{}}}, testLogger(), time.Second)
	worker.processOne(context.Background())
	if !repository.failed || !repository.retryable || repository.retryAfter != 12*time.Second {
		t.Fatalf("repository = %#v", repository)
	}
}

func TestWorkerFailsUnknownProviderPermanently(t *testing.T) {
	repository := &workerRepositoryStub{job: &ledgersync.Job{ID: "job", Provider: "unknown", OrganizationID: "organization"}}
	worker := NewWorker(repository, nil, testLogger(), time.Second)
	worker.processOne(context.Background())
	if !repository.failed || repository.retryable {
		t.Fatalf("repository = %#v", repository)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
