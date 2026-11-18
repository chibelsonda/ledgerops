# Financial controls and forecasting

## Month-end close

Each close period represents one accounting month and receives the same six control tasks: cash reconciliation, receivables review, accruals, operating expenses, project billing, and management variance review. Completion requires a written evidence note.

The lifecycle is `draft → in_review → closed`. A controller creates and submits the period. An admin who did not submit it approves or returns it for changes. Approval writes an audit event and permanently locks both the period and its task evidence at the PostgreSQL layer.

The reconciliation queue remains a separate exception-management workflow. Its open-item count is displayed in the close workspace as context, while checklist evidence records how those exceptions were considered during close.

## 13-week cash forecast

Forecasts start on the current Monday and use integer minor units throughout. Every scenario exposes four assumptions:

- collection percentage applied to open invoice balances;
- collection delay applied to invoice and milestone dates;
- growth applied to the trailing 13-week operating-expense run-rate;
- realization probability applied to approved, unbilled project milestones.

Invoice balances account for payments, approved credit notes, and approved payment reversals. Planned project milestones are included only when their contract is approved. Expected amounts outside the 13-week horizon are intentionally excluded.

Manual one-off inflows and outflows are append-only. They carry an author and timestamp and also write an audit event, preserving the reasoning behind scenario changes.

This is a short-term direct cash forecast, not a GAAP income statement projection. Future production increments would add bank-account opening balances, scheduled payables, payroll calendars, foreign exchange, scenario versioning, and forecast-versus-actual accuracy reporting.
