-- Public resources use time-ordered UUIDv7 identifiers for API-safe IDs with
-- better B-tree insertion locality. Existing UUIDv4 values remain unchanged.
ALTER TABLE organizations
    ALTER COLUMN id SET DEFAULT uuidv7();

ALTER TABLE clients
    ALTER COLUMN id SET DEFAULT uuidv7();

ALTER TABLE invoices
    ALTER COLUMN id SET DEFAULT uuidv7();

ALTER TABLE payments
    ALTER COLUMN id SET DEFAULT uuidv7();

ALTER TABLE expenses
    ALTER COLUMN id SET DEFAULT uuidv7();

ALTER TABLE budgets
    ALTER COLUMN id SET DEFAULT uuidv7();

ALTER TABLE sync_runs
    ALTER COLUMN id SET DEFAULT uuidv7();

ALTER TABLE reconciliation_items
    ALTER COLUMN id SET DEFAULT uuidv7();
