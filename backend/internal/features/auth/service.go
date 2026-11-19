package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidLogin        = errors.New("invalid login request")
	ErrUnauthenticated     = errors.New("authentication required")
	ErrInvalidCSRF         = errors.New("invalid CSRF token")
	ErrInvalidRegistration = errors.New("invalid registration request")
	ErrEmailInUse          = errors.New("email address is already registered")
	ErrInvalidResetRequest = errors.New("invalid password reset request")
	ErrInvalidResetToken   = errors.New("invalid or expired password reset token")
)

const (
	sessionTokenBytes = 32
	csrfTokenBytes    = 32
	maxLoginAttempts  = 5
	lockoutDuration   = 15 * time.Minute
	passwordResetTTL  = 30 * time.Minute
)

type PasswordResetNotifier interface {
	SendPasswordReset(context.Context, string, string, string) error
}

type ServiceOption func(*Service)

func WithPasswordResetNotifier(notifier PasswordResetNotifier) ServiceOption {
	return func(service *Service) { service.passwordResetNotifier = notifier }
}

func WithPasswordResetURL(frontendURL string, exposePreview bool) ServiceOption {
	return func(service *Service) {
		service.frontendURL = strings.TrimRight(frontendURL, "/")
		service.exposeResetPreview = exposePreview
	}
}

type Service struct {
	repository            Repository
	sessionTTL            time.Duration
	sessionIdleTTL        time.Duration
	dummyHash             string
	now                   func() time.Time
	passwordResetNotifier PasswordResetNotifier
	frontendURL           string
	exposeResetPreview    bool
}

func NewService(repository Repository, sessionTTL, sessionIdleTTL time.Duration, options ...ServiceOption) (*Service, error) {
	dummyHash, err := HashPassword("timing-only-password-value")
	if err != nil {
		return nil, err
	}
	service := &Service{repository: repository, sessionTTL: sessionTTL, sessionIdleTTL: sessionIdleTTL, dummyHash: dummyHash, now: time.Now}
	for _, option := range options {
		option(service)
	}
	return service, nil
}

func (s *Service) Register(ctx context.Context, request RegisterRequest) (SessionResponse, string, error) {
	account := NewAccount{
		OrganizationName: strings.TrimSpace(request.OrganizationName),
		DisplayName:      strings.TrimSpace(request.DisplayName),
		Email:            strings.ToLower(strings.TrimSpace(request.Email)),
	}
	if len(account.OrganizationName) < 2 || len(account.OrganizationName) > 120 ||
		len(account.DisplayName) < 2 || len(account.DisplayName) > 120 ||
		!validEmail(account.Email) || !validPassword(request.Password) {
		return SessionResponse{}, "", ErrInvalidRegistration
	}
	passwordHash, err := HashPassword(request.Password)
	if err != nil {
		return SessionResponse{}, "", fmt.Errorf("hash registration password: %w", err)
	}
	account.PasswordHash = passwordHash
	user, err := s.repository.CreateAccount(ctx, account)
	if err != nil {
		return SessionResponse{}, "", err
	}
	return s.createSession(ctx, *user)
}

func (s *Service) RequestPasswordReset(ctx context.Context, request PasswordResetRequest) (PasswordResetResponse, error) {
	email := strings.ToLower(strings.TrimSpace(request.Email))
	if !validEmail(email) {
		return PasswordResetResponse{}, ErrInvalidResetRequest
	}
	token, tokenHash, err := randomToken(sessionTokenBytes)
	if err != nil {
		return PasswordResetResponse{}, err
	}
	now := s.now()
	recipient, err := s.repository.CreatePasswordReset(ctx, email, tokenHash, now, now.Add(passwordResetTTL))
	if err != nil {
		return PasswordResetResponse{}, err
	}
	response := PasswordResetResponse{Message: "If an account exists for that email, a password reset link has been sent."}
	resetURL, err := s.passwordResetURL(token)
	if err != nil {
		return PasswordResetResponse{}, err
	}
	if s.exposeResetPreview {
		response.ResetURL = resetURL
	}
	if recipient == nil {
		return response, nil
	}
	if s.passwordResetNotifier != nil {
		if err := s.passwordResetNotifier.SendPasswordReset(ctx, recipient.Email, recipient.DisplayName, resetURL); err != nil {
			return response, fmt.Errorf("send password reset: %w", err)
		}
	}
	return response, nil
}

func (s *Service) ResetPassword(ctx context.Context, request PasswordResetConfirmRequest) error {
	if !validPassword(request.NewPassword) {
		return ErrInvalidResetRequest
	}
	decoded, err := base64.RawURLEncoding.DecodeString(request.Token)
	if err != nil || len(decoded) != sessionTokenBytes {
		return ErrInvalidResetToken
	}
	tokenHash := sha256.Sum256(decoded)
	passwordHash, err := HashPassword(request.NewPassword)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	return s.repository.ConsumePasswordReset(ctx, tokenHash[:], passwordHash, s.now())
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
			if err := s.repository.RecordFailedLogin(ctx, user.ID, now, maxLoginAttempts, lockoutDuration); err != nil {
				return SessionResponse{}, "", err
			}
		}
		return SessionResponse{}, "", ErrInvalidCredentials
	}
	if err := s.repository.ResetFailedLogins(ctx, user.ID); err != nil {
		return SessionResponse{}, "", err
	}
	return s.createSession(ctx, *user)
}

func (s *Service) createSession(ctx context.Context, user User) (SessionResponse, string, error) {
	sessionToken, sessionHash, err := randomToken(sessionTokenBytes)
	if err != nil {
		return SessionResponse{}, "", err
	}
	csrfToken, csrfHash, err := randomToken(csrfTokenBytes)
	if err != nil {
		return SessionResponse{}, "", err
	}
	expiresAt := s.now().Add(s.sessionTTL)
	if err := s.repository.CreateSession(ctx, user.ID, user.OrganizationID, sessionHash, csrfHash, expiresAt); err != nil {
		return SessionResponse{}, "", err
	}
	return SessionResponse{
		User:      principalFromUser(user),
		CSRFToken: csrfToken,
		ExpiresAt: expiresAt,
	}, sessionToken, nil
}

func (s *Service) passwordResetURL(token string) (string, error) {
	if s.frontendURL == "" {
		return "", errors.New("frontend URL is not configured")
	}
	resetURL, err := url.Parse(s.frontendURL + "/reset-password")
	if err != nil {
		return "", fmt.Errorf("build password reset URL: %w", err)
	}
	query := resetURL.Query()
	query.Set("token", token)
	resetURL.RawQuery = query.Encode()
	return resetURL.String(), nil
}

func validPassword(value string) bool {
	return len(value) >= 12 && len(value) <= 128
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
	if err := s.repository.TouchSession(ctx, session.ID, s.now()); err != nil {
		return identity.Principal{}, err
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
	now := s.now()
	session, err := s.repository.FindSession(ctx, hash[:], now, now.Add(-s.sessionIdleTTL))
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
	if len(value) < 3 || len(value) > 254 || !isASCII(value) {
		return false
	}
	parts := strings.Split(value, "@")
	if len(parts) != 2 || len(parts[0]) > 64 || parts[0] == "" || parts[1] == "" {
		return false
	}
	local := parts[0]
	if local[0] == '.' || local[len(local)-1] == '.' || strings.Contains(local, "..") {
		return false
	}
	for _, character := range local {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.!#$%&'*+-/=?^_`{|}~", character) {
			return false
		}
	}
	labels := strings.Split(parts[1], ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, character := range label {
			if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '-') {
				return false
			}
		}
	}
	return true
}

func isASCII(value string) bool {
	for index := range len(value) {
		if value[index] > 127 {
			return false
		}
	}
	return true
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
