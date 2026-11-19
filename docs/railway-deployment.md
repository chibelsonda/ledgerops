# Railway deployment

LedgerOps runs on Railway as four resources in one project:

- `frontend`: public Vite application served by Nginx
- `ledgerops`: private Go HTTP API service
- `worker`: private Go background worker
- `postgres`: private PostgreSQL database

The browser uses same-origin `/api/v1` requests. Nginx forwards those requests to the API over Railway private networking, so the API does not need its own public domain.

## Apply the infrastructure

The current Railway Infrastructure as Code definition is in `.railway/railway.ts`. Install and authenticate the Railway CLI, then link the existing Railway project. Existing API variables are preserved and referenced by the worker; secret values never appear in the definition.

Preview the change before applying it:

```bash
railway link
railway config plan
railway config apply
```

The configuration adopts the existing `ledgerops` and `Postgres` resources, creates `worker` and `frontend`, connects both Go services to PostgreSQL, configures `/healthz` checks, and sends frontend API traffic to the API's `railway.internal` hostname.

After the first deployment, generate a public domain for `frontend`. Set the API's `ALLOWED_ORIGIN` and `FRONTEND_URL` to that exact HTTPS origin, then redeploy the API and worker. Do not generate public domains for the worker or PostgreSQL.

## Runtime variables

Set sensitive values in Railway rather than committing `.env`. Both `api` and `worker` need the same application and integration configuration unless noted otherwise.

For a temporary private preview:

```dotenv
APP_ENV=development
SEED_DEMO=true
AUTH_COOKIE_SECURE=true
```

Never expose the seeded demo identities together with real QuickBooks, SMTP, or customer credentials. A public deployment should use `APP_ENV=production`, `SEED_DEMO=false`, HTTPS origins, SMTP, managed QuickBooks token encryption, and separate runtime and migration database roles as required by `docs/security.md`.

QuickBooks URLs should use the frontend domain because Nginx owns the public ingress:

```dotenv
QUICKBOOKS_REDIRECT_URL=https://your-frontend-domain.up.railway.app/api/v1/integrations/quickbooks/callback
```

Register the webhook endpoint in Intuit as:

```text
https://your-frontend-domain.up.railway.app/api/v1/integrations/quickbooks/webhook
```

Set these integration secrets on both Go services when QuickBooks is enabled:

```text
QUICKBOOKS_CLIENT_ID
QUICKBOOKS_CLIENT_SECRET
QUICKBOOKS_KMS_KEY_ID
QUICKBOOKS_KMS_REGION
QUICKBOOKS_WEBHOOK_VERIFIER
```

The API also requires SMTP and email webhook settings in production. See `docs/security.md`, `docs/integration-runbook.md`, and `docs/email-delivery.md` for the complete production variable set.

## Manual dashboard equivalent

If Infrastructure as Code is not used, connect the same GitHub repository to three Railway services:

| Service | Root directory | Start command | Public domain |
| --- | --- | --- | --- |
| `frontend` | `/frontend` | Dockerfile default | Yes |
| `ledgerops` | `/backend` | Dockerfile default (`/app/api`) | No |
| `worker` | `/backend` | `/app/worker` | No |

Set `API_HOST=${{ledgerops.RAILWAY_PRIVATE_DOMAIN}}` and `API_PORT=8080` on `frontend`. Set `DATABASE_URL=${{Postgres.DATABASE_URL}}` on the API and worker. The production database-role separation described in `docs/security.md` still applies.
