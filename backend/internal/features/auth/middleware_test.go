package auth

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

func TestRequireRolesRejectsReviewer(t *testing.T) {
	called := false
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := RequireRoles(logger, identity.RoleAdmin, identity.RoleController)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", nil)
	request = request.WithContext(identity.WithPrincipal(request.Context(), identity.Principal{Role: identity.RoleReviewer}))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
	if called {
		t.Fatal("protected handler was called for reviewer")
	}
}

func TestRequireRolesAllowsController(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := RequireRoles(logger, identity.RoleAdmin, identity.RoleController)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/invoices", nil)
	request = request.WithContext(identity.WithPrincipal(request.Context(), identity.Principal{Role: identity.RoleController}))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.Code)
	}
}
