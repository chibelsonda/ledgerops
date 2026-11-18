package quickbooks

import "time"

type Connection struct {
	OrganizationID        string
	RealmID               string
	CompanyName           string
	AccessToken           []byte
	RefreshToken          []byte
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	Scope                 string
	ConnectedAt           time.Time
}

type ConnectionStatus struct {
	Configured  bool       `json:"configured"`
	Connected   bool       `json:"connected"`
	Environment string     `json:"environment"`
	CompanyName string     `json:"companyName,omitempty"`
	RealmID     string     `json:"realmId,omitempty"`
	ConnectedAt *time.Time `json:"connectedAt,omitempty"`
}

type Authorization struct {
	AuthorizationURL string `json:"authorizationUrl"`
}

type Tokens struct {
	AccessToken           string
	RefreshToken          string
	Scope                 string
	AccessTokenExpiresIn  time.Duration
	RefreshTokenExpiresIn time.Duration
}
