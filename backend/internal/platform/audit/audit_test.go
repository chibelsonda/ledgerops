package audit

import "testing"

func TestDecodeMetadata(t *testing.T) {
	metadata, err := DecodeMetadata(`{"note":"approved","amountMinor":1250}`)
	if err != nil {
		t.Fatalf("DecodeMetadata() error = %v", err)
	}
	if metadata["note"] != "approved" || metadata["amountMinor"] != float64(1250) {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestDecodeMetadataRejectsInvalidJSON(t *testing.T) {
	if _, err := DecodeMetadata(`{"note":`); err == nil {
		t.Fatal("DecodeMetadata accepted invalid JSON")
	}
}
