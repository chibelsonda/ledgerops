package sync

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type disconnectedRepository struct{}

func (disconnectedRepository) ListRuns(context.Context, string) ([]Run, error) { return nil, nil }
func (disconnectedRepository) Enqueue(context.Context, string) (Run, error) {
	return Run{}, ErrProviderNotConnected
}

func TestEnqueueRequiresQuickBooksConnection(t *testing.T) {
	handler := NewHandler(NewService(disconnectedRepository{}), slog.New(slog.NewTextHandler(io.Discard, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/sync-runs", nil)
	request = request.WithContext(identity.WithPrincipal(request.Context(), identity.Principal{OrganizationID: "11111111-1111-4111-8111-111111111111", UserID: "81111111-1111-7111-8111-111111111111"}))
	response := httptest.NewRecorder()

	handler.Enqueue(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}
