package dashboard

import "context"

type Repository interface {
	Get(context.Context, string) (View, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Get(ctx context.Context, organizationID string) (View, error) {
	return s.repository.Get(ctx, organizationID)
}
