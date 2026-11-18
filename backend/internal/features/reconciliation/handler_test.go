package reconciliation

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type stubRepository struct{}

func (stubRepository) List(context.Context, string) ([]Item, error)                  { return nil, nil }
func (stubRepository) Resolve(context.Context, string, string, string, string) error { return nil }

func TestResolveRejectsShortNote(t *testing.T) {
	handler := NewHandler(NewService(stubRepository{}), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/reconciliation/71111111-1111-4111-8111-111111111111", strings.NewReader(`{"note":"x"}`))
	request.SetPathValue("itemID", "71111111-1111-4111-8111-111111111111")
	request = request.WithContext(identity.WithPrincipal(request.Context(), identity.Principal{OrganizationID: "11111111-1111-4111-8111-111111111111", UserID: "81111111-1111-7111-8111-111111111111", Email: "reviewer@ledgerops.demo"}))
	response := httptest.NewRecorder()
	handler.Resolve(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", response.Code)
	}
}
