package quickbooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestWebhookSignatureAndCloudEventDecoding(t *testing.T) {
	body := []byte(`[{"id":"evt-1","type":"com.intuit.quickbooks.accounting.invoice.updated","time":"2026-11-19T01:00:00Z","intuitentityid":"42","intuitaccountid":"123456","data":{}}]`)
	mac := hmac.New(sha256.New, []byte("verifier-secret"))
	_, _ = mac.Write(body)
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !validWebhookSignature(body, signature, "verifier-secret") {
		t.Fatal("valid Intuit webhook signature was rejected")
	}
	if validWebhookSignature(body, signature, "wrong-secret") {
		t.Fatal("webhook signed by a different verifier was accepted")
	}
	events, err := decodeWebhookEvents(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EntityType != EntityInvoice || events[0].Operation != "updated" || events[0].RealmID != "123456" {
		t.Fatalf("events = %#v", events)
	}
}

func TestLegacyDeletedWebhookDecoding(t *testing.T) {
	body := []byte(`{"eventNotifications":[{"realmId":"123456","dataChangeEvent":{"entities":[{"name":"BillPayment","id":"9","operation":"Delete","lastUpdated":"2026-11-19T01:00:00+0000"}]}}]}`)
	events, err := decodeWebhookEvents(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].EntityType != EntityBillPayment || events[0].Operation != "delete" || events[0].EventID == "" {
		t.Fatalf("events = %#v", events)
	}
}
