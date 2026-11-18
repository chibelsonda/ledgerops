package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidLogin       = errors.New("invalid login request")
	ErrUnauthenticated    = errors.New("authentication required")
	ErrInvalidCSRF        = errors.New("invalid CSRF token")
)

const (
	sessionTokenBytes = 32
	csrfTokenBytes    = 32
	maxLoginAttempts  = 5
	lockoutDuration   = 15 * time.Minute
)

type Service struct {
	repository Repository
	sessionTTL time.Duration
	dummyHash  string
	now        func() time.Time
}

func NewService(repository Repository, sessionTTL time.Duration) (*Service, error) {
	dummyHash, err := HashPassword("timing-only-password-value")
	if err != nil {
		return nil, err
	}
	return &Service{repository: repository, sessionTTL: sessionTTL, dummyHash: dummyHash, now: time.Now}, nil
}

func (s *Service) Login(ctx context.Context, request LoginRequest) (SessionResponse, string, error) {
	email := strings.ToLower(strings.TrimSpace(request.Email))
	if !validEmail(email) || len(request.Password) < 12 || len(request.Password) > 128 {
		return SessionResponse{}, "", ErrInvalidLogin
	}
	user, err := s.repository.FindUserByEmail(ctx, email)
	if err != nil {
		return SessionResponse{}, "", err
	}
	if user == nil {
		_, _ = VerifyPassword(s.dummyHash, request.Password)
		return SessionResponse{}, "", ErrInvalidCredentials
	}
	passwordMatches, err := VerifyPassword(user.PasswordHash, request.Password)
	if err != nil {
		return SessionResponse{}, "", fmt.Errorf("verify password hash: %w", err)
	}
	now := s.now()
	if !user.Active || (user.LockedUntil != nil && user.LockedUntil.After(now)) || !passwordMatches {
		if !passwordMatches && user.Active && (user.LockedUntil == nil || !user.LockedUntil.After(now)) {
			attempts := user.FailedLoginAttempts + 1
			var lockedUntil *time.Time
			if attempts >= maxLoginAttempts {
				locked := now.Add(lockoutDuration)
				lockedUntil = &locked
				attempts = 0
			}
			if err := s.repository.RecordFailedLogin(ctx, user.ID, attempts, lockedUntil); err != nil {
				return SessionResponse{}, "", err
			}
		}
		return SessionResponse{}, "", ErrInvalidCredentials
	}
	if err := s.repository.ResetFailedLogins(ctx, user.ID); err != nil {
		return SessionResponse{}, "", err
	}
	sessionToken, sessionHash, err := randomToken(sessionTokenBytes)
	if err != nil {
		return SessionResponse{}, "", err
	}
	csrfToken, csrfHash, err := randomToken(csrfTokenBytes)
	if err != nil {
		return SessionResponse{}, "", err
	}
	expiresAt := now.Add(s.sessionTTL)
	if err := s.repository.CreateSession(ctx, user.ID, user.OrganizationID, sessionHash, csrfHash, expiresAt); err != nil {
		return SessionResponse{}, "", err
	}
	return SessionResponse{
		User:      principalFromUser(*user),
		CSRFToken: csrfToken,
		ExpiresAt: expiresAt,
	}, sessionToken, nil
}

func (s *Service) RefreshSession(ctx context.Context, sessionToken string) (SessionResponse, error) {
	session, err := s.findSession(ctx, sessionToken)
	if err != nil {
		return SessionResponse{}, err
	}
	csrfToken, csrfHash, err := randomToken(csrfTokenBytes)
	if err != nil {
		return SessionResponse{}, err
	}
	if err := s.repository.RotateCSRF(ctx, session.ID, csrfHash, s.now()); err != nil {
		return SessionResponse{}, err
	}
	return SessionResponse{User: session.Principal, CSRFToken: csrfToken, ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) Authenticate(ctx context.Context, sessionToken, csrfToken string, requireCSRF bool) (identity.Principal, error) {
	session, err := s.findSession(ctx, sessionToken)
	if err != nil {
		return identity.Principal{}, err
	}
	if requireCSRF {
		decoded, err := base64.RawURLEncoding.DecodeString(csrfToken)
		if err != nil || len(decoded) != csrfTokenBytes {
			return identity.Principal{}, ErrInvalidCSRF
		}
		hash := sha256.Sum256(decoded)
		if subtle.ConstantTimeCompare(hash[:], session.CSRFHash) != 1 {
			return identity.Principal{}, ErrInvalidCSRF
		}
	}
	return session.Principal, nil
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	decoded, err := base64.RawURLEncoding.DecodeString(sessionToken)
	if err != nil || len(decoded) != sessionTokenBytes {
		return nil
	}
	hash := sha256.Sum256(decoded)
	return s.repository.RevokeSession(ctx, hash[:], s.now())
}

func (s *Service) findSession(ctx context.Context, token string) (*StoredSession, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(decoded) != sessionTokenBytes {
		return nil, ErrUnauthenticated
	}
	hash := sha256.Sum256(decoded)
	session, err := s.repository.FindSession(ctx, hash[:], s.now())
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrUnauthenticated
	}
	return session, nil
}

func randomToken(size int) (string, []byte, error) {
	raw := make([]byte, size)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate authentication token: %w", err)
	}
	hash := sha256.Sum256(raw)
	return base64.RawURLEncoding.EncodeToString(raw), hash[:], nil
}

func validEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value) && len(value) <= 254
}

func principalFromUser(user User) identity.Principal {
	return identity.Principal{
		UserID:           user.ID,
		OrganizationID:   user.OrganizationID,
		OrganizationName: user.OrganizationName,
		Email:            user.Email,
		DisplayName:      user.DisplayName,
		Role:             user.Role,
	}
}
