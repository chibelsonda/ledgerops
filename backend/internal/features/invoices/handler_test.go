package invoices

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

const (
	testOrganizationID = "11111111-1111-4111-8111-111111111111"
	testClientID       = "21111111-1111-4111-8111-111111111111"
	testInvoiceID      = "31111111-1111-4111-8111-111111111111"
)

type stubRepository struct {
	page          Page
	listParams    ListParams
	created       MutationInput
	updated       MutationInput
	createdActor  string
	updatedActor  string
	auditEvents   []AuditEvent
	payments      []Payment
	paymentInput  PaymentInput
	paymentActor  Actor
	clientOptions []ClientOption
}

func (s *stubRepository) List(_ context.Context, _ string, params ListParams) (Page, error) {
	s.listParams = params
	return s.page, nil
}
func (s *stubRepository) ListClients(context.Context, string) ([]ClientOption, error) {
	return s.clientOptions, nil
}
func (s *stubRepository) Create(_ context.Context, _, actor string, input MutationInput) (Invoice, error) {
	s.created, s.createdActor = input, actor
	return Invoice{ID: testInvoiceID, InvoiceNumber: input.InvoiceNumber}, nil
}
func (s *stubRepository) Update(_ context.Context, _, _, actor string, input MutationInput) (Invoice, error) {
	s.updated, s.updatedActor = input, actor
	return Invoice{ID: testInvoiceID, InvoiceNumber: input.InvoiceNumber}, nil
}
func (s *stubRepository) ListAuditEvents(context.Context, string, string) ([]AuditEvent, error) {
	return s.auditEvents, nil
}
func (s *stubRepository) ListPayments(context.Context, string, string) ([]Payment, error) {
	return s.payments, nil
}
func (s *stubRepository) RecordPayment(_ context.Context, _, _ string, actor Actor, input PaymentInput) (PaymentResult, error) {
	s.paymentActor, s.paymentInput = actor, input
	return PaymentResult{Payment: Payment{ID: testInvoiceID, AmountMinor: input.AmountMinor}}, nil
}

func TestListParsesPaginationAndFilters(t *testing.T) {
	repository := &stubRepository{page: Page{Items: []Invoice{{ID: testInvoiceID}}, Pagination: Pagination{Page: 2, PageSize: 5, TotalItems: 8, TotalPages: 2}}}
	handler := newTestHandler(repository)
	request := authenticatedRequest(http.MethodGet, "/api/v1/invoices?query=Arc&status=overdue&page=2&pageSize=5", nil)
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	want := ListParams{Query: "Arc", Status: "overdue", Page: 2, PageSize: 5}
	if repository.listParams != want {
		t.Fatalf("params = %#v, want %#v", repository.listParams, want)
	}
}

func TestCreateNormalizesAndPassesAuthenticatedActor(t *testing.T) {
	repository := &stubRepository{}
	handler := newTestHandler(repository)
	body := map[string]any{
		"clientId": testClientID, "invoiceNumber": " INV-NEW ", "status": "OPEN",
		"currencyCode": "usd", "amountMinor": 12500, "issuedOn": "2026-11-18", "dueOn": "2026-12-18",
	}
	request := authenticatedRequest(http.MethodPost, "/api/v1/invoices", body)
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", response.Code, response.Body.String())
	}
	if repository.created.InvoiceNumber != "INV-NEW" || repository.created.CurrencyCode != "USD" || repository.created.Status != "open" {
		t.Fatalf("normalized input = %#v", repository.created)
	}
	if repository.createdActor != "controller@ledgerops.demo" {
		t.Fatalf("actor = %q", repository.createdActor)
	}
}

func TestCreateRejectsPaymentDerivedStatus(t *testing.T) {
	repository := &stubRepository{}
	handler := newTestHandler(repository)
	body := map[string]any{
		"clientId": testClientID, "invoiceNumber": "INV-PAID", "status": "paid",
		"currencyCode": "USD", "amountMinor": 12500, "issuedOn": "2026-11-18", "dueOn": "2026-12-18",
	}
	request := authenticatedRequest(http.MethodPost, "/api/v1/invoices", body)
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.Code)
	}
	if repository.created.InvoiceNumber != "" {
		t.Fatal("repository create was called for a payment-derived status")
	}
}

func TestUpdateRequiresAuditNote(t *testing.T) {
	repository := &stubRepository{}
	handler := newTestHandler(repository)
	body := map[string]any{
		"clientId": testClientID, "invoiceNumber": "INV-NEW", "status": "open",
		"currencyCode": "USD", "amountMinor": 12500, "issuedOn": "2026-11-18", "dueOn": "2026-12-18",
		"auditNote": "",
	}
	request := authenticatedRequest(http.MethodPatch, "/api/v1/invoices/"+testInvoiceID, body)
	request.SetPathValue("invoiceID", testInvoiceID)
	response := httptest.NewRecorder()

	handler.Update(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.Code)
	}
}

func TestInvalidPageIsRejected(t *testing.T) {
	handler := newTestHandler(&stubRepository{})
	request := authenticatedRequest(http.MethodGet, "/api/v1/invoices?page=0", nil)
	response := httptest.NewRecorder()
	handler.List(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.Code)
	}
}

func TestRecordPaymentNormalizesInputAndUsesAuthenticatedActor(t *testing.T) {
	repository := &stubRepository{}
	handler := newTestHandler(repository)
	body := map[string]any{
		"amountMinor": 30000,
		"receivedOn":  "2026-11-18",
		"reference":   " BANK-123 ",
		"auditNote":   " Confirmed bank deposit. ",
	}
	request := authenticatedRequest(http.MethodPost, "/api/v1/invoices/"+testInvoiceID+"/payments", body)
	request.SetPathValue("invoiceID", testInvoiceID)
	response := httptest.NewRecorder()

	handler.RecordPayment(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", response.Code, response.Body.String())
	}
	if repository.paymentInput.Reference != "BANK-123" || repository.paymentInput.AuditNote != "Confirmed bank deposit." {
		t.Fatalf("normalized payment = %#v", repository.paymentInput)
	}
	if repository.paymentActor.Email != "controller@ledgerops.demo" || repository.paymentActor.UserID == "" {
		t.Fatalf("payment actor = %#v", repository.paymentActor)
	}
}

func newTestHandler(repository Repository) *Handler {
	return NewHandler(NewService(repository), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func authenticatedRequest(method, target string, body any) *http.Request {
	var encoded io.Reader
	if body != nil {
		payload, _ := json.Marshal(body)
		encoded = bytes.NewReader(payload)
	}
	request := httptest.NewRequest(method, target, encoded)
	principal := identity.Principal{
		OrganizationID: testOrganizationID,
		UserID:         "82222222-2222-7222-8222-222222222222",
		Email:          "controller@ledgerops.demo",
		Role:           identity.RoleController,
	}
	return request.WithContext(identity.WithPrincipal(request.Context(), principal))
}
