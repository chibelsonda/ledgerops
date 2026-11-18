CREATE TABLE invoice_adjustments (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invoice_id UUID NOT NULL REFERENCES invoices(id),
    payment_id UUID REFERENCES payments(id),
    kind TEXT NOT NULL CHECK (kind IN ('credit_note', 'payment_reversal')),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 3 AND 500),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_by UUID NOT NULL REFERENCES users(id),
    reviewed_by UUID REFERENCES users(id),
    review_note TEXT CHECK (review_note IS NULL OR char_length(review_note) BETWEEN 3 AND 500),
    requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_at TIMESTAMPTZ,
    CONSTRAINT adjustment_kind_payment CHECK (
        (kind = 'payment_reversal' AND payment_id IS NOT NULL)
        OR (kind = 'credit_note' AND payment_id IS NULL)
    ),
    CONSTRAINT adjustment_review_state CHECK (
        (status = 'pending' AND reviewed_by IS NULL AND reviewed_at IS NULL AND review_note IS NULL)
        OR (status IN ('approved', 'rejected') AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND review_note IS NOT NULL)
    )
);

CREATE INDEX invoice_adjustments_invoice_created_idx
    ON invoice_adjustments (invoice_id, requested_at DESC);

CREATE UNIQUE INDEX invoice_adjustments_active_reversal_idx
    ON invoice_adjustments (payment_id)
    WHERE kind = 'payment_reversal' AND status IN ('pending', 'approved');

CREATE FUNCTION protect_finalized_adjustment()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'adjustment records cannot be deleted' USING ERRCODE = '55000';
    END IF;
    IF OLD.status <> 'pending' THEN
        RAISE EXCEPTION 'finalized adjustment records are immutable' USING ERRCODE = '55000';
    END IF;
    IF NEW.id <> OLD.id
       OR NEW.organization_id <> OLD.organization_id
       OR NEW.invoice_id <> OLD.invoice_id
       OR NEW.payment_id IS DISTINCT FROM OLD.payment_id
       OR NEW.kind <> OLD.kind
       OR NEW.amount_minor <> OLD.amount_minor
       OR NEW.reason <> OLD.reason
       OR NEW.requested_by <> OLD.requested_by
       OR NEW.requested_at <> OLD.requested_at
       OR NEW.status NOT IN ('approved', 'rejected') THEN
        RAISE EXCEPTION 'only a pending adjustment decision is allowed' USING ERRCODE = '55000';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER invoice_adjustments_are_immutable
    BEFORE UPDATE OR DELETE ON invoice_adjustments
    FOR EACH ROW
    EXECUTE FUNCTION protect_finalized_adjustment();
