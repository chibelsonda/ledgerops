package httpx

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

func TestDecodeJSONRejectsTrailingJSONValue(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/things", bytes.NewBufferString(`{"name":"first"}{"name":"second"}`))
	response := httptest.NewRecorder()
	var input struct{ Name string `json:"name"` }

	if err := DecodeJSON(response, request, &input, 1024); err == nil {
		t.Fatal("DecodeJSON accepted multiple JSON values")
	}
}

func TestDecodeJSONAcceptsOneValueAndWhitespace(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/things", bytes.NewBufferString("{\"name\":\"only\"} \n\t"))
	response := httptest.NewRecorder()
	var input struct{ Name string `json:"name"` }

	if err := DecodeJSON(response, request, &input, 1024); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if input.Name != "only" {
		t.Fatalf("name = %q, want only", input.Name)
	}
}

const (
	testOrganizationID = "11111111-1111-4111-8111-111111111111"
	testUserID         = "81111111-1111-7111-8111-111111111111"
)

func TestAuthenticatedPrincipalReturnsTenantFromContext(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
	want := identity.Principal{OrganizationID: testOrganizationID, UserID: testUserID, Email: "controller@ledgerops.demo"}
	request = request.WithContext(identity.WithPrincipal(request.Context(), want))
	response := httptest.NewRecorder()

	got, ok := AuthenticatedPrincipal(response, request)
	if !ok {
		t.Fatalf("principal was rejected; status = %d", response.Code)
	}
	if got != want {
		t.Fatalf("principal = %#v, want %#v", got, want)
	}
}

func TestAuthenticatedPrincipalRejectsIncompleteTenantIdentity(t *testing.T) {
	tests := map[string]identity.Principal{
		"missing principal":    {},
		"invalid organization": {OrganizationID: "not-a-uuid", UserID: testUserID},
		"invalid user":         {OrganizationID: testOrganizationID, UserID: "not-a-uuid"},
	}
	for name, principal := range tests {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/invoices", nil)
			if name != "missing principal" {
				request = request.WithContext(identity.WithPrincipal(request.Context(), principal))
			}
			response := httptest.NewRecorder()

			if _, ok := AuthenticatedPrincipal(response, request); ok {
				t.Fatal("invalid principal was accepted")
			}
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestWrapAddsSecurityAndNoStoreHeaders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}), logger, "https://ledgerops.example")
	request := httptest.NewRequest(http.MethodGet, "https://api.ledgerops.example/api/v1/session", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	for header, expected := range map[string]string{
		"Cache-Control": "no-store", "Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'",
		"Permissions-Policy": "camera=(), geolocation=(), microphone=(), payment=(), usb=()", "Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY",
	} {
		if got := response.Header().Get(header); got != expected {
			t.Errorf("%s = %q, want %q", header, got, expected)
		}
	}
}

func TestWrapRejectsUnexpectedBrowserOrigin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), logger, "https://ledgerops.example")
	request := httptest.NewRequest(http.MethodPost, "https://api.ledgerops.example/api/v1/invoices", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", response.Code)
	}
}
