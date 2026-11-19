INSERT INTO quickbooks_sync_schedules (organization_id, enabled, next_run_at)
SELECT organization_id, true, now()
FROM quickbooks_connections
ON CONFLICT (organization_id) DO NOTHING;

