package auth

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type fakeRepository struct {
	user           *User
	session        *StoredSession
	failedAttempts int
	lockedUntil    *time.Time
	resetTokenHash []byte
	resetExpiresAt time.Time
	resetUsed      bool
}

func (r *fakeRepository) FindUserByEmail(context.Context, string) (*User, error) { return r.user, nil }
func (r *fakeRepository) CreateAccount(_ context.Context, account NewAccount) (*User, error) {
	if r.user != nil {
		return nil, ErrEmailInUse
	}
	r.user = &User{
		ID: "82222222-2222-7222-8222-222222222222", OrganizationID: "11111111-1111-4111-8111-111111111111",
		OrganizationName: account.OrganizationName, Email: account.Email, DisplayName: account.DisplayName,
		PasswordHash: account.PasswordHash, Role: identity.RoleAdmin, Active: true,
	}
	return r.user, nil
}
func (r *fakeRepository) CreatePasswordReset(_ context.Context, email string, tokenHash []byte, _ time.Time, expiresAt time.Time) (*PasswordResetRecipient, error) {
	if r.user == nil || r.user.Email != email {
		return nil, nil
	}
	r.resetTokenHash = append([]byte(nil), tokenHash...)
	r.resetExpiresAt = expiresAt
	r.resetUsed = false
	return &PasswordResetRecipient{Email: r.user.Email, DisplayName: r.user.DisplayName}, nil
}
func (r *fakeRepository) ConsumePasswordReset(_ context.Context, tokenHash []byte, passwordHash string, now time.Time) error {
	if r.resetUsed || !bytes.Equal(tokenHash, r.resetTokenHash) || !r.resetExpiresAt.After(now) {
		return ErrInvalidResetToken
	}
	r.resetUsed = true
	r.user.PasswordHash = passwordHash
	r.session = nil
	return nil
}
func (r *fakeRepository) RecordFailedLogin(_ context.Context, _ string, now time.Time, maxAttempts int, lockout time.Duration) error {
	r.failedAttempts++
	if r.failedAttempts >= maxAttempts {
		lockedUntil := now.Add(lockout)
		r.lockedUntil = &lockedUntil
		r.failedAttempts = 0
	}
	return nil
}
func (r *fakeRepository) ResetFailedLogins(context.Context, string) error { return nil }
func (r *fakeRepository) CreateSession(_ context.Context, _ string, _ string, tokenHash, csrfHash []byte, expiresAt time.Time) error {
	r.session = &StoredSession{
		ID:        1,
		TokenHash: append([]byte(nil), tokenHash...),
		CSRFHash:  append([]byte(nil), csrfHash...),
		ExpiresAt: expiresAt,
		Principal: principalFromUser(*r.user),
	}
	return nil
}
func (r *fakeRepository) FindSession(_ context.Context, tokenHash []byte, _ time.Time, idleCutoff time.Time) (*StoredSession, error) {
	if r.session == nil || !bytes.Equal(tokenHash, r.session.TokenHash) {
		return nil, nil
	}
	if !r.session.LastSeenAt.IsZero() && !r.session.LastSeenAt.After(idleCutoff) {
		return nil, nil
	}
	return r.session, nil
}
func (r *fakeRepository) RotateCSRF(_ context.Context, _ int64, csrfHash []byte, _ time.Time) error {
	r.session.CSRFHash = append([]byte(nil), csrfHash...)
	return nil
}
func (r *fakeRepository) TouchSession(_ context.Context, _ int64, now time.Time) error {
	r.session.LastSeenAt = now
	return nil
}
func (r *fakeRepository) RevokeSession(context.Context, []byte, time.Time) error {
	r.session = nil
	return nil
}

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	matched, err := VerifyPassword(hash, "correct horse battery staple")
	if err != nil || !matched {
		t.Fatalf("matched = %v, error = %v", matched, err)
	}
	matched, err = VerifyPassword(hash, "wrong password")
	if err != nil || matched {
		t.Fatalf("wrong password matched = %v, error = %v", matched, err)
	}
}

func TestRegisterCreatesAdminAndSession(t *testing.T) {
	repository := &fakeRepository{}
	service, err := NewService(repository, 12*time.Hour, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	session, token, err := service.Register(context.Background(), RegisterRequest{
		OrganizationName: " Northstar Studio ", DisplayName: " Mara Chen ",
		Email: "MARA@example.com", Password: "correct horse battery staple",
	})
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || session.User.Role != identity.RoleAdmin || repository.user.Email != "mara@example.com" {
		t.Fatalf("unexpected registration result: session=%+v user=%+v", session, repository.user)
	}
}

func TestPasswordResetIsSingleUseAndRevokesSession(t *testing.T) {
	passwordHash, err := HashPassword("old password value")
	if err != nil {
		t.Fatal(err)
	}
	repository := &fakeRepository{user: &User{
		ID: "82222222-2222-7222-8222-222222222222", Email: "mara@example.com", DisplayName: "Mara Chen",
		PasswordHash: passwordHash, Active: true,
	}}
	service, err := NewService(repository, 12*time.Hour, 30*time.Minute, WithPasswordResetURL("http://localhost:5173", true))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	response, err := service.RequestPasswordReset(context.Background(), PasswordResetRequest{Email: "mara@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if response.ResetURL == "" {
		t.Fatal("development reset URL was not returned")
	}
	resetURL, err := url.Parse(response.ResetURL)
	if err != nil {
		t.Fatal(err)
	}
	request := PasswordResetConfirmRequest{Token: resetURL.Query().Get("token"), NewPassword: "new correct horse battery staple"}
	if err := service.ResetPassword(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if err := service.ResetPassword(context.Background(), request); !errors.Is(err, ErrInvalidResetToken) {
		t.Fatalf("second reset error = %v, want ErrInvalidResetToken", err)
	}
	matched, err := VerifyPassword(repository.user.PasswordHash, request.NewPassword)
	if err != nil || !matched {
		t.Fatalf("new password matched = %v, error = %v", matched, err)
	}
}

func TestLoginCreatesHashedSessionAndValidatesCSRF(t *testing.T) {
	passwordHash, err := HashPassword("LedgerOpsDemo!2026")
	if err != nil {
		t.Fatal(err)
	}
	repository := &fakeRepository{user: &User{
		ID:               "82222222-2222-7222-8222-222222222222",
		OrganizationID:   "11111111-1111-4111-8111-111111111111",
		OrganizationName: "Northstar Creative Studio",
		Email:            "mara@ledgerops.demo",
		DisplayName:      "Mara Chen",
		PasswordHash:     passwordHash,
		Role:             identity.RoleController,
		Active:           true,
	}}
	service, err := NewService(repository, 12*time.Hour, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC) }

	session, token, err := service.Login(context.Background(), LoginRequest{Email: "MARA@ledgerops.demo", Password: "LedgerOpsDemo!2026"})
	if err != nil {
		t.Fatal(err)
	}
	if repository.session == nil || bytes.Equal(repository.session.TokenHash, []byte(token)) {
		t.Fatal("repository must store a hash, not the raw session token")
	}
	principal, err := service.Authenticate(context.Background(), token, session.CSRFToken, true)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Role != identity.RoleController {
		t.Fatalf("role = %q", principal.Role)
	}
	if _, err := service.Authenticate(context.Background(), token, "invalid", true); !errors.Is(err, ErrInvalidCSRF) {
		t.Fatalf("error = %v, want ErrInvalidCSRF", err)
	}
}

func TestLoginRecordsFailedAttempt(t *testing.T) {
	passwordHash, err := HashPassword("LedgerOpsDemo!2026")
	if err != nil {
		t.Fatal(err)
	}
	repository := &fakeRepository{user: &User{
		ID:             "82222222-2222-7222-8222-222222222222",
		OrganizationID: "11111111-1111-4111-8111-111111111111",
		PasswordHash:   passwordHash,
		Active:         true,
	}}
	service, err := NewService(repository, 12*time.Hour, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = service.Login(context.Background(), LoginRequest{Email: "mara@ledgerops.demo", Password: "incorrect-password"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
	if repository.failedAttempts != 1 {
		t.Fatalf("failed attempts = %d, want 1", repository.failedAttempts)
	}
}

func TestAuthenticateRejectsIdleSession(t *testing.T) {
	passwordHash, err := HashPassword("LedgerOpsDemo!2026")
	if err != nil {
		t.Fatal(err)
	}
	repository := &fakeRepository{user: &User{
		ID: "82222222-2222-7222-8222-222222222222", OrganizationID: "11111111-1111-4111-8111-111111111111",
		Email: "mara@ledgerops.demo", PasswordHash: passwordHash, Active: true,
	}}
	service, err := NewService(repository, 12*time.Hour, 30*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.November, 18, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	session, token, err := service.Login(context.Background(), LoginRequest{Email: repository.user.Email, Password: "LedgerOpsDemo!2026"})
	if err != nil {
		t.Fatal(err)
	}
	repository.session.LastSeenAt = now.Add(-31 * time.Minute)
	if _, err := service.Authenticate(context.Background(), token, session.CSRFToken, true); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("error = %v, want ErrUnauthenticated", err)
	}
}
