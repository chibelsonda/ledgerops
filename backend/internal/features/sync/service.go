package sync

import "context"

type Repository interface {
	ListRuns(context.Context, string) ([]Run, error)
	Enqueue(context.Context, string) (Run, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) ListRuns(ctx context.Context, organizationID string) ([]Run, error) {
	return s.repository.ListRuns(ctx, organizationID)
}

func (s *Service) Enqueue(ctx context.Context, organizationID string) (Run, error) {
	return s.repository.Enqueue(ctx, organizationID)
}
