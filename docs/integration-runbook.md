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

5. Add these values to `ledgerops/.env` and restart the API:

   ```dotenv
   FRONTEND_URL=http://localhost:5173
   QUICKBOOKS_CLIENT_ID=your-development-client-id
   QUICKBOOKS_CLIENT_SECRET=your-development-client-secret
   QUICKBOOKS_REDIRECT_URL=http://localhost:8080/api/v1/integrations/quickbooks/callback
   QUICKBOOKS_API_BASE_URL=https://sandbox-quickbooks.api.intuit.com
   QUICKBOOKS_TOKEN_ENCRYPTION_KEY=your-generated-base64-key
   ```

Keep the encryption key stable. Changing it makes previously stored tokens unreadable; disconnect and reconnect the sandbox if key rotation is necessary in this local portfolio environment.

## Expected authorization flow

1. The UI requests an authorization URL from the API.
2. The API generates a cryptographically random state, stores only its SHA-256 hash for ten minutes, and returns the Intuit consent URL.
3. Intuit returns an authorization code, realm ID, and state to the backend callback.
4. The API atomically consumes the one-time state, exchanges the code server-to-server, and reads `CompanyInfo` from the sandbox Accounting API.
5. Access and refresh tokens are encrypted with AES-256-GCM before PostgreSQL persistence.
6. The callback redirects to `/sync` without leaving the authorization code in the browser URL.

Never log tokens, authorization codes, client secrets, encryption keys, or full customer payloads.

## Expected synchronization flow

1. Validate and acknowledge a webhook quickly.
2. Insert a job with an idempotency key.
3. Refresh OAuth access if needed.
4. Fetch changed entities with pagination.
5. Upsert source records in bounded transactions.
6. Run a periodic Change Data Capture reconciliation to recover missed webhook events.

## When a sync fails

1. Find the failed `sync_runs` record and correlation ID.
2. Check the provider status page and classify the response: authorization, rate limit, validation, or transient upstream error.
3. Inspect redacted response metadata only.
4. Reproduce against the sandbox with the same entity type and cursor.
5. Fix the adapter or data mapping, then retry the existing idempotent job.
6. Compare entity counts and financial totals after recovery.

## Retry policy

Transient network and `5xx` failures retry with bounded exponential backoff and jitter. Rate limits honor `Retry-After`. Authentication failures attempt one token refresh. Validation and permission failures stop and surface an actionable message.
