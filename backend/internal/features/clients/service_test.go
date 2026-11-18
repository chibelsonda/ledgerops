package clients

import (
	"context"
	"testing"
)

type serviceRepository struct{ creates int }

func (r *serviceRepository) List(context.Context, string, ListParams) (Page, error) {
	return Page{}, nil
}
func (r *serviceRepository) Create(context.Context, string, Actor, MutationInput) (Client, error) {
	r.creates++
	return Client{}, nil
}
func (r *serviceRepository) Update(context.Context, string, string, Actor, MutationInput) (Client, error) {
	return Client{}, nil
}
func (r *serviceRepository) ListContacts(context.Context, string, string) ([]Contact, error) {
	return nil, nil
}
func (r *serviceRepository) CreateContact(context.Context, string, string, Actor, ContactInput) (Contact, error) {
	return Contact{}, nil
}
func (r *serviceRepository) UpdateContact(context.Context, string, string, Actor, ContactInput) (Contact, error) {
	return Contact{}, nil
}

func TestCreateRequiresValidPrimaryContact(t *testing.T) {
	repository := &serviceRepository{}
	service := NewService(repository)
	_, err := service.Create(context.Background(), "organization", Actor{}, MutationInput{Name: "Acme", PaymentTermsDays: 30, RemindersEnabled: true, ReminderDelayDays: 3, PrimaryContact: &PrimaryContactInput{Name: "A", Email: "invalid"}})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if repository.creates != 0 {
		t.Fatal("repository called for invalid client")
	}
}
