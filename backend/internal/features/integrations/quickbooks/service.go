package quickbooks

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"time"
)

var (
	ErrNotConfigured   = errors.New("QuickBooks integration is not configured")
	ErrInvalidCallback = errors.New("QuickBooks callback is invalid")
	realmIDPattern     = regexp.MustCompile(`^[0-9]{4,32}$`)
)

type Cipher interface {
	Encrypt(string) ([]byte, error)
	Decrypt([]byte) (string, error)
}

type Service struct {
	repository Repository
	client     Client
	cipher     Cipher
	configured bool
	now        func() time.Time
}

func NewService(repository Repository, client Client, cipher Cipher, configured bool) *Service {
	return &Service{
		repository: repository,
		client:     client,
		cipher:     cipher,
		configured: configured,
		now:        time.Now,
	}
}

func (s *Service) Status(ctx context.Context, organizationID string) (ConnectionStatus, error) {
	status := ConnectionStatus{Configured: s.configured, Environment: "sandbox"}
	connection, err := s.repository.FindConnection(ctx, organizationID)
	if err != nil {
		return ConnectionStatus{}, err
	}
	if connection == nil {
		return status, nil
	}
	status.Connected = true
	status.CompanyName = connection.CompanyName
	status.RealmID = connection.RealmID
	status.ConnectedAt = &connection.ConnectedAt
	return status, nil
}

func (s *Service) BeginAuthorization(ctx context.Context, organizationID string) (Authorization, error) {
	if !s.configured || s.client == nil || s.cipher == nil {
		return Authorization{}, ErrNotConfigured
	}
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return Authorization{}, fmt.Errorf("generate QuickBooks OAuth state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(state))
	if err := s.repository.SaveState(ctx, organizationID, hash[:], s.now().Add(10*time.Minute)); err != nil {
		return Authorization{}, err
	}
	return Authorization{AuthorizationURL: s.client.AuthorizationURL(state)}, nil
}

func (s *Service) CompleteAuthorization(ctx context.Context, state, code, realmID string) error {
	if !s.configured || s.client == nil || s.cipher == nil {
		return ErrNotConfigured
	}
	if state == "" || code == "" || !realmIDPattern.MatchString(realmID) {
		return ErrInvalidCallback
	}
	hash := sha256.Sum256([]byte(state))
	organizationID, err := s.repository.ConsumeState(ctx, hash[:], s.now())
	if err != nil {
		return err
	}
	tokens, err := s.client.ExchangeCode(ctx, code)
	if err != nil {
		return err
	}
	companyName, err := s.client.CompanyName(ctx, tokens.AccessToken, realmID)
	if err != nil {
		return err
	}
	accessToken, err := s.cipher.Encrypt(tokens.AccessToken)
	if err != nil {
		return fmt.Errorf("encrypt QuickBooks access token: %w", err)
	}
	refreshToken, err := s.cipher.Encrypt(tokens.RefreshToken)
	if err != nil {
		return fmt.Errorf("encrypt QuickBooks refresh token: %w", err)
	}
	now := s.now()
	return s.repository.UpsertConnection(ctx, Connection{
		OrganizationID:        organizationID,
		RealmID:               realmID,
		CompanyName:           companyName,
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  now.Add(tokens.AccessTokenExpiresIn),
		RefreshTokenExpiresAt: now.Add(tokens.RefreshTokenExpiresIn),
		Scope:                 tokens.Scope,
		ConnectedAt:           now,
	})
}

func (s *Service) Disconnect(ctx context.Context, organizationID string) error {
	return s.repository.DeleteConnection(ctx, organizationID)
}
