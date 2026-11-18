package httpx

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
