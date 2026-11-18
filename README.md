# LedgerOps

LedgerOps is a production-shaped FinOps portfolio project: a Go and PostgreSQL backend, a React/TypeScript client, and a background synchronization worker. It gives accountants one place to inspect cash, receivables, budgets, integration health, and reconciliation exceptions.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white) ![GORM](https://img.shields.io/badge/GORM-PostgreSQL-00ADD8) ![React](https://img.shields.io/badge/React-TypeScript-149ECA?logo=react&logoColor=white) ![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-4169E1?logo=postgresql&logoColor=white)

## What it demonstrates

- Correct money handling with integer minor units
- Time-ordered UUIDv7 identifiers for public API resources
- Multi-tenant query scoping and database constraints
- Financial summaries, A/R aging, and budget-to-actual reporting
- Client contacts, payment terms, and automated overdue-reminder policies
- Project contracts, controlled change orders, milestone billing, and two-person approvals
- SMTP reminder delivery with bounded retries and webhook delivery tracking
- PostgreSQL-backed background jobs with idempotency and retry state
- Reconciliation decisions coupled transactionally to audit events
- GORM repositories with explicit SQL for auditable financial reporting
- Feature-based Handler-Service-Repository Go architecture with constructor dependency injection
- Headless UI controls styled with Tailwind CSS
- Zod and React Hook Form validation
- Go unit and HTTP tests, Vitest, Playwright, and CI
- Safe migration and third-party integration runbooks
- QuickBooks sandbox OAuth with hashed state and AES-256-GCM token encryption
- Argon2id passwords, hashed opaque sessions, CSRF protection, and workspace RBAC

## Run it natively

Prerequisites: Go 1.25+, Node.js 22+, and PostgreSQL 18. Start your installed PostgreSQL cluster and create the application database once:

```bash
sudo pg_ctlcluster 18 main start
createdb ledgerops
```

The default connection uses the PostgreSQL Unix socket and current operating-system user through peer authentication. If that role does not exist yet, create it once with `sudo -u postgres createuser --createdb "$USER"`. Override `DATABASE_URL` if your cluster uses a different role, password, port, or database.

Run each process in its own terminal:

```bash
make api
make worker
make web
```

Open <http://localhost:5173>. The API health endpoint is <http://localhost:8080/healthz>. The API applies embedded migrations and idempotent demo seed data on startup. Use the controller demo account shown on the login screen:

```text
mara@ledgerops.demo
LedgerOpsDemo!2026
```

To connect a real QuickBooks Online sandbox, add the development credentials and a stable token-encryption key described in [the integration runbook](docs/integration-runbook.md), restart the API, and choose **Data sync → Connect QuickBooks**.

To deliver collection reminders, set `SMTP_HOST`, `SMTP_PORT`, and `EMAIL_FROM_ADDRESS` in `ledgerops/.env` (plus credentials when required), then restart the worker. Set `EMAIL_WEBHOOK_SECRET` on the API and on the trusted email relay that posts normalized delivery events to `POST /api/v1/integrations/email/webhook`. If SMTP is omitted, reminders remain queued and are never reported as sent.

## Test it

```bash
make test
npm --prefix frontend exec playwright test
```

Playwright expects the native API, worker, web client, and PostgreSQL service to be running.
If the bundled Playwright browser is unavailable but Chrome is installed locally, run
`PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH=/usr/bin/google-chrome npm --prefix frontend run test:e2e`.

## Project map

```text
backend/
  cmd/api/             HTTP process
  cmd/worker/          background sync process
  internal/app/        route composition
  internal/database/   connection, embedded migrations, demo seed
  internal/features/   dashboard, projects, invoices, clients, reminders, reconciliation, and sync slices
  internal/financial/  pure financial rules
  internal/jobs/       queue orchestration
  internal/platform/   shared HTTP and application error infrastructure
frontend/
  src/app/             route and application shell composition
  src/features/        feature-owned API, types, schemas, and views
  src/shared/          Headless UI controls, HTTP, and formatting
  e2e/                 Playwright workflows
docs/                  architecture and operational runbooks
```

## Intentional demo boundaries

The app includes a real QuickBooks sandbox OAuth connection and verifies the authorized company through the Accounting API. The sync worker still processes the seeded QuickBooks-shaped snapshot so reviewers can exercise the queue without private credentials; importing remote entities through the stored connection is the next adapter increment. See [the integration runbook](docs/integration-runbook.md) and [architecture notes](docs/architecture.md).

The included accounts demonstrate `admin`, `controller`, and `reviewer` workspace roles. Controllers can submit project contracts and change orders, while an admin must approve them; requesters cannot approve their own submissions. Approved contract value caps milestone allocation, and milestone invoicing creates a protected invoice and audit trail atomically. Organization access is always derived from the verified server-side session; the browser cannot select a tenant through a trusted header. Demo credentials are for local portfolio use only.

## Engineering notes

- [Architecture and tradeoffs](docs/architecture.md)
- [Safe financial-data migration](docs/migration-runbook.md)
- [QuickBooks incident runbook](docs/integration-runbook.md)
- [Email delivery runbook](docs/email-delivery.md)
- [Authentication and role model](docs/authentication.md)
