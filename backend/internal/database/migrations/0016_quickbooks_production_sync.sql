CREATE TABLE quickbooks_entities (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL CHECK (entity_type IN (
        'Account', 'Bill', 'BillPayment', 'CreditMemo', 'Customer',
        'Invoice', 'Payment', 'Purchase', 'Vendor'
    )),
    external_id TEXT NOT NULL,
    sync_token TEXT,
    display_name TEXT,
    document_number TEXT,
    status TEXT,
    currency_code CHAR(3),
    amount_minor BIGINT,
    balance_minor BIGINT,
    transaction_date DATE,
    source_updated_at TIMESTAMPTZ,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, entity_type, external_id)
);

CREATE INDEX quickbooks_entities_org_type_updated_idx
    ON quickbooks_entities (organization_id, entity_type, source_updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE quickbooks_sync_cursors (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    changed_since TIMESTAMPTZ NOT NULL,
    last_full_sync_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, entity_type)
);

CREATE TABLE sync_run_entities (
    sync_run_id UUID NOT NULL REFERENCES sync_runs(id) ON DELETE CASCADE,
    entity_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    records_processed INTEGER NOT NULL DEFAULT 0 CHECK (records_processed >= 0),
    error_message TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    PRIMARY KEY (sync_run_id, entity_type)
);

CREATE TABLE quickbooks_webhook_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    event_id TEXT NOT NULL UNIQUE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    realm_id TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT,
    operation TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX quickbooks_webhook_events_pending_idx
    ON quickbooks_webhook_events (organization_id, occurred_at)
    WHERE processed_at IS NULL;

CREATE TABLE quickbooks_sync_schedules (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    interval_minutes INTEGER NOT NULL DEFAULT 60 CHECK (interval_minutes BETWEEN 5 AND 10080),
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_enqueued_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX quickbooks_sync_schedules_due_idx
    ON quickbooks_sync_schedules (next_run_at)
    WHERE enabled;

CREATE TABLE quickbooks_outbox (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    aggregate_type TEXT NOT NULL CHECK (aggregate_type IN ('client', 'invoice', 'payment')),
    aggregate_id UUID NOT NULL,
    operation TEXT NOT NULL CHECK (operation IN ('create', 'update')),
    idempotency_key TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    provider_external_id TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX quickbooks_outbox_claim_idx
    ON quickbooks_outbox (status, next_attempt_at, created_at);

ALTER TABLE clients ADD COLUMN source_deleted_at TIMESTAMPTZ;
ALTER TABLE invoices ADD COLUMN source_deleted_at TIMESTAMPTZ;
ALTER TABLE payments ADD COLUMN source_deleted_at TIMESTAMPTZ;
ALTER TABLE expenses ADD COLUMN source_deleted_at TIMESTAMPTZ;

CREATE OR REPLACE FUNCTION prevent_payment_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'payment records are immutable' USING ERRCODE = '55000';
    END IF;
    IF OLD.external_id LIKE 'quickbooks:payment:%'
       AND NEW.source_deleted_at IS DISTINCT FROM OLD.source_deleted_at
       AND (to_jsonb(NEW) - 'source_deleted_at') = (to_jsonb(OLD) - 'source_deleted_at') THEN
        RETURN NEW;
    END IF;
    IF OLD.source = 'manual'
       AND OLD.external_id IS NULL
       AND NEW.source = OLD.source
       AND NEW.external_id IS NOT NULL
       AND (to_jsonb(NEW) - 'external_id') = (to_jsonb(OLD) - 'external_id') THEN
        RETURN NEW;
    END IF;
    RAISE EXCEPTION 'payment records are immutable' USING ERRCODE = '55000';
END;
$$;

DROP VIEW invoice_balances;

CREATE VIEW invoice_balances AS
WITH payment_totals AS (
    SELECT organization_id, invoice_id, SUM(amount_minor)::bigint AS payment_minor
    FROM payments
    WHERE source_deleted_at IS NULL
    GROUP BY organization_id, invoice_id
), adjustment_totals AS (
    SELECT
        organization_id,
        invoice_id,
        COALESCE(SUM(amount_minor) FILTER (WHERE kind = 'credit_note' AND status = 'approved'), 0)::bigint AS credit_minor,
        COALESCE(SUM(amount_minor) FILTER (WHERE kind = 'payment_reversal' AND status = 'approved'), 0)::bigint AS reversal_minor
    FROM invoice_adjustments
    GROUP BY organization_id, invoice_id
)
SELECT
    i.organization_id,
    i.id AS invoice_id,
    i.amount_minor AS gross_minor,
    COALESCE(a.credit_minor, 0)::bigint AS credit_minor,
    COALESCE(p.payment_minor, 0)::bigint AS payment_minor,
    COALESCE(a.reversal_minor, 0)::bigint AS reversal_minor,
    (COALESCE(p.payment_minor, 0) - COALESCE(a.reversal_minor, 0))::bigint AS paid_minor,
    (i.amount_minor - COALESCE(a.credit_minor, 0))::bigint AS effective_amount_minor,
    GREATEST(
        i.amount_minor - COALESCE(a.credit_minor, 0) - COALESCE(p.payment_minor, 0) + COALESCE(a.reversal_minor, 0),
        0
    )::bigint AS outstanding_minor
FROM invoices i
LEFT JOIN payment_totals p ON p.organization_id = i.organization_id AND p.invoice_id = i.id
LEFT JOIN adjustment_totals a ON a.organization_id = i.organization_id AND a.invoice_id = i.id
WHERE i.source_deleted_at IS NULL;
