ALTER TABLE invoice_reminders
    ADD CONSTRAINT invoice_reminders_status_state CHECK (
        (status = 'queued' AND sent_at IS NULL AND cancelled_at IS NULL AND cancelled_by IS NULL)
        OR (status = 'sent' AND sent_at IS NOT NULL AND cancelled_at IS NULL AND cancelled_by IS NULL)
        OR (status = 'cancelled' AND sent_at IS NULL AND cancelled_at IS NOT NULL AND cancelled_by IS NOT NULL)
    );
