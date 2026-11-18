package quickbooks

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeRepository struct {
	connection     *Connection
	organizationID string
	stateHash      []byte
	expiresAt      time.Time
}

func (r *fakeRepository) FindConnection(context.Context, string) (*Connection, error) {
	return r.connection, nil
}

func (r *fakeRepository) SaveState(_ context.Context, organizationID string, stateHash []byte, expiresAt time.Time) error {
	r.organizationID = organizationID
	r.stateHash = append([]byte(nil), stateHash...)
	r.expiresAt = expiresAt
	return nil
}

func (r *fakeRepository) ConsumeState(_ context.Context, stateHash []byte, now time.Time) (string, error) {
	if !bytes.Equal(stateHash, r.stateHash) || !now.Before(r.expiresAt) {
		return "", ErrInvalidState
	}
	return r.organizationID, nil
}

func (r *fakeRepository) UpsertConnection(_ context.Context, connection Connection) error {
	r.connection = &connection
	return nil
}

func (r *fakeRepository) DeleteConnection(context.Context, string) error {
	r.connection = nil
	return nil
}

type fakeClient struct{}

func (fakeClient) AuthorizationURL(state string) string {
	return "https://example.test/authorize?state=" + state
}
func (fakeClient) ExchangeCode(context.Context, string) (Tokens, error) {
	return Tokens{
		AccessToken:           "access",
		RefreshToken:          "refresh",
		Scope:                 accountingScope,
		AccessTokenExpiresIn:  time.Hour,
		RefreshTokenExpiresIn: 100 * 24 * time.Hour,
	}, nil
}
func (fakeClient) CompanyName(context.Context, string, string) (string, error) {
	return "Sandbox Co", nil
}

type prefixCipher struct{}

func (prefixCipher) Encrypt(value string) ([]byte, error) { return []byte("encrypted:" + value), nil }
func (prefixCipher) Decrypt(value []byte) (string, error) {
	return strings.TrimPrefix(string(value), "encrypted:"), nil
}

func TestAuthorizationFlowStoresEncryptedConnection(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository, fakeClient{}, prefixCipher{}, true)
	now := time.Date(2026, time.November, 18, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	authorization, err := service.BeginAuthorization(context.Background(), "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	state := strings.TrimPrefix(authorization.AuthorizationURL, "https://example.test/authorize?state=")
	hash := sha256.Sum256([]byte(state))
	if base64.StdEncoding.EncodeToString(repository.stateHash) != base64.StdEncoding.EncodeToString(hash[:]) {
		t.Fatal("repository did not receive the hashed OAuth state")
	}
	if err := service.CompleteAuthorization(context.Background(), state, "authorization-code", "1234567890"); err != nil {
		t.Fatal(err)
	}
	if repository.connection == nil || string(repository.connection.AccessToken) != "encrypted:access" {
		t.Fatal("encrypted QuickBooks connection was not saved")
	}
	if repository.connection.CompanyName != "Sandbox Co" {
		t.Fatalf("company name = %q", repository.connection.CompanyName)
	}
}

func TestBeginAuthorizationRequiresConfiguration(t *testing.T) {
	service := NewService(&fakeRepository{}, nil, nil, false)
	_, err := service.BeginAuthorization(context.Background(), "11111111-1111-4111-8111-111111111111")
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("error = %v, want ErrNotConfigured", err)
	}
}

func TestAESGCMCipherRoundTrip(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	cipher, err := NewAESGCMCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := cipher.Encrypt("secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if string(ciphertext) == "secret-token" {
		t.Fatal("token was stored as plaintext")
	}
	plaintext, err := cipher.Decrypt(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if plaintext != "secret-token" {
		t.Fatalf("plaintext = %q", plaintext)
	}
}
