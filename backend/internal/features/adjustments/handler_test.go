package adjustments

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
	testUserID         = "21111111-1111-4111-8111-111111111111"
	testInvoiceID      = "31111111-1111-4111-8111-111111111111"
	testAdjustmentID   = "41111111-1111-4111-8111-111111111111"
)

type stubRepository struct {
	createInput   CreateInput
	decisionInput DecisionInput
	actor         Actor
}

func (s *stubRepository) List(context.Context, string, string) ([]Adjustment, error) {
	return []Adjustment{}, nil
}
func (s *stubRepository) Create(_ context.Context, _, _ string, actor Actor, input CreateInput) (Adjustment, error) {
	s.actor, s.createInput = actor, input
	return Adjustment{ID: testAdjustmentID}, nil
}
func (s *stubRepository) Decide(_ context.Context, _, _ string, actor Actor, input DecisionInput) (DecisionResult, error) {
	s.actor, s.decisionInput = actor, input
	return DecisionResult{Adjustment: Adjustment{ID: testAdjustmentID}}, nil
}

func TestCreateNormalizesJustificationAndUsesPrincipal(t *testing.T) {
	repository := &stubRepository{}
	handler := NewHandler(NewService(repository), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := authenticatedRequest(http.MethodPost, map[string]any{
		"kind": "CREDIT_NOTE", "amountMinor": 5000, "reason": "  Pricing concession  ",
	})
	request.SetPathValue("invoiceID", testInvoiceID)
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", response.Code, response.Body.String())
	}
	if repository.createInput.Kind != KindCreditNote || repository.createInput.Reason != "Pricing concession" {
		t.Fatalf("normalized input = %#v", repository.createInput)
	}
	if repository.actor.UserID != testUserID || repository.actor.Email != "controller@ledgerops.demo" {
		t.Fatalf("actor = %#v", repository.actor)
	}
}

func TestDecisionRejectsShortReviewNote(t *testing.T) {
	repository := &stubRepository{}
	handler := NewHandler(NewService(repository), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := authenticatedRequest(http.MethodPatch, map[string]any{"decision": "approved", "note": "no"})
	request.SetPathValue("adjustmentID", testAdjustmentID)
	response := httptest.NewRecorder()

	handler.Decide(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.Code)
	}
	if repository.decisionInput.Decision != "" {
		t.Fatal("repository was called for invalid decision")
	}
}

func authenticatedRequest(method string, body any) *http.Request {
	payload, _ := json.Marshal(body)
	request := httptest.NewRequest(method, "/", bytes.NewReader(payload))
	principal := identity.Principal{OrganizationID: testOrganizationID, UserID: testUserID, Email: "controller@ledgerops.demo", Role: identity.RoleController}
	return request.WithContext(identity.WithPrincipal(request.Context(), principal))
}
