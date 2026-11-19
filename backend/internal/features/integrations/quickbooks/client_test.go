package quickbooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRefreshTokensStoresRotatedValues(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s", request.Method)
		}
		if username, password, ok := request.BasicAuth(); !ok || username != "client" || password != "secret" {
			t.Fatal("missing client credentials")
		}
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("grant_type") != "refresh_token" || request.Form.Get("refresh_token") != "old-refresh" {
			t.Fatalf("unexpected form: %v", request.Form)
		}
		return jsonResponse(http.StatusOK, `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600,"x_refresh_token_expires_in":8640000}`), nil
	})

	client := NewIntuitClient("client", "secret", "https://example.test/callback", "https://quickbooks.test")
	client.tokenURL = "https://oauth.test/token"
	client.httpClient = &http.Client{Transport: transport}
	tokens, err := client.RefreshTokens(context.Background(), "old-refresh")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "new-access" || tokens.RefreshToken != "new-refresh" {
		t.Fatalf("tokens = %#v", tokens)
	}
}

func TestListInvoicesMapsMoneyAndPagination(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Header.Get("Authorization") != "Bearer access" {
			t.Fatal("missing bearer token")
		}
		query, err := url.QueryUnescape(request.URL.Query().Get("query"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(query, "FROM Invoice") || !strings.Contains(query, "STARTPOSITION 1 MAXRESULTS 1") {
			t.Fatalf("query = %q", query)
		}
		return jsonResponse(http.StatusOK, `{"QueryResponse":{"Invoice":[{"Id":"42","DocNumber":"1001","TxnDate":"2026-11-01","DueDate":"2026-11-30","TotalAmt":123.45,"Balance":23.45,"CustomerRef":{"value":"7"},"CurrencyRef":{"value":"USD"},"MetaData":{"LastUpdatedTime":"2026-11-19T00:00:00Z"}}]}}`), nil
	})

	client := NewIntuitClient("client", "secret", "https://example.test/callback", "https://quickbooks.test")
	client.httpClient = &http.Client{Transport: transport}
	page, err := client.ListInvoices(context.Background(), "access", "123456", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !page.HasMore || len(page.Items) != 1 {
		t.Fatalf("page = %#v", page)
	}
	invoice := page.Items[0]
	if invoice.ExternalID != "quickbooks:invoice:42" || invoice.CustomerExternalID != "quickbooks:customer:7" {
		t.Fatalf("invoice IDs = %#v", invoice)
	}
	if invoice.AmountMinor != 12345 || invoice.Status != "partial" {
		t.Fatalf("invoice money/status = %#v", invoice)
	}
}

func TestAPIErrorExposesRetryPolicyWithoutBody(t *testing.T) {
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		response := jsonResponse(http.StatusTooManyRequests, `{"Fault":{"Error":[{"Message":"Rate limit","Detail":"Try later","code":"003001"}]}}`)
		response.Header.Set("Retry-After", "17")
		response.Header.Set("intuit_tid", "request-123")
		return response, nil
	})

	client := NewIntuitClient("client", "secret", "https://example.test/callback", "https://quickbooks.test")
	client.httpClient = &http.Client{Transport: transport}
	_, err := client.ListCustomers(context.Background(), "access", "123456", 1, 100)
	apiError, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error = %T %v", err, err)
	}
	if !apiError.Retryable() || apiError.RetryAfter() != 17*time.Second || !strings.Contains(err.Error(), "request-123") {
		t.Fatalf("unexpected retry policy: %#v", apiError)
	}
}

func TestRevokeTokenUsesIntuitContract(t *testing.T) {
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected request: %s", request.Method)
		}
		username, password, ok := request.BasicAuth()
		if !ok || username != "client" || password != "secret" {
			t.Fatal("missing basic auth")
		}
		body, _ := io.ReadAll(request.Body)
		if string(body) != `{"token":"refresh-token"}` {
			t.Fatalf("body = %s", body)
		}
		return jsonResponse(http.StatusOK, `{}`), nil
	})
	client := NewIntuitClient("client", "secret", "https://example.test/callback", "https://quickbooks.test")
	client.revokeURL = "https://oauth.test/revoke"
	client.httpClient = &http.Client{Transport: transport}
	if err := client.RevokeToken(context.Background(), "refresh-token"); err != nil {
		t.Fatal(err)
	}
}

func TestWriteInvoiceResolvesSalesItem(t *testing.T) {
	requests := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if request.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, `{"QueryResponse":{"Item":[{"Id":"7","Type":"Service","Active":true}]}}`), nil
		}
		body, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(body), `"ItemRef":{"value":"7"}`) {
			t.Fatalf("invoice body = %s", body)
		}
		return jsonResponse(http.StatusOK, `{"Invoice":{"Id":"99","SyncToken":"0","DocNumber":"L-1","TxnDate":"2026-11-19","TotalAmt":25,"Balance":25,"CurrencyRef":{"value":"USD"}}}`), nil
	})
	client := NewIntuitClient("client", "secret", "https://example.test/callback", "https://quickbooks.test")
	client.httpClient = &http.Client{Transport: transport}
	payload := map[string]any{"Line": []any{map[string]any{"SalesItemLineDetail": map[string]any{}}}}
	entity, err := client.WriteEntity(context.Background(), "access", "123456", EntityInvoice, payload)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 || entity.ExternalID != "99" || entity.SyncToken != "0" {
		t.Fatalf("requests=%d entity=%#v", requests, entity)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestMinorUnitsUsesCurrencyExponent(t *testing.T) {
	tests := []struct {
		amount   string
		currency string
		want     int64
	}{
		{"12.34", "USD", 1234},
		{"12.5", "JPY", 13},
		{"1.2345", "KWD", 1235},
	}
	for _, test := range tests {
		got, err := minorUnits(json.Number(test.amount), test.currency)
		if err != nil {
			t.Fatal(err)
		}
		if got != test.want {
			t.Fatalf("minorUnits(%s, %s) = %d, want %d", test.amount, test.currency, got, test.want)
		}
	}
}
