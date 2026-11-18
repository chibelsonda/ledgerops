package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/portfolio/ledgerops/backend/internal/platform/identity"
)

type fakeRepository struct {
	user           *User
	session        *StoredSession
	failedAttempts int
	lockedUntil    *time.Time
}

func (r *fakeRepository) FindUserByEmail(context.Context, string) (*User, error) { return r.user, nil }
func (r *fakeRepository) RecordFailedLogin(_ context.Context, _ string, attempts int, lockedUntil *time.Time) error {
	r.failedAttempts = attempts
	r.lockedUntil = lockedUntil
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
func (r *fakeRepository) FindSession(_ context.Context, tokenHash []byte, _ time.Time) (*StoredSession, error) {
	if r.session == nil || !bytes.Equal(tokenHash, r.session.TokenHash) {
		return nil, nil
	}
	return r.session, nil
}
func (r *fakeRepository) RotateCSRF(_ context.Context, _ int64, csrfHash []byte, _ time.Time) error {
	r.session.CSRFHash = append([]byte(nil), csrfHash...)
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
	service, err := NewService(repository, 12*time.Hour)
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
	service, err := NewService(repository, 12*time.Hour)
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
