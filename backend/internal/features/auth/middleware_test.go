package auth

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type staticRequestAuthenticator struct {
	principal identity.Principal
}

func (a staticRequestAuthenticator) Authenticate(context.Context, string, string, bool) (identity.Principal, error) {
	return a.principal, nil
}

func TestAuthenticateRejectsInvalidTenantIdentityBeforeProtectedHandler(t *testing.T) {
	called := false
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	middleware := NewMiddleware(staticRequestAuthenticator{principal: identity.Principal{
		OrganizationID: "not-a-uuid",
		UserID:         "81111111-1111-7111-8111-111111111111",
		Role:           identity.RoleController,
	}}, logger)
	handler := middleware.Authenticate(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
	request.AddCookie(&http.Cookie{Name: CookieName, Value: "session-token"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	if called {
		t.Fatal("protected handler was called with an invalid tenant identity")
	}
}

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
