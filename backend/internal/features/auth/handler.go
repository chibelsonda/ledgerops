package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
)

type SessionService interface {
	Login(context.Context, LoginRequest) (SessionResponse, string, error)
	RefreshSession(context.Context, string) (SessionResponse, error)
	Logout(context.Context, string) error
}

type Handler struct {
	service      SessionService
	logger       *slog.Logger
	cookieSecure bool
	sessionTTL   time.Duration
}

func NewHandler(service SessionService, logger *slog.Logger, cookieSecure bool, sessionTTL time.Duration) *Handler {
	return &Handler{service: service, logger: logger, cookieSecure: cookieSecure, sessionTTL: sessionTTL}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var request LoginRequest
	if err := httpx.DecodeJSON(w, r, &request, 4<<10); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Enter a valid email address and password.")
		return
	}
	session, token, err := h.service.Login(r.Context(), request)
	if err != nil {
		h.logger.Warn("security login failed", "remote_addr", r.RemoteAddr, "reason", authErrorCode(err))
		handleAuthError(w, h.logger, "log in", err)
		return
	}
	h.setCookie(w, token, session.ExpiresAt)
	h.logger.Info("security login succeeded", "remote_addr", r.RemoteAddr, "user_id", session.User.UserID, "organization_id", session.User.OrganizationID)
	httpx.WriteJSON(w, http.StatusOK, session)
}

func (h *Handler) Session(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
		return
	}
	session, err := h.service.RefreshSession(r.Context(), cookie.Value)
	if err != nil {
		h.clearCookie(w)
		h.logger.Warn("security session refresh failed", "remote_addr", r.RemoteAddr, "reason", authErrorCode(err))
		handleAuthError(w, h.logger, "refresh session", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(CookieName); err == nil {
		if err := h.service.Logout(r.Context(), cookie.Value); err != nil {
			handleAuthError(w, h.logger, "log out", err)
			return
		}
	}
	h.clearCookie(w)
	w.Header().Set("Clear-Site-Data", `"cache", "cookies", "storage"`)
	h.logger.Info("security logout completed", "remote_addr", r.RemoteAddr)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(h.sessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

func handleAuthError(w http.ResponseWriter, logger *slog.Logger, operation string, err error) {
	switch {
	case errors.Is(err, ErrInvalidLogin):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid_login", "Enter a valid email address and a password between 12 and 128 characters.")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "The email or password is incorrect.")
	case errors.Is(err, ErrUnauthenticated):
		httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
	case errors.Is(err, ErrInvalidCSRF):
		httpx.WriteError(w, http.StatusForbidden, "invalid_csrf_token", "The request could not be verified. Refresh and try again.")
	default:
		logger.Error(operation, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "The request could not be completed.")
	}
}

func authErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidLogin):
		return "invalid_login"
	case errors.Is(err, ErrInvalidCredentials):
		return "invalid_credentials"
	case errors.Is(err, ErrUnauthenticated):
		return "unauthenticated"
	case errors.Is(err, ErrInvalidCSRF):
		return "invalid_csrf"
	default:
		return "internal_error"
	}
}
