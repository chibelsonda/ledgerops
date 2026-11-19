export interface SyncRun {
  id: string
  provider: string
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  recordsProcessed: number
  errorMessage: string | null
  createdAt: string
  finishedAt: string | null
  entities: Array<{
    entityType: string
    status: 'queued' | 'running' | 'succeeded' | 'failed'
    recordsProcessed: number
    errorMessage: string | null
  }>
}

export interface QuickBooksConnection {
  configured: boolean
  connected: boolean
  environment: 'sandbox'
  companyName?: string
  realmId?: string
  connectedAt?: string
  operations: {
    scheduleEnabled: boolean
    nextScheduledAt?: string
    lastSuccessfulSync?: string
    pendingWebhooks: number
    queuedWrites: number
    failedWrites: number
  }
}

export interface QuickBooksAuthorization {
  authorizationUrl: string
}
