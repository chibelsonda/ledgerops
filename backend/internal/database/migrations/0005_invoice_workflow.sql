CREATE INDEX audit_events_org_entity_created_idx
    ON audit_events (organization_id, entity_type, entity_id, created_at DESC);

CREATE INDEX clients_org_name_idx
    ON clients (organization_id, name);
