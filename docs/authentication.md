# Authentication and role model

## Local demo accounts

| Role | Email | Password |
| --- | --- | --- |
| Admin | `admin@ledgerops.demo` | `LedgerOpsAdmin!2026` |
| Controller | `mara@ledgerops.demo` | `LedgerOpsDemo!2026` |
| Reviewer | `reviewer@ledgerops.demo` | `LedgerOpsReview!2026` |

These credentials are seeded only for the local portfolio environment and must not be reused in a deployed system.

## Session design

- Passwords are stored with Argon2id using a unique random salt.
- Login creates independent 256-bit session and CSRF tokens.
- Only SHA-256 verifiers are persisted for both tokens.
- The opaque session token is sent only in an HttpOnly, SameSite Strict cookie.
- The CSRF token is returned in the authenticated JSON response and required in `X-CSRF-Token` for every mutation.
- Sessions expire server-side after `AUTH_SESSION_TTL` and are revoked during logout.
- Five consecutive failed passwords temporarily lock the account for fifteen minutes.
- Login failures use a generic response, and nonexistent accounts still perform a password hash calculation to reduce timing differences.

Set `AUTH_COOKIE_SECURE=true` whenever the app is served through HTTPS.

## Permissions

| Capability | Admin | Controller | Reviewer |
| --- | :---: | :---: | :---: |
| View overview and invoices | Yes | Yes | Yes |
| View and resolve reconciliation items | Yes | Yes | Yes |
| View and start synchronization | Yes | Yes | No |
| Connect or disconnect accounting providers | Yes | No | No |

All repository queries receive the organization ID resolved from the authenticated session. The removed `X-Organization-ID` client header is not accepted as identity.
