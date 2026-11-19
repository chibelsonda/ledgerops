# QuickBooks sandbox integration runbook

## Configure OAuth locally

1. Create or open an app in the Intuit Developer Portal and select the QuickBooks Online Accounting scope.
2. Under **Development → Keys and credentials**, copy the sandbox Client ID and Client Secret.
3. Register this redirect URI exactly:

   ```text
   http://localhost:8080/api/v1/integrations/quickbooks/callback
   ```

4. Generate a stable 32-byte encryption key once:

   ```bash
   openssl rand -base64 32
   ```

5. Add these values to `ledgerops/.env` and restart both the API and worker:

   ```dotenv
   FRONTEND_URL=http://localhost:5173
   QUICKBOOKS_CLIENT_ID=your-development-client-id
   QUICKBOOKS_CLIENT_SECRET=your-development-client-secret
   QUICKBOOKS_REDIRECT_URL=http://localhost:8080/api/v1/integrations/quickbooks/callback
   QUICKBOOKS_API_BASE_URL=https://sandbox-quickbooks.api.intuit.com
   QUICKBOOKS_TOKEN_ENCRYPTION_KEY=your-generated-base64-key
   ```

Keep the encryption key stable. Changing it makes previously stored tokens unreadable; disconnect and reconnect the sandbox if key rotation is necessary in this local portfolio environment.

For webhooks, copy the verifier token from the Intuit app and set `QUICKBOOKS_WEBHOOK_VERIFIER`. Register the public HTTPS endpoint `https://your-api.example/api/v1/integrations/quickbooks/webhook`. Localhost cannot receive Intuit webhooks without an HTTPS tunnel.

Production does not accept the local AES key. Grant the API and worker `kms:Encrypt` and `kms:Decrypt` on a dedicated symmetric KMS key, then configure:

```dotenv
QUICKBOOKS_KMS_KEY_ID=alias/ledgerops-quickbooks
QUICKBOOKS_KMS_REGION=ap-southeast-1
QUICKBOOKS_WEBHOOK_VERIFIER=your-intuit-verifier-token
QUICKBOOKS_SCHEDULE_POLL_INTERVAL=1m
QUICKBOOKS_OUTBOX_POLL_INTERVAL=2s
```

Switching an existing local connection from AES to KMS requires disconnecting and reconnecting so Intuit issues tokens encrypted by KMS.

## Expected authorization flow

1. The UI requests an authorization URL from the API.
2. The API generates a cryptographically random state, stores only its SHA-256 hash for ten minutes, and returns the Intuit consent URL.
3. Intuit returns an authorization code, realm ID, and state to the backend callback.
4. The API atomically consumes the one-time state, exchanges the code server-to-server, and reads `CompanyInfo` from the sandbox Accounting API.
5. Access and refresh tokens are encrypted with AES-256-GCM locally or AWS KMS in production before PostgreSQL persistence.
6. The callback redirects to `/sync` without leaving the authorization code in the browser URL.

Never log tokens, authorization codes, client secrets, encryption keys, or full customer payloads.

## Expected synchronization flow

1. A manual request, an hourly schedule, or a verified webhook queues one idempotent PostgreSQL job.
2. A worker claims it with `FOR UPDATE SKIP LOCKED`, refreshes OAuth when needed, and persists both rotated tokens.
3. The first run imports Accounts, Vendors, Customers, Invoices, Payments, Purchases, Bills, Bill Payments, and Credit Memos with pagination.
4. Later runs call Intuit Change Data Capture from the earliest per-entity cursor. A full import is forced when any cursor is absent or older than 29 days.
5. Deleted and voided remote records are retained in the generic mirror and removed from active LedgerOps financial projections.
6. Each run records status and counts per entity. Webhook rows are marked processed only after the covering CDC run succeeds.
7. Local client, invoice, and payment mutations write an outbox row in the same transaction. The worker creates or sparsely updates the corresponding QuickBooks record with bounded retries. Bills and other accounts-payable entities are intentionally inbound-only.

## When a sync fails

1. Find the failed `sync_runs`, `sync_run_entities`, or `quickbooks_outbox` record and correlation ID.
2. Check the provider status page and classify the response: authorization, rate limit, validation, or transient upstream error.
3. Inspect redacted response metadata only.
4. Reproduce against the sandbox with the same entity type and cursor.
5. Fix the adapter or data mapping, then retry the existing idempotent job.
6. Compare `quickbooks_entities` counts, deletion markers, and financial totals after recovery.

Disconnect revokes the refresh token at Intuit before local credentials, schedules, and pending writes are removed. If Intuit is unavailable, LedgerOps leaves the local connection intact so revocation can be retried.

## Retry policy

Transient network and `5xx` failures retry with bounded quadratic backoff. Rate limits honor `Retry-After`. A `401` triggers one token refresh and one replay of the current page. Validation, revoked authorization, and permission failures stop and surface an actionable message.
