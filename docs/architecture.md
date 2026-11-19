# Architecture

LedgerOps is a deliberately small modular monolith. The API and worker are separate processes built from the same Go module; PostgreSQL is the source of truth; React owns presentation and interaction only.

```text
React + TypeScript
       │ JSON/HTTP + HttpOnly session
       ▼
Go Handler → Service → Repository/GORM ────────► PostgreSQL
                            ▲
                            │ FOR UPDATE SKIP LOCKED
                       Go sync worker
                            │
                            ▼
                 QuickBooks adapter boundary
```

## Design choices

- **Money is stored as integer minor units.** Authoritative totals are calculated in Go or PostgreSQL, never with browser floating-point arithmetic.
- **Tenant scope is explicit.** Every financial and operational query requires an organization ID.
- **Tenant identity is server-derived.** Opaque cookie sessions resolve a user, organization, and role; API clients cannot assert tenant identity through request headers.
- **Mutations require CSRF proof.** A session-bound synchronizer token is returned in JSON and sent back through `X-CSRF-Token`.
- **Public identifiers use UUIDv7.** API resources keep one opaque identifier while benefiting from time-ordered B-tree insertion locality. Internal queue and audit identifiers remain implementation details and can evolve independently.
- **Sync work is asynchronous.** The API creates a run and an idempotent queue record. The worker claims jobs with `FOR UPDATE SKIP LOCKED`, refreshes OAuth credentials when required, imports paginated entities in dependency order, records attempts, and applies classified retry backoff.
- **Collection reminders use a transactional outbox.** The worker evaluates client policies and primary contacts, then idempotently queues one immutable reminder per overdue invoice. A separate SMTP adapter claims due messages with `FOR UPDATE SKIP LOCKED`, applies bounded quadratic retry backoff, and only records `sent` after provider acceptance. Without SMTP configuration, messages remain safely queued.
- **Delivery callbacks are append-only.** A shared-secret webhook accepts normalized delivered, bounced, complained, and deferred events. Provider event IDs are unique, making retries idempotent without mutating the immutable reminder record.
- **External IDs are unique per organization and source.** Retrying an import cannot create a duplicate invoice or payment.
- **Review actions are transactional.** Resolving an exception and writing its audit event either both succeed or both roll back.
- **Project billing uses controlled state transitions.** Contract and change-order decisions enforce separation of duties. Approved value caps milestone allocations, and milestone invoicing creates the invoice, advances the milestone, and appends both audit records in one transaction.
- **Financial approvals are immutable.** PostgreSQL triggers prevent approved or rejected contracts and change orders from being rewritten or deleted. Invoiced milestones cannot return to a planned state, and their generated invoices cannot be edited through the general invoice endpoint.
- **Month-end close is a controlled state machine.** Controllers create close periods, operators attach evidence to a generated checklist, and an independent admin either returns the period to draft or permanently closes it. Closed periods and their task evidence are protected by PostgreSQL triggers.
- **Cash forecasting is explainable.** Each 13-week scenario stores explicit collection, timing, expense-growth, and project-probability assumptions. The repository combines those assumptions with open invoice balances, approved project milestones, historical operating run-rate, and append-only manual adjustments.
- **GORM does not hide financial SQL.** GORM owns connection pooling, scoped writes, result mapping, and transactions. Reporting CTEs and queue locking remain explicit through GORM's raw-query API so they can be reviewed and explained with `EXPLAIN (ANALYZE, BUFFERS)`.

## SOLID, DRY, and KISS

- Each API feature owns its model, HTTP handler, application service, narrow repository interface, and GORM-backed PostgreSQL repository.
- Each UI feature owns its API client, types, validation schema, and page; only true cross-feature behavior lives under `shared`.
- HTTP handlers translate HTTP concerns and depend only on feature-level service interfaces.
- Services normalize inputs, enforce use-case policy, and depend only on the repository capabilities they consume.
- Repositories own PostgreSQL queries, row locking, and atomic transactions; concurrent financial invariants are rechecked inside those transactions.
- Project billing locks the approved contract row while allocating change orders and milestones, so concurrent requests cannot oversubscribe the authorized value.
- Close submission and approval lock the period row and recheck checklist completion and separation of duties inside the transaction.
- `internal/app/router.go` is the composition root: it constructs concrete repositories, injects them into services, and injects services into handlers.
- The background worker also depends on a narrow queue repository interface; only application composition creates the concrete GORM repository.
- Configuration, migrations, financial rules, transport, data access, and background processing have focused packages.
- Shared JSON, error, money, date, and status formatting has one implementation.
- Shared controls wrap Headless UI once and apply a consistent Tailwind design system.
- Zod schemas are the client-side source of truth for form validation; the API independently enforces the same invariants.
- The worker uses PostgreSQL instead of adding a second queue service before scale requires one.
- Reminder generation and sync processing share the worker process but use independent services and polling intervals.
- The QuickBooks feature isolates authorization, provider HTTP calls, entity mapping, encrypted persistence, CDC, webhook ingestion, and outbound delivery behind narrow interfaces without changing queue semantics.
- A PostgreSQL transactional outbox couples local client, invoice, and payment commits to eventual QuickBooks delivery; AP entities are inbound-only.

## API request flow

```text
net/http route
    │
    ▼
Handler             decode/encode JSON, path/query parsing, HTTP status mapping
    │ service interface
    ▼
Service             normalization, validation, authorization-independent use-case rules
    │ repository interface
    ▼
PostgresRepository  SQL, GORM mapping, transactions, locks, persistence constraints
    │
    ▼
PostgreSQL
```

Interfaces are defined at the consuming boundary. Constructors receive dependencies explicitly; there is no service locator, global database handle, or reflection-based dependency-injection framework. Read-only features keep intentionally thin services so the dependency direction remains consistent without inventing domain abstractions.

## Production evolution

The portfolio demo uses seeded identities alongside optional live sandbox synchronization. Passwords use Argon2id, session verifiers are hashed, cookies are HttpOnly and SameSite Strict, and mutations require a CSRF token. Production configuration requires Secure cookies behind HTTPS, AWS KMS for OAuth tokens, and the Intuit webhook verifier; infrastructure should additionally restrict the metrics and webhook ingress paths and can add PostgreSQL row-level security as defense in depth.
