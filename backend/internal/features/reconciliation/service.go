package reconciliation

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidResolution = errors.New("resolution note must contain 3 to 500 characters")

type Repository interface {
	List(context.Context, string) ([]Item, error)
	Resolve(context.Context, string, string, string, string) error
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, organizationID string) ([]Item, error) {
	return s.repository.List(ctx, organizationID)
}

func (s *Service) Resolve(ctx context.Context, organizationID, itemID, actor, note string) error {
	note = strings.TrimSpace(note)
	if len(note) < 3 || len(note) > 500 {
		return ErrInvalidResolution
	}
	return s.repository.Resolve(ctx, organizationID, itemID, actor, note)
}
