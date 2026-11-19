package quickbooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	accountingScope = "com.intuit.quickbooks.accounting"
	defaultTokenURL = "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer"
)

type Client interface {
	AuthorizationURL(state string) string
	ExchangeCode(context.Context, string) (Tokens, error)
	CompanyName(context.Context, string, string) (string, error)
	RevokeToken(context.Context, string) error
}

type AccountingClient interface {
	RefreshTokens(context.Context, string) (Tokens, error)
	RevokeToken(context.Context, string) error
	ListCustomers(context.Context, string, string, int, int) (Page[CustomerRecord], error)
	ListInvoices(context.Context, string, string, int, int) (Page[InvoiceRecord], error)
	ListPayments(context.Context, string, string, int, int) (Page[PaymentRecord], error)
	ListExpenses(context.Context, string, string, int, int) (Page[ExpenseRecord], error)
	ListRemoteEntities(context.Context, string, string, string, int, int) (Page[RemoteEntity], error)
	ChangeDataCapture(context.Context, string, string, []string, time.Time) (CDCResult, error)
}

type APIError struct {
	Operation  string
	StatusCode int
	Code       string
	Message    string
	RequestID  string
	RetryDelay time.Duration
}

func (e *APIError) Error() string {
	message := e.Message
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	if e.RequestID != "" {
		return fmt.Sprintf("QuickBooks %s failed (%d, request %s): %s", e.Operation, e.StatusCode, e.RequestID, message)
	}
	return fmt.Sprintf("QuickBooks %s failed (%d): %s", e.Operation, e.StatusCode, message)
}

func (e *APIError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= http.StatusInternalServerError
}

func (e *APIError) RetryAfter() time.Duration { return e.RetryDelay }

type IntuitClient struct {
	clientID     string
	clientSecret string
	redirectURL  string
	apiBaseURL   string
	tokenURL     string
	revokeURL    string
	httpClient   *http.Client
}

func NewIntuitClient(clientID, clientSecret, redirectURL, apiBaseURL string) *IntuitClient {
	return &IntuitClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		apiBaseURL:   strings.TrimRight(apiBaseURL, "/"),
		tokenURL:     defaultTokenURL,
		revokeURL:    "https://developer.api.intuit.com/v2/oauth2/tokens/revoke",
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
	return c.exchangeTokens(ctx, "exchange authorization code", form)
}

func (c *IntuitClient) RefreshTokens(ctx context.Context, refreshToken string) (Tokens, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	return c.exchangeTokens(ctx, "refresh tokens", form)
}

func (c *IntuitClient) exchangeTokens(ctx context.Context, operation string, form url.Values) (Tokens, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, fmt.Errorf("create QuickBooks token request: %w", err)
	}
	request.SetBasicAuth(c.clientID, c.clientSecret)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("x-include-refresh-token-hard-expires-in", "true")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return Tokens{}, fmt.Errorf("%s: %w", operation, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return Tokens{}, fmt.Errorf("read QuickBooks token response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return Tokens{}, decodeAPIError(operation, response, body)
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
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return "", decodeAPIError("fetch company", response, body)
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

type qboReference struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

type qboMetadata struct {
	LastUpdatedTime string `json:"LastUpdatedTime"`
}

func (c *IntuitClient) ListCustomers(ctx context.Context, token, realmID string, start, maximum int) (Page[CustomerRecord], error) {
	body, err := c.query(ctx, token, realmID, "Customer", start, maximum)
	if err != nil {
		return Page[CustomerRecord]{}, err
	}
	var items []struct {
		ID               string `json:"Id"`
		DisplayName      string `json:"DisplayName"`
		PrimaryEmailAddr struct {
			Address string `json:"Address"`
		} `json:"PrimaryEmailAddr"`
	}
	if err := decodeQueryItems(body, "Customer", &items); err != nil {
		return Page[CustomerRecord]{}, err
	}
	records := make([]CustomerRecord, 0, len(items))
	for _, item := range items {
		if item.ID == "" || strings.TrimSpace(item.DisplayName) == "" {
			continue
		}
		records = append(records, CustomerRecord{
			ExternalID: externalID("customer", item.ID), Name: strings.TrimSpace(item.DisplayName),
			Email: strings.TrimSpace(item.PrimaryEmailAddr.Address),
		})
	}
	return Page[CustomerRecord]{Items: records, HasMore: len(items) == maximum}, nil
}

func (c *IntuitClient) ListInvoices(ctx context.Context, token, realmID string, start, maximum int) (Page[InvoiceRecord], error) {
	body, err := c.query(ctx, token, realmID, "Invoice", start, maximum)
	if err != nil {
		return Page[InvoiceRecord]{}, err
	}
	var items []struct {
		ID          string       `json:"Id"`
		DocNumber   string       `json:"DocNumber"`
		TxnDate     string       `json:"TxnDate"`
		DueDate     string       `json:"DueDate"`
		TotalAmount json.Number  `json:"TotalAmt"`
		Balance     json.Number  `json:"Balance"`
		TxnStatus   string       `json:"TxnStatus"`
		CustomerRef qboReference `json:"CustomerRef"`
		CurrencyRef qboReference `json:"CurrencyRef"`
		Metadata    qboMetadata  `json:"MetaData"`
	}
	if err := decodeQueryItems(body, "Invoice", &items); err != nil {
		return Page[InvoiceRecord]{}, err
	}
	records := make([]InvoiceRecord, 0, len(items))
	for _, item := range items {
		currency := currencyCode(item.CurrencyRef.Value)
		amount, err := minorUnits(item.TotalAmount, currency)
		if err != nil {
			return Page[InvoiceRecord]{}, fmt.Errorf("map QuickBooks invoice %s total: %w", item.ID, err)
		}
		balance, err := minorUnits(item.Balance, currency)
		if err != nil {
			return Page[InvoiceRecord]{}, fmt.Errorf("map QuickBooks invoice %s balance: %w", item.ID, err)
		}
		issued, err := parseDate(item.TxnDate)
		if err != nil {
			return Page[InvoiceRecord]{}, fmt.Errorf("map QuickBooks invoice %s issue date: %w", item.ID, err)
		}
		due, err := parseDate(item.DueDate)
		if err != nil {
			due = issued
		}
		number := strings.TrimSpace(item.DocNumber)
		if number == "" {
			number = "QBO-" + item.ID
		}
		records = append(records, InvoiceRecord{
			ExternalID: externalID("invoice", item.ID), CustomerExternalID: externalID("customer", item.CustomerRef.Value),
			Number: number, Status: invoiceStatus(item.TxnStatus, balance, amount, due, time.Now()), CurrencyCode: currency,
			AmountMinor: amount, IssuedOn: issued, DueOn: due, SourceUpdatedAt: parseOptionalTime(item.Metadata.LastUpdatedTime),
		})
	}
	return Page[InvoiceRecord]{Items: records, HasMore: len(items) == maximum}, nil
}

func (c *IntuitClient) ListPayments(ctx context.Context, token, realmID string, start, maximum int) (Page[PaymentRecord], error) {
	body, err := c.query(ctx, token, realmID, "Payment", start, maximum)
	if err != nil {
		return Page[PaymentRecord]{}, err
	}
	var items []struct {
		ID          string       `json:"Id"`
		TxnDate     string       `json:"TxnDate"`
		PaymentRef  string       `json:"PaymentRefNum"`
		CurrencyRef qboReference `json:"CurrencyRef"`
		Lines       []struct {
			Amount    json.Number `json:"Amount"`
			LinkedTxn []struct {
				TxnID   string `json:"TxnId"`
				TxnType string `json:"TxnType"`
			} `json:"LinkedTxn"`
		} `json:"Line"`
	}
	if err := decodeQueryItems(body, "Payment", &items); err != nil {
		return Page[PaymentRecord]{}, err
	}
	records := make([]PaymentRecord, 0, len(items))
	for _, item := range items {
		date, err := parseDate(item.TxnDate)
		if err != nil {
			return Page[PaymentRecord]{}, fmt.Errorf("map QuickBooks payment %s date: %w", item.ID, err)
		}
		currency := currencyCode(item.CurrencyRef.Value)
		for lineIndex, line := range item.Lines {
			amount, err := minorUnits(line.Amount, currency)
			if err != nil {
				return Page[PaymentRecord]{}, fmt.Errorf("map QuickBooks payment %s amount: %w", item.ID, err)
			}
			if amount <= 0 {
				continue
			}
			for linkIndex, link := range line.LinkedTxn {
				if !strings.EqualFold(link.TxnType, "Invoice") || link.TxnID == "" {
					continue
				}
				records = append(records, PaymentRecord{
					ExternalID:        externalID("payment", fmt.Sprintf("%s:%d:%d", item.ID, lineIndex, linkIndex)),
					InvoiceExternalID: externalID("invoice", link.TxnID), CurrencyCode: currency,
					AmountMinor: amount, ReceivedOn: date, Reference: strings.TrimSpace(item.PaymentRef),
				})
			}
		}
	}
	return Page[PaymentRecord]{Items: records, HasMore: len(items) == maximum}, nil
}

func (c *IntuitClient) ListExpenses(ctx context.Context, token, realmID string, start, maximum int) (Page[ExpenseRecord], error) {
	body, err := c.query(ctx, token, realmID, "Purchase", start, maximum)
	if err != nil {
		return Page[ExpenseRecord]{}, err
	}
	var items []struct {
		ID          string       `json:"Id"`
		TxnDate     string       `json:"TxnDate"`
		TotalAmount json.Number  `json:"TotalAmt"`
		PaymentType string       `json:"PaymentType"`
		CurrencyRef qboReference `json:"CurrencyRef"`
		EntityRef   qboReference `json:"EntityRef"`
		Lines       []struct {
			AccountDetail *struct {
				AccountRef qboReference `json:"AccountRef"`
			} `json:"AccountBasedExpenseLineDetail"`
		} `json:"Line"`
	}
	if err := decodeQueryItems(body, "Purchase", &items); err != nil {
		return Page[ExpenseRecord]{}, err
	}
	records := make([]ExpenseRecord, 0, len(items))
	for _, item := range items {
		currency := currencyCode(item.CurrencyRef.Value)
		amount, err := minorUnits(item.TotalAmount, currency)
		if err != nil {
			return Page[ExpenseRecord]{}, fmt.Errorf("map QuickBooks purchase %s total: %w", item.ID, err)
		}
		if amount <= 0 {
			continue
		}
		date, err := parseDate(item.TxnDate)
		if err != nil {
			return Page[ExpenseRecord]{}, fmt.Errorf("map QuickBooks purchase %s date: %w", item.ID, err)
		}
		category := strings.TrimSpace(item.PaymentType)
		for _, line := range item.Lines {
			if line.AccountDetail != nil && strings.TrimSpace(line.AccountDetail.AccountRef.Name) != "" {
				category = strings.TrimSpace(line.AccountDetail.AccountRef.Name)
				break
			}
		}
		if category == "" {
			category = "Uncategorized"
		}
		vendor := strings.TrimSpace(item.EntityRef.Name)
		if vendor == "" {
			vendor = "Unknown vendor"
		}
		records = append(records, ExpenseRecord{ExternalID: externalID("purchase", item.ID), Category: category,
			VendorName: vendor, CurrencyCode: currency, AmountMinor: amount, IncurredOn: date})
	}
	return Page[ExpenseRecord]{Items: records, HasMore: len(items) == maximum}, nil
}

func (c *IntuitClient) query(ctx context.Context, token, realmID, entity string, start, maximum int) ([]byte, error) {
	if start < 1 || maximum < 1 || maximum > 1000 {
		return nil, fmt.Errorf("invalid QuickBooks query page")
	}
	query := fmt.Sprintf("SELECT * FROM %s STARTPOSITION %d MAXRESULTS %d", entity, start, maximum)
	endpoint := fmt.Sprintf("%s/v3/company/%s/query?%s", c.apiBaseURL, url.PathEscape(realmID), url.Values{"query": {query}}.Encode())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create QuickBooks %s query: %w", entity, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("query QuickBooks %s: %w", entity, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read QuickBooks %s response: %w", entity, err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, decodeAPIError("query "+entity, response, body)
	}
	return body, nil
}

func decodeQueryItems(body []byte, key string, destination any) error {
	var envelope struct {
		QueryResponse map[string]json.RawMessage `json:"QueryResponse"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("decode QuickBooks query response: %w", err)
	}
	raw, exists := envelope.QueryResponse[key]
	if !exists {
		raw = []byte("[]")
	}
	decoder = json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode QuickBooks %s records: %w", key, err)
	}
	return nil
}

func decodeAPIError(operation string, response *http.Response, body []byte) error {
	apiError := &APIError{Operation: operation, StatusCode: response.StatusCode, RequestID: response.Header.Get("intuit_tid")}
	var payload struct {
		Fault struct {
			Errors []struct {
				Message string `json:"Message"`
				Detail  string `json:"Detail"`
				Code    string `json:"code"`
			} `json:"Error"`
		} `json:"Fault"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if json.Unmarshal(body, &payload) == nil {
		if len(payload.Fault.Errors) > 0 {
			apiError.Code = payload.Fault.Errors[0].Code
			apiError.Message = strings.TrimSpace(payload.Fault.Errors[0].Message + " " + payload.Fault.Errors[0].Detail)
		} else {
			apiError.Code = payload.Error
			apiError.Message = payload.ErrorDescription
		}
	}
	if seconds, err := strconv.Atoi(response.Header.Get("Retry-After")); err == nil && seconds > 0 {
		apiError.RetryDelay = time.Duration(seconds) * time.Second
	} else if retryAt, err := http.ParseTime(response.Header.Get("Retry-After")); err == nil {
		apiError.RetryDelay = time.Until(retryAt)
	}
	return apiError
}

func minorUnits(number json.Number, currency string) (int64, error) {
	if number == "" {
		number = "0"
	}
	amount, ok := new(big.Rat).SetString(number.String())
	if !ok {
		return 0, fmt.Errorf("invalid decimal amount")
	}
	exponent := 2
	switch currency {
	case "BIF", "CLP", "DJF", "GNF", "ISK", "JPY", "KMF", "KRW", "PYG", "RWF", "UGX", "VND", "VUV", "XAF", "XOF", "XPF":
		exponent = 0
	case "BHD", "IQD", "JOD", "KWD", "LYD", "OMR", "TND":
		exponent = 3
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
	amount.Mul(amount, new(big.Rat).SetInt(scale))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(amount.Num(), amount.Denom(), remainder)
	if new(big.Int).Mul(new(big.Int).Abs(remainder), big.NewInt(2)).Cmp(amount.Denom()) >= 0 {
		if amount.Sign() >= 0 {
			quotient.Add(quotient, big.NewInt(1))
		} else {
			quotient.Sub(quotient, big.NewInt(1))
		}
	}
	if !quotient.IsInt64() {
		return 0, fmt.Errorf("amount exceeds supported range")
	}
	return quotient.Int64(), nil
}

func currencyCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return "USD"
	}
	return value
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func parseOptionalTime(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func invoiceStatus(providerStatus string, balance, total int64, due, now time.Time) string {
	if strings.EqualFold(providerStatus, "Voided") {
		return "void"
	}
	if balance <= 0 {
		return "paid"
	}
	if total > balance {
		return "partial"
	}
	if due.Before(now.Truncate(24 * time.Hour)) {
		return "overdue"
	}
	return "open"
}

func externalID(entity, id string) string {
	return "quickbooks:" + entity + ":" + strings.TrimSpace(id)
}

func isUnauthorized(err error) bool {
	var apiError *APIError
	return errors.As(err, &apiError) && apiError.StatusCode == http.StatusUnauthorized
}
