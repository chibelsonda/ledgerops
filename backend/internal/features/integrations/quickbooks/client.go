package quickbooks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const accountingScope = "com.intuit.quickbooks.accounting"

type Client interface {
	AuthorizationURL(state string) string
	ExchangeCode(context.Context, string) (Tokens, error)
	CompanyName(context.Context, string, string) (string, error)
}

type IntuitClient struct {
	clientID     string
	clientSecret string
	redirectURL  string
	apiBaseURL   string
	httpClient   *http.Client
}

func NewIntuitClient(clientID, clientSecret, redirectURL, apiBaseURL string) *IntuitClient {
	return &IntuitClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		apiBaseURL:   strings.TrimRight(apiBaseURL, "/"),
		httpClient:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *IntuitClient) AuthorizationURL(state string) string {
	query := url.Values{
		"client_id":     {c.clientID},
		"scope":         {accountingScope},
		"redirect_uri":  {c.redirectURL},
		"response_type": {"code"},
		"state":         {state},
	}
	return "https://appcenter.intuit.com/connect/oauth2?" + query.Encode()
}

func (c *IntuitClient) ExchangeCode(ctx context.Context, code string) (Tokens, error) {
	form := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {c.redirectURL},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer", strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, fmt.Errorf("create token exchange request: %w", err)
	}
	request.SetBasicAuth(c.clientID, c.clientSecret)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("x-include-refresh-token-hard-expires-in", "true")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Tokens{}, fmt.Errorf("exchange QuickBooks authorization code: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return Tokens{}, fmt.Errorf("read QuickBooks token response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return Tokens{}, fmt.Errorf("QuickBooks token exchange returned status %d", response.StatusCode)
	}
	var payload struct {
		AccessToken               string `json:"access_token"`
		RefreshToken              string `json:"refresh_token"`
		Scope                     string `json:"scope"`
		ExpiresIn                 int64  `json:"expires_in"`
		RefreshTokenExpiresIn     int64  `json:"x_refresh_token_expires_in"`
		RefreshTokenHardExpiresIn int64  `json:"x_refresh_token_hard_expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Tokens{}, fmt.Errorf("decode QuickBooks token response: %w", err)
	}
	refreshExpiry := payload.RefreshTokenExpiresIn
	if refreshExpiry == 0 {
		refreshExpiry = payload.RefreshTokenHardExpiresIn
	}
	if payload.AccessToken == "" || payload.RefreshToken == "" || payload.ExpiresIn <= 0 || refreshExpiry <= 0 {
		return Tokens{}, fmt.Errorf("QuickBooks token response is incomplete")
	}
	return Tokens{
		AccessToken:           payload.AccessToken,
		RefreshToken:          payload.RefreshToken,
		Scope:                 payload.Scope,
		AccessTokenExpiresIn:  time.Duration(payload.ExpiresIn) * time.Second,
		RefreshTokenExpiresIn: time.Duration(refreshExpiry) * time.Second,
	}, nil
}

func (c *IntuitClient) CompanyName(ctx context.Context, accessToken, realmID string) (string, error) {
	endpoint := fmt.Sprintf("%s/v3/company/%s/companyinfo/%s", c.apiBaseURL, url.PathEscape(realmID), url.PathEscape(realmID))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create company request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("fetch QuickBooks company: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return "", fmt.Errorf("QuickBooks company request returned status %d", response.StatusCode)
	}
	var payload struct {
		CompanyInfo struct {
			CompanyName string `json:"CompanyName"`
		} `json:"CompanyInfo"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode QuickBooks company: %w", err)
	}
	if payload.CompanyInfo.CompanyName == "" {
		return "", fmt.Errorf("QuickBooks company response is incomplete")
	}
	return payload.CompanyInfo.CompanyName, nil
}
