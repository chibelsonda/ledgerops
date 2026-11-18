# Security and production hardening

LedgerOps applies defense in depth at the API and browser boundary. `APP_ENV=production` enables fail-closed configuration validation: demo seeding must be disabled, the session cookie must be secure, public origins must use HTTPS, and the database URL cannot use the bundled demo credentials or `sslmode=disable`.

## Required production settings

Start from `.env.example`, keep secrets outside source control, and set at least:

```text
APP_ENV=production
SEED_DEMO=false
AUTH_COOKIE_SECURE=true
ALLOWED_ORIGIN=https://ledgerops.example
FRONTEND_URL=https://ledgerops.example
DATABASE_URL=postgres://runtime-user:...@database/ledgerops?sslmode=verify-full
MIGRATION_DATABASE_URL=postgres://migration-user:...@database/ledgerops?sslmode=verify-full
```

Use a dedicated, least-privileged runtime PostgreSQL role and grant it only the schema/table/sequence access required by the application. `MIGRATION_DATABASE_URL` is used only while applying embedded migrations; production mode refuses to reuse the runtime connection for DDL. Rotate database, SMTP, OAuth, webhook, and token-encryption secrets through the deployment secret manager.

## Request and session controls

- The API enforces a general per-IP limit, an authenticated per-user limit, and stricter limits for login, OAuth callbacks, and email webhooks. Tune the `RATE_LIMIT_*`, `LOGIN_RATE_LIMIT_*`, and `SENSITIVE_RATE_LIMIT_*` variables for the deployment.
- Only configure `TRUSTED_PROXY_CIDRS` for proxies you control. Forwarded client IP headers are ignored from every other peer.
- The built-in limiter is bounded and process-local. Multi-replica deployments must also enforce a distributed limit at the trusted ingress or API gateway.
- Sessions have absolute and idle expiration, opaque HttpOnly/Secure/SameSite cookies, server-side revocation, and CSRF verification for state changes.
- Authentication failures, CSRF failures, authorization denials, and rate-limit events are emitted as structured security logs. Route these logs to alerting with sensitive-field redaction.

## Browser and dependency controls

The Nginx deployment adds CSP, anti-framing, MIME-sniffing, referrer, permissions, cross-origin, and HSTS headers. API responses are marked `Cache-Control: no-store`. CI runs race tests, static analysis, Go vulnerability scanning, production npm auditing, and CodeQL; Dependabot monitors Go, npm, and GitHub Actions dependencies.

The local demo accounts and known passwords are never rendered by the production frontend and must never be enabled in a deployed environment.

## Tracked upstream exception

As of 2026-11-18, Vite and Vitest depend on `picomatch@4.0.7`, which has a development-tooling ReDoS advisory but no patched npm release ([GHSA-fc5q-39xc-vjpf](https://github.com/advisories/GHSA-fc5q-39xc-vjpf)). Node and these development dependencies are not present in the production web runtime, the development server binds only to loopback, and CI separately requires a clean production-dependency audit. Dependabot is enabled so the exception can be removed as soon as an upstream patch is published.
