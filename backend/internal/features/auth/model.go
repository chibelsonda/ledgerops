package auth

import (
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

const CookieName = "ledgerops_id"

type User struct {
	ID                  string
	OrganizationID      string
	OrganizationName    string
	Email               string
	DisplayName         string
	PasswordHash        string
	Role                identity.Role
	FailedLoginAttempts int
	LockedUntil         *time.Time
	Active              bool
}

type StoredSession struct {
	ID         int64
	TokenHash  []byte
	CSRFHash   []byte
	ExpiresAt  time.Time
	LastSeenAt time.Time
	Principal  identity.Principal
}

type SessionResponse struct {
	User      identity.Principal `json:"user"`
	CSRFToken string             `json:"csrfToken"`
	ExpiresAt time.Time          `json:"expiresAt"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
