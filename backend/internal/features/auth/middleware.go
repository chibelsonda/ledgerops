package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/portfolio/ledgerops/backend/internal/platform/httpx"
	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type RequestAuthenticator interface {
	Authenticate(context.Context, string, string, bool) (identity.Principal, error)
}

type Middleware struct {
	service RequestAuthenticator
	logger  *slog.Logger
}

func NewMiddleware(service RequestAuthenticator, logger *slog.Logger) *Middleware {
	return &Middleware{service: service, logger: logger}
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
			return
		}
		requireCSRF := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
		principal, err := m.service.Authenticate(r.Context(), cookie.Value, r.Header.Get("X-CSRF-Token"), requireCSRF)
		if err != nil {
			if errors.Is(err, ErrInvalidCSRF) {
				m.logger.Warn("security CSRF validation failed", "remote_addr", r.RemoteAddr, "method", r.Method, "path", r.URL.Path)
				httpx.WriteError(w, http.StatusForbidden, "invalid_csrf_token", "The request could not be verified. Refresh and try again.")
				return
			}
			if !errors.Is(err, ErrUnauthenticated) {
				m.logger.Error("authenticate request", "error", err)
			}
			m.logger.Warn("security authentication failed", "remote_addr", r.RemoteAddr, "method", r.Method, "path", r.URL.Path)
			httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
			return
		}
		if !httpx.ValidPrincipal(principal) {
			m.logger.Error("authenticated session has invalid tenant identity", "user_id", principal.UserID, "organization_id", principal.OrganizationID)
			httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
			return
		}
		next.ServeHTTP(w, r.WithContext(identity.WithPrincipal(r.Context(), principal)))
	})
}

func RequireRoles(logger *slog.Logger, roles ...identity.Role) func(http.Handler) http.Handler {
	allowed := make(map[identity.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := identity.FromContext(r.Context())
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, "authentication_required", "Sign in to continue.")
				return
			}
			if _, ok := allowed[principal.Role]; !ok {
				logger.Warn("security authorization denied", "user_id", principal.UserID, "organization_id", principal.OrganizationID, "role", principal.Role, "method", r.Method, "path", r.URL.Path)
				httpx.WriteError(w, http.StatusForbidden, "forbidden", "You do not have permission to perform this action.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
