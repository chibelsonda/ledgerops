CREATE VIEW invoice_balances AS
WITH payment_totals AS (
    SELECT organization_id, invoice_id, SUM(amount_minor)::bigint AS payment_minor
    FROM payments
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
LEFT JOIN adjustment_totals a ON a.organization_id = i.organization_id AND a.invoice_id = i.id;
