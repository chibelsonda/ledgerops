# Safe migration runbook: introducing multi-currency minor units

This is an expand/backfill/validate/contract plan for changing a live `NUMERIC` amount column to integer minor units without a table-rewrite deployment surprise.

## Risks

- Incorrect rounding changes customer balances.
- A long lock blocks invoice writes.
- New writes arrive while the backfill is running.
- A partial rollback leaves old and new readers disagreeing.

## Expand

1. Add nullable `amount_minor BIGINT` and `currency_code CHAR(3)` columns without defaults.
2. Deploy dual-write code that populates both old and new representations.
3. Add a check that new amounts are non-negative using `NOT VALID`, avoiding an immediate full-table scan.

## Backfill

Backfill in primary-key batches, committing each batch separately. Record the last completed key so the job is resumable. Use the currency exponent when converting rather than assuming every currency has two decimal places.

## Validate

- Compare record counts where the new value is null.
- Compare old and new totals grouped by currency and accounting period.
- Sample invoices, refunds, partial payments, and values at rounding boundaries.
- Validate the check constraint separately during a low-traffic period.

## Cut over

1. Deploy read code that prefers `amount_minor` and alerts on unexpected nulls.
2. Observe one complete accounting close.
3. Set the new columns `NOT NULL` only after validation.
4. Stop dual-writing in a later release.
5. Remove the legacy column in a final, independently reversible release.

## Rollback

Before the contract phase, rollback means directing reads back to the legacy column while dual writes continue. Never remove the old representation in the same release that changes readers.

