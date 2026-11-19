import {
  defineRailway,
  github,
  postgres,
  preserve,
  project,
  service,
} from 'railway/iac'

const repository = 'chibelsonda/ledgerops'

export default defineRailway(() => {
  const database = postgres('Postgres')

  const api = service('ledgerops', {
    source: github(repository, {
      branch: 'main',
      rootDirectory: '/backend',
    }),
    healthcheck: '/healthz',
    healthcheckTimeout: 120,
    env: {
      ALLOWED_ORIGIN: preserve(),
      APP_ENV: preserve(),
      AUTH_COOKIE_SECURE: preserve(),
      AUTH_SESSION_IDLE_TTL: preserve(),
      AUTH_SESSION_TTL: preserve(),
      DATABASE_URL: database.env.DATABASE_URL,
      EMAIL_DELIVERY_POLL_INTERVAL: preserve(),
      FRONTEND_URL: preserve(),
      LOGIN_RATE_LIMIT_REQUESTS: preserve(),
      LOGIN_RATE_LIMIT_WINDOW: preserve(),
      QUICKBOOKS_API_BASE_URL: preserve(),
      QUICKBOOKS_CLIENT_ID: preserve(),
      QUICKBOOKS_CLIENT_SECRET: preserve(),
      QUICKBOOKS_OUTBOX_POLL_INTERVAL: preserve(),
      QUICKBOOKS_REDIRECT_URL: preserve(),
      QUICKBOOKS_SCHEDULE_POLL_INTERVAL: preserve(),
      QUICKBOOKS_TOKEN_ENCRYPTION_KEY: preserve(),
      RATE_LIMIT_REQUESTS: preserve(),
      RATE_LIMIT_WINDOW: preserve(),
      REMINDER_POLL_INTERVAL: preserve(),
      SEED_DEMO: preserve(),
      SENSITIVE_RATE_LIMIT_REQUESTS: preserve(),
      SENSITIVE_RATE_LIMIT_WINDOW: preserve(),
      WORKER_POLL_INTERVAL: preserve(),
    },
  })

  const worker = service('worker', {
    source: github(repository, {
      branch: 'main',
      rootDirectory: '/backend',
    }),
    start: '/app/worker',
    env: {
      ALLOWED_ORIGIN: api.env.ALLOWED_ORIGIN,
      APP_ENV: api.env.APP_ENV,
      AUTH_COOKIE_SECURE: api.env.AUTH_COOKIE_SECURE,
      AUTH_SESSION_IDLE_TTL: api.env.AUTH_SESSION_IDLE_TTL,
      AUTH_SESSION_TTL: api.env.AUTH_SESSION_TTL,
      DATABASE_URL: api.env.DATABASE_URL,
      EMAIL_DELIVERY_POLL_INTERVAL: api.env.EMAIL_DELIVERY_POLL_INTERVAL,
      FRONTEND_URL: api.env.FRONTEND_URL,
      LOGIN_RATE_LIMIT_REQUESTS: api.env.LOGIN_RATE_LIMIT_REQUESTS,
      LOGIN_RATE_LIMIT_WINDOW: api.env.LOGIN_RATE_LIMIT_WINDOW,
      QUICKBOOKS_API_BASE_URL: api.env.QUICKBOOKS_API_BASE_URL,
      QUICKBOOKS_CLIENT_ID: api.env.QUICKBOOKS_CLIENT_ID,
      QUICKBOOKS_CLIENT_SECRET: api.env.QUICKBOOKS_CLIENT_SECRET,
      QUICKBOOKS_OUTBOX_POLL_INTERVAL: api.env.QUICKBOOKS_OUTBOX_POLL_INTERVAL,
      QUICKBOOKS_REDIRECT_URL: api.env.QUICKBOOKS_REDIRECT_URL,
      QUICKBOOKS_SCHEDULE_POLL_INTERVAL: api.env.QUICKBOOKS_SCHEDULE_POLL_INTERVAL,
      QUICKBOOKS_TOKEN_ENCRYPTION_KEY: api.env.QUICKBOOKS_TOKEN_ENCRYPTION_KEY,
      RATE_LIMIT_REQUESTS: api.env.RATE_LIMIT_REQUESTS,
      RATE_LIMIT_WINDOW: api.env.RATE_LIMIT_WINDOW,
      REMINDER_POLL_INTERVAL: api.env.REMINDER_POLL_INTERVAL,
      SEED_DEMO: api.env.SEED_DEMO,
      SENSITIVE_RATE_LIMIT_REQUESTS: api.env.SENSITIVE_RATE_LIMIT_REQUESTS,
      SENSITIVE_RATE_LIMIT_WINDOW: api.env.SENSITIVE_RATE_LIMIT_WINDOW,
      WORKER_POLL_INTERVAL: api.env.WORKER_POLL_INTERVAL,
    },
  })

  const frontend = service('frontend', {
    source: github(repository, {
      branch: 'main',
      rootDirectory: '/frontend',
    }),
    healthcheck: '/healthz',
    healthcheckTimeout: 120,
    env: {
      API_HOST: api.env.RAILWAY_PRIVATE_DOMAIN,
      API_PORT: '8080',
    },
  })

  return project('ledgerops', {
    resources: [database, api, worker, frontend],
  })
})
