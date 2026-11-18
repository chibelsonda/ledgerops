package httpx

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterRejectsRequestsOverLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	limiter := NewRateLimiter(2, time.Minute, "test", logger)
	now := time.Date(2026, time.November, 18, 0, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), func(*http.Request) string { return "client" })

	for index, expected := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/test", nil))
		if response.Code != expected {
			t.Fatalf("request %d status = %d, want %d", index+1, response.Code, expected)
		}
	}
}

func TestClientIPResolverOnlyTrustsConfiguredProxy(t *testing.T) {
	resolver, err := NewClientIPResolver([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "10.0.0.4:1234"
	request.Header.Set("X-Forwarded-For", "203.0.113.7")
	if got := resolver.Key(request); got != "203.0.113.7" {
		t.Fatalf("trusted proxy client = %q", got)
	}

	request.RemoteAddr = "198.51.100.10:1234"
	request.Header.Set("X-Forwarded-For", "203.0.113.8")
	if got := resolver.Key(request); got != "198.51.100.10" {
		t.Fatalf("untrusted proxy client = %q", got)
	}
}
