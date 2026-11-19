package audit

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

func Write(tx *gorm.DB, organizationID, actor, action, entityType, entityID string, metadata map[string]any) error {
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	if err := tx.Exec(`INSERT INTO audit_events (organization_id,actor,action,entity_type,entity_id,metadata) VALUES (?,?,?,?,?,?::jsonb)`, organizationID, actor, action, entityType, entityID, string(encoded)).Error; err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func DecodeMetadata(encoded string) (map[string]any, error) {
	metadata := make(map[string]any)
	if err := json.Unmarshal([]byte(encoded), &metadata); err != nil {
		return nil, fmt.Errorf("decode audit metadata: %w", err)
	}
	return metadata, nil
}
