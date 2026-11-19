package quickbooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (c *IntuitClient) RevokeToken(ctx context.Context, token string) error {
	body, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		return fmt.Errorf("encode QuickBooks revoke request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.revokeURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create QuickBooks revoke request: %w", err)
	}
	request.SetBasicAuth(c.clientID, c.clientSecret)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("revoke QuickBooks token: %w", err)
	}
	defer response.Body.Close()
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if readErr != nil {
		return fmt.Errorf("read QuickBooks revoke response: %w", readErr)
	}
	if response.StatusCode != http.StatusOK {
		return decodeAPIError("revoke token", response, responseBody)
	}
	return nil
}

func (c *IntuitClient) WriteEntity(ctx context.Context, token, realmID, entityType string, payload map[string]any) (RemoteEntity, error) {
	if entityType != EntityCustomer && entityType != EntityInvoice && entityType != EntityPayment {
		return RemoteEntity{}, &PermanentError{Err: fmt.Errorf("unsupported outbound QuickBooks entity %q", entityType)}
	}
	if entityType == EntityInvoice {
		if err := c.addDefaultSalesItem(ctx, token, realmID, payload); err != nil {
			return RemoteEntity{}, err
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return RemoteEntity{}, &PermanentError{Err: fmt.Errorf("encode QuickBooks %s: %w", entityType, err)}
	}
	endpoint := fmt.Sprintf("%s/v3/company/%s/%s?minorversion=75", c.apiBaseURL, url.PathEscape(realmID), strings.ToLower(entityType))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return RemoteEntity{}, fmt.Errorf("create QuickBooks %s write request: %w", entityType, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return RemoteEntity{}, fmt.Errorf("write QuickBooks %s: %w", entityType, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 5<<20))
	if err != nil {
		return RemoteEntity{}, fmt.Errorf("read QuickBooks %s write response: %w", entityType, err)
	}
	if response.StatusCode != http.StatusOK {
		return RemoteEntity{}, decodeAPIError("write "+entityType, response, responseBody)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return RemoteEntity{}, fmt.Errorf("decode QuickBooks %s write response: %w", entityType, err)
	}
	raw, ok := envelope[entityType]
	if !ok {
		return RemoteEntity{}, &PermanentError{Err: fmt.Errorf("QuickBooks %s write response is incomplete", entityType)}
	}
	return mapRemoteEntity(entityType, raw, nil)
}

func (c *IntuitClient) addDefaultSalesItem(ctx context.Context, token, realmID string, payload map[string]any) error {
	body, err := c.query(ctx, token, realmID, "Item", 1, 100)
	if err != nil {
		return err
	}
	items, err := decodeRawQueryItems(body, "Item")
	if err != nil {
		return err
	}
	itemID := ""
	for _, raw := range items {
		var item struct {
			ID, Type string
			Active   bool
		}
		if err := json.Unmarshal(raw, &item); err == nil && item.ID != "" && item.Active && (item.Type == "Service" || item.Type == "NonInventory") {
			itemID = item.ID
			break
		}
	}
	if itemID == "" {
		return &PermanentError{Err: fmt.Errorf("QuickBooks requires an active service or non-inventory item for outbound invoices")}
	}
	lines, ok := payload["Line"].([]any)
	if !ok {
		return &PermanentError{Err: fmt.Errorf("outbound invoice has no lines")}
	}
	for _, value := range lines {
		line, ok := value.(map[string]any)
		if !ok {
			continue
		}
		detail, ok := line["SalesItemLineDetail"].(map[string]any)
		if ok {
			detail["ItemRef"] = map[string]string{"value": itemID}
		}
	}
	return nil
}

func (c *IntuitClient) ListRemoteEntities(
	ctx context.Context,
	token, realmID, entityType string,
	start, maximum int,
) (Page[RemoteEntity], error) {
	if !supportedEntityType(entityType) {
		return Page[RemoteEntity]{}, fmt.Errorf("unsupported QuickBooks entity type %q", entityType)
	}
	body, err := c.query(ctx, token, realmID, entityType, start, maximum)
	if err != nil {
		return Page[RemoteEntity]{}, err
	}
	items, err := decodeRawQueryItems(body, entityType)
	if err != nil {
		return Page[RemoteEntity]{}, err
	}
	records := make([]RemoteEntity, 0, len(items))
	for _, item := range items {
		record, err := mapRemoteEntity(entityType, item, nil)
		if err != nil {
			return Page[RemoteEntity]{}, err
		}
		records = append(records, record)
	}
	return Page[RemoteEntity]{Items: records, HasMore: len(items) == maximum}, nil
}

func (c *IntuitClient) ChangeDataCapture(
	ctx context.Context,
	token, realmID string,
	entityTypes []string,
	changedSince time.Time,
) (CDCResult, error) {
	if len(entityTypes) == 0 {
		return CDCResult{CursorAt: time.Now().UTC()}, nil
	}
	for _, entityType := range entityTypes {
		if !supportedEntityType(entityType) {
			return CDCResult{}, fmt.Errorf("unsupported QuickBooks CDC entity type %q", entityType)
		}
	}
	cursorAt := time.Now().UTC()
	query := url.Values{
		"entities":     {strings.Join(entityTypes, ",")},
		"changedSince": {changedSince.UTC().Format(time.RFC3339)},
	}
	endpoint := fmt.Sprintf("%s/v3/company/%s/cdc?%s", c.apiBaseURL, url.PathEscape(realmID), query.Encode())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return CDCResult{}, fmt.Errorf("create QuickBooks CDC request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return CDCResult{}, fmt.Errorf("query QuickBooks CDC: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 20<<20))
	if err != nil {
		return CDCResult{}, fmt.Errorf("read QuickBooks CDC response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return CDCResult{}, decodeAPIError("change data capture", response, body)
	}

	var envelope struct {
		CDCResponses []struct {
			QueryResponses []map[string]json.RawMessage `json:"QueryResponse"`
		} `json:"CDCResponse"`
		Time string `json:"time"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		return CDCResult{}, fmt.Errorf("decode QuickBooks CDC response: %w", err)
	}
	if parsed := parseOptionalTime(envelope.Time); parsed != nil {
		cursorAt = *parsed
	}

	result := CDCResult{CursorAt: cursorAt, Entities: make([]RemoteEntity, 0)}
	for _, response := range envelope.CDCResponses {
		for _, queryResponse := range response.QueryResponses {
			for _, entityType := range entityTypes {
				rawItems, exists := queryResponse[entityType]
				if !exists {
					continue
				}
				var items []json.RawMessage
				decoder := json.NewDecoder(bytes.NewReader(rawItems))
				decoder.UseNumber()
				if err := decoder.Decode(&items); err != nil {
					return CDCResult{}, fmt.Errorf("decode QuickBooks CDC %s records: %w", entityType, err)
				}
				for _, item := range items {
					record, err := mapRemoteEntity(entityType, item, &cursorAt)
					if err != nil {
						return CDCResult{}, err
					}
					result.Entities = append(result.Entities, record)
				}
			}
		}
	}
	return result, nil
}

func decodeRawQueryItems(body []byte, entityType string) ([]json.RawMessage, error) {
	var envelope struct {
		QueryResponse map[string]json.RawMessage `json:"QueryResponse"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode QuickBooks %s query response: %w", entityType, err)
	}
	raw, exists := envelope.QueryResponse[entityType]
	if !exists {
		return []json.RawMessage{}, nil
	}
	var items []json.RawMessage
	decoder = json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&items); err != nil {
		return nil, fmt.Errorf("decode QuickBooks %s records: %w", entityType, err)
	}
	return items, nil
}

func mapRemoteEntity(entityType string, raw json.RawMessage, deletedAtFallback *time.Time) (RemoteEntity, error) {
	var item struct {
		ID          string       `json:"Id"`
		SyncToken   string       `json:"SyncToken"`
		DisplayName string       `json:"DisplayName"`
		Name        string       `json:"Name"`
		DocNumber   string       `json:"DocNumber"`
		TxnStatus   string       `json:"TxnStatus"`
		Status      string       `json:"status"`
		TxnDate     string       `json:"TxnDate"`
		TotalAmount json.Number  `json:"TotalAmt"`
		Balance     json.Number  `json:"Balance"`
		CurrencyRef qboReference `json:"CurrencyRef"`
		Metadata    qboMetadata  `json:"MetaData"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&item); err != nil {
		return RemoteEntity{}, fmt.Errorf("decode QuickBooks %s entity: %w", entityType, err)
	}
	if strings.TrimSpace(item.ID) == "" {
		return RemoteEntity{}, &PermanentError{Err: fmt.Errorf("QuickBooks %s entity is missing Id", entityType)}
	}
	record := RemoteEntity{
		EntityType: entityType, ExternalID: item.ID, SyncToken: item.SyncToken,
		DisplayName: firstNonEmpty(item.DisplayName, item.Name), DocumentNumber: item.DocNumber,
		Status: firstNonEmpty(item.TxnStatus, item.Status), CurrencyCode: currencyCode(item.CurrencyRef.Value),
		SourceUpdatedAt: parseOptionalTime(item.Metadata.LastUpdatedTime), Payload: append(json.RawMessage(nil), raw...),
	}
	if item.TotalAmount != "" {
		amount, err := minorUnits(item.TotalAmount, record.CurrencyCode)
		if err != nil {
			return RemoteEntity{}, &PermanentError{Err: fmt.Errorf("map QuickBooks %s %s total: %w", entityType, item.ID, err)}
		}
		record.AmountMinor = &amount
	}
	if item.Balance != "" {
		balance, err := minorUnits(item.Balance, record.CurrencyCode)
		if err != nil {
			return RemoteEntity{}, &PermanentError{Err: fmt.Errorf("map QuickBooks %s %s balance: %w", entityType, item.ID, err)}
		}
		record.BalanceMinor = &balance
	}
	if item.TxnDate != "" {
		date, err := parseDate(item.TxnDate)
		if err != nil {
			return RemoteEntity{}, &PermanentError{Err: fmt.Errorf("map QuickBooks %s %s date: %w", entityType, item.ID, err)}
		}
		record.TransactionDate = &date
	}
	if strings.EqualFold(item.Status, "Deleted") {
		deletedAt := deletedAtFallback
		if record.SourceUpdatedAt != nil {
			deletedAt = record.SourceUpdatedAt
		}
		if deletedAt == nil {
			now := time.Now().UTC()
			deletedAt = &now
		}
		record.DeletedAt = deletedAt
	}
	return record, nil
}

func supportedEntityType(entityType string) bool {
	for _, supported := range ImportEntityTypes {
		if entityType == supported {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
