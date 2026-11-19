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
- Sessions expire server-side after `AUTH_SESSION_TTL`, expire after `AUTH_SESSION_IDLE_TTL` without activity, and are revoked during logout.
- Five consecutive failed passwords temporarily lock the account for fifteen minutes.
- Login failures use a generic response, and nonexistent accounts still perform a password hash calculation to reduce timing differences.
- Login is rate-limited per client IP, and authenticated API traffic is limited both per IP and per user.

## Account creation and recovery

- Public registration creates a new organization and assigns the first user the `admin` role.
- Registration, login, reset requests, and reset confirmations share the stricter authentication rate limit.
- Password reset responses do not reveal whether an email address exists.
- Reset tokens contain 256 bits of randomness. Only their SHA-256 hashes are stored, and each token expires after 30 minutes and can be used once.
- Completing a password reset revokes every existing session for that user and clears any login lockout.
- Development responses include a local reset link so the flow works without SMTP. Production requires SMTP and never includes the token in the API response.

Set `AUTH_COOKIE_SECURE=true` whenever the app is served through HTTPS. Production mode refuses to start without it.

## Permissions

| Capability | Admin | Controller | Reviewer |
| --- | :---: | :---: | :---: |
| View overview and invoices | Yes | Yes | Yes |
| View and resolve reconciliation items | Yes | Yes | Yes |
| View and start synchronization | Yes | Yes | No |
| Connect or disconnect accounting providers | Yes | No | No |

All repository queries receive the organization ID resolved from the authenticated session. The removed `X-Organization-ID` client header is not accepted as identity.
