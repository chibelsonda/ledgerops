ALTER TABLE invoice_reminders
    DROP CONSTRAINT invoice_reminders_status_state,
    DROP CONSTRAINT invoice_reminders_status_check,
    ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    ADD COLUMN max_attempts INTEGER NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    ADD COLUMN next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN locked_at TIMESTAMPTZ,
    ADD COLUMN last_error TEXT,
    ADD COLUMN provider_message_id TEXT,
    ADD COLUMN failed_at TIMESTAMPTZ,
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD CONSTRAINT invoice_reminders_status_check CHECK (status IN ('queued', 'sent', 'cancelled', 'failed')),
    ADD CONSTRAINT invoice_reminders_status_state CHECK (
        (status = 'queued' AND sent_at IS NULL AND cancelled_at IS NULL AND cancelled_by IS NULL AND failed_at IS NULL)
        OR (status = 'sent' AND sent_at IS NOT NULL AND cancelled_at IS NULL AND cancelled_by IS NULL AND failed_at IS NULL)
        OR (status = 'cancelled' AND sent_at IS NULL AND cancelled_at IS NOT NULL AND cancelled_by IS NOT NULL AND failed_at IS NULL)
        OR (status = 'failed' AND sent_at IS NULL AND cancelled_at IS NULL AND cancelled_by IS NULL AND failed_at IS NOT NULL)
    );

CREATE UNIQUE INDEX invoice_reminders_provider_message_idx
    ON invoice_reminders (provider_message_id) WHERE provider_message_id IS NOT NULL;

CREATE INDEX invoice_reminders_claim_idx
    ON invoice_reminders (next_attempt_at, scheduled_for, created_at)
    WHERE status = 'queued';

CREATE OR REPLACE FUNCTION protect_finalized_reminder()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'reminder records cannot be deleted' USING ERRCODE = '55000';
    END IF;
    IF OLD.status <> 'queued' THEN
        RAISE EXCEPTION 'finalized reminder records are immutable' USING ERRCODE = '55000';
    END IF;
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.organization_id IS DISTINCT FROM OLD.organization_id
       OR NEW.invoice_id IS DISTINCT FROM OLD.invoice_id
       OR NEW.client_id IS DISTINCT FROM OLD.client_id
       OR NEW.contact_id IS DISTINCT FROM OLD.contact_id
       OR NEW.sequence IS DISTINCT FROM OLD.sequence
       OR NEW.scheduled_for IS DISTINCT FROM OLD.scheduled_for
       OR NEW.recipient_email IS DISTINCT FROM OLD.recipient_email
       OR NEW.subject IS DISTINCT FROM OLD.subject
       OR NEW.message IS DISTINCT FROM OLD.message
       OR NEW.created_at IS DISTINCT FROM OLD.created_at
       OR NEW.status NOT IN ('queued', 'sent', 'cancelled', 'failed') THEN
        RAISE EXCEPTION 'reminder content is immutable' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TABLE reminder_delivery_events (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    reminder_id UUID NOT NULL REFERENCES invoice_reminders(id),
    provider_event_id TEXT NOT NULL,
    provider_message_id TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('delivered', 'bounced', 'complained', 'deferred')),
    occurred_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider_event_id)
);

CREATE INDEX reminder_delivery_events_reminder_idx
    ON reminder_delivery_events (reminder_id, occurred_at DESC, received_at DESC);

CREATE FUNCTION protect_delivery_event()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'delivery events are append-only' USING ERRCODE = '55000';
END;
$$;

CREATE TRIGGER reminder_delivery_events_are_append_only
    BEFORE UPDATE OR DELETE ON reminder_delivery_events
    FOR EACH ROW
    EXECUTE FUNCTION protect_delivery_event();
