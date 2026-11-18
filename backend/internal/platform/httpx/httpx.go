package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/apperrors"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-57][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

func AuthenticatedPrincipal(w http.ResponseWriter, r *http.Request) (identity.Principal, bool) {
	principal, ok := identity.FromContext(r.Context())
	if !ok || !ValidPrincipal(principal) {
		WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
		return identity.Principal{}, false
	}
	return principal, true
}

func OrganizationID(w http.ResponseWriter, r *http.Request) (string, bool) {
	principal, ok := AuthenticatedPrincipal(w, r)
	if !ok {
		return "", false
	}
	return principal.OrganizationID, true
}

func ValidUUID(value string) bool {
	return uuidPattern.MatchString(value)
}

func ValidPrincipal(principal identity.Principal) bool {
	return ValidUUID(principal.OrganizationID) && ValidUUID(principal.UserID)
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, destination any, maxBytes int64) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON value")
		}
		return err
	}
	return nil
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func HandleApplicationError(w http.ResponseWriter, logger *slog.Logger, operation string, err error) {
	if errors.Is(err, apperrors.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "not_found", "The requested record was not found.")
		return
	}
	if errors.Is(err, apperrors.ErrConflict) {
		WriteError(w, http.StatusConflict, "conflict", "A record with those unique details already exists.")
		return
	}
	if errors.Is(err, apperrors.ErrInvalidReference) {
		WriteError(w, http.StatusUnprocessableEntity, "invalid_client", "Select a client from this workspace.")
		return
	}
	if errors.Is(err, apperrors.ErrInvalidState) {
		WriteError(w, http.StatusUnprocessableEntity, "invalid_state", "The action conflicts with the invoice balance or status.")
		return
	}
	logger.Error(operation, "error", err)
	WriteError(w, http.StatusInternalServerError, "internal_error", "The request could not be completed.")
}

type HealthCheck func(context.Context) error

func HealthHandler(check HealthCheck, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := check(ctx); err != nil {
			logger.Error("health check failed", "error", err)
			WriteError(w, http.StatusServiceUnavailable, "database_unavailable", "The database is not available.")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func Wrap(next http.Handler, logger *slog.Logger, allowedOrigin string) http.Handler {
	return requestLog(recoverPanic(securityHeaders(cors(next, allowedOrigin)), logger), logger)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func requestLog(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := randomID()
		w.Header().Set("X-Request-ID", requestID)
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		logger.Info("request complete", "request_id", requestID, "method", r.Method, "path", r.URL.Path, "status", recorder.status, "duration_ms", time.Since(started).Milliseconds())
	})
}

func cors(next http.Handler, allowedOrigin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		w.Header().Add("Vary", "Origin")
		if origin != "" && origin != allowedOrigin {
			WriteError(w, http.StatusForbidden, "origin_not_allowed", "The request origin is not allowed.")
			return
		}
		if origin == allowedOrigin {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=(), payment=(), usb=()")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-site")
		if r.URL.Path == "/api" || len(r.URL.Path) > 5 && r.URL.Path[:5] == "/api/" {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
		}
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func recoverPanic(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "panic", recovered)
				WriteError(w, http.StatusInternalServerError, "internal_error", "The request could not be completed.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func randomID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(bytes)
}
