package adjustments

import "context"

type Repository interface {
	List(context.Context, string, string) ([]Adjustment, error)
	Create(context.Context, string, string, Actor, CreateInput) (Adjustment, error)
	Decide(context.Context, string, string, Actor, DecisionInput) (DecisionResult, error)
}

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, organizationID, invoiceID string) ([]Adjustment, error) {
	return s.repository.List(ctx, organizationID, invoiceID)
}

func (s *Service) Create(ctx context.Context, organizationID, invoiceID string, actor Actor, input CreateInput) (Adjustment, error) {
	normalizeCreate(&input)
	if message := validateCreate(input); message != "" {
		return Adjustment{}, ValidationError{Message: message}
	}
	return s.repository.Create(ctx, organizationID, invoiceID, actor, input)
}

func (s *Service) Decide(ctx context.Context, organizationID, adjustmentID string, actor Actor, input DecisionInput) (DecisionResult, error) {
	normalizeDecision(&input)
	if message := validateDecision(input); message != "" {
		return DecisionResult{}, ValidationError{Message: message}
	}
	return s.repository.Decide(ctx, organizationID, adjustmentID, actor, input)
}
