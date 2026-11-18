ALTER TABLE clients
    ADD COLUMN payment_terms_days INTEGER NOT NULL DEFAULT 30 CHECK (payment_terms_days BETWEEN 0 AND 365),
    ADD COLUMN reminders_enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN reminder_delay_days INTEGER NOT NULL DEFAULT 3 CHECK (reminder_delay_days BETWEEN 0 AND 90),
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE client_contacts (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
    email TEXT NOT NULL,
    phone TEXT,
    role_title TEXT,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (client_id, email)
);

CREATE UNIQUE INDEX client_contacts_one_primary_idx
    ON client_contacts (client_id) WHERE is_primary;

CREATE INDEX client_contacts_client_idx
    ON client_contacts (client_id, is_primary DESC, name);

INSERT INTO client_contacts (organization_id, client_id, name, email, role_title, is_primary)
SELECT organization_id, id, name || ' billing', email, 'Billing contact', true
FROM clients
WHERE email IS NOT NULL AND email <> ''
ON CONFLICT (client_id, email) DO NOTHING;

CREATE TABLE invoice_reminders (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    client_id UUID NOT NULL REFERENCES clients(id),
    contact_id UUID NOT NULL REFERENCES client_contacts(id),
    sequence INTEGER NOT NULL DEFAULT 1 CHECK (sequence > 0),
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sent', 'cancelled')),
    scheduled_for DATE NOT NULL,
    recipient_email TEXT NOT NULL,
    subject TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancelled_by UUID REFERENCES users(id),
    UNIQUE (invoice_id, sequence)
);

CREATE INDEX invoice_reminders_org_status_schedule_idx
    ON invoice_reminders (organization_id, status, scheduled_for, created_at DESC);

CREATE FUNCTION protect_finalized_reminder()
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
    IF NEW.id <> OLD.id OR NEW.organization_id <> OLD.organization_id OR NEW.invoice_id <> OLD.invoice_id
       OR NEW.client_id <> OLD.client_id OR NEW.contact_id <> OLD.contact_id OR NEW.sequence <> OLD.sequence
       OR NEW.scheduled_for <> OLD.scheduled_for OR NEW.recipient_email <> OLD.recipient_email
       OR NEW.subject <> OLD.subject OR NEW.message <> OLD.message OR NEW.created_at <> OLD.created_at
       OR NEW.status NOT IN ('sent', 'cancelled') THEN
        RAISE EXCEPTION 'only a queued reminder status transition is allowed' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER invoice_reminders_are_immutable
    BEFORE UPDATE OR DELETE ON invoice_reminders
    FOR EACH ROW
    EXECUTE FUNCTION protect_finalized_reminder();
