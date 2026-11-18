# Email delivery runbook

LedgerOps keeps reminder generation separate from delivery. The scheduler creates immutable outbox records; the delivery worker claims and sends them only when SMTP is configured.

## Configuration

Set these values in `ledgerops/.env`, then restart both the API and worker:

```dotenv
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
SMTP_REQUIRE_TLS=true
EMAIL_FROM_ADDRESS=billing@example.com
EMAIL_FROM_NAME=LedgerOps
EMAIL_DELIVERY_POLL_INTERVAL=5s
EMAIL_WEBHOOK_SECRET=replace-with-a-long-random-value
```

Username and password may remain empty for a trusted local relay. STARTTLS with TLS 1.2 or newer is required by default; set `SMTP_REQUIRE_TLS=false` only for a trusted local development relay. Leaving `SMTP_HOST` or `EMAIL_FROM_ADDRESS` empty disables delivery; queued records remain untouched. Never commit real credentials.

## Retry behavior

Workers claim one due reminder at a time with `FOR UPDATE SKIP LOCKED`. A failed send is retried after a quadratic delay (`attempt²` minutes). After five failed attempts the reminder becomes `failed`. Message content and all terminal reminder records are immutable.

Each reminder keeps the same RFC Message-ID across retries. Delivery remains at-least-once because SMTP acceptance and PostgreSQL cannot share a transaction, but the stable ID gives relays and recipients a consistent deduplication key.

## Delivery webhooks

Email services use different callback formats. A trusted relay should verify the provider-specific signature, normalize the event, and call:

```http
POST /api/v1/integrations/email/webhook
X-LedgerOps-Webhook-Secret: <EMAIL_WEBHOOK_SECRET>
Content-Type: application/json
```

```json
{
  "eventId": "provider-event-unique-id",
  "messageId": "<message-id-returned-by-ledgerops>",
  "event": "delivered",
  "occurredAt": "2026-11-18T13:00:00Z",
  "payload": {}
}
```

Allowed events are `delivered`, `bounced`, `complained`, and `deferred`. Event IDs are unique, so provider retries are idempotent. The optional payload is retained for diagnostics; do not place secrets or unnecessary personal data in it.

## Troubleshooting

- A queued reminder with zero attempts means SMTP delivery is disabled or the worker is not running.
- `lastError` and the attempt counter appear in Collections after a send failure.
- A failed reminder exhausted its retry budget and requires investigation; it is not silently re-sent.
- A `401` webhook response means the shared secret differs. A `404` means the provider message ID is unknown.
