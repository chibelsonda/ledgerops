package projects

import (
	"context"
	"errors"
	"testing"
)

type repositoryStub struct {
	createInput      CreateInput
	changeOrderInput ChangeOrderInput
	invoiceInput     InvoiceInput
	actor            Actor
}

func (s *repositoryStub) List(context.Context, string) ([]ProjectSummary, error) { return nil, nil }
func (s *repositoryStub) ListClients(context.Context, string) ([]ClientOption, error) {
	return nil, nil
}
func (s *repositoryStub) Get(context.Context, string, string) (Detail, error) { return Detail{}, nil }
func (s *repositoryStub) Create(_ context.Context, _ string, actor Actor, input CreateInput) (Detail, error) {
	s.actor, s.createInput = actor, input
	return Detail{}, nil
}
func (s *repositoryStub) DecideContract(context.Context, string, string, Actor, DecisionInput) (Detail, error) {
	return Detail{}, nil
}
func (s *repositoryStub) CreateChangeOrder(_ context.Context, _, _ string, actor Actor, input ChangeOrderInput) (Detail, error) {
	s.actor, s.changeOrderInput = actor, input
	return Detail{}, nil
}
func (s *repositoryStub) DecideChangeOrder(context.Context, string, string, Actor, DecisionInput) (Detail, error) {
	return Detail{}, nil
}
func (s *repositoryStub) CreateMilestone(context.Context, string, string, Actor, MilestoneInput) (Detail, error) {
	return Detail{}, nil
}
func (s *repositoryStub) InvoiceMilestone(_ context.Context, _, _ string, actor Actor, input InvoiceInput) (Detail, error) {
	s.actor, s.invoiceInput = actor, input
	return Detail{}, nil
}

func TestCreateNormalizesProjectAndContractIdentifiers(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository)
	input := CreateInput{
		ClientID: "11111111-1111-4111-8111-111111111111", ProjectCode: " prj-10 ", Name: " Launch ",
		CurrencyCode: "usd", StartOn: "2026-11-18", ContractNumber: " ctr-10 ", ContractTitle: " Delivery ", OriginalValueMinor: 100_000,
	}
	actor := Actor{UserID: "user", Email: "controller@example.com"}
	if _, err := service.Create(context.Background(), "organization", actor, input); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repository.createInput.ProjectCode != "PRJ-10" || repository.createInput.ContractNumber != "CTR-10" || repository.createInput.CurrencyCode != "USD" {
		t.Fatalf("normalized input = %#v", repository.createInput)
	}
	if repository.actor != actor {
		t.Fatalf("actor = %#v, want %#v", repository.actor, actor)
	}
}

func TestChangeOrderRejectsZeroAmount(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository)
	_, err := service.CreateChangeOrder(context.Background(), "organization", "contract", Actor{}, ChangeOrderInput{
		ChangeOrderNumber: "CO-1", Title: "Scope", Description: "Changed scope", AmountMinor: 0,
	})
	var validation ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("CreateChangeOrder() error = %v, want ValidationError", err)
	}
	if repository.changeOrderInput.ChangeOrderNumber != "" {
		t.Fatal("repository was called for an invalid change order")
	}
}

func TestMilestoneInvoiceRejectsInvalidDateOrder(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository)
	_, err := service.InvoiceMilestone(context.Background(), "organization", "milestone", Actor{}, InvoiceInput{
		InvoiceNumber: "INV-1", IssuedOn: "2026-11-18", DueOn: "2026-11-17",
	})
	var validation ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("InvoiceMilestone() error = %v, want ValidationError", err)
	}
}
