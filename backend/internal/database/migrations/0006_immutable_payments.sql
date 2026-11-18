ALTER TABLE payments
    ADD COLUMN reference TEXT,
    ADD COLUMN created_by UUID REFERENCES users(id);

CREATE INDEX payments_invoice_received_idx
    ON payments (invoice_id, received_on DESC, created_at DESC);

CREATE FUNCTION prevent_payment_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'payment records are immutable'
        USING ERRCODE = '55000';
END;
$$;

CREATE TRIGGER payments_are_immutable
    BEFORE UPDATE OR DELETE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION prevent_payment_mutation();
