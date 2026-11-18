export interface SyncRun {
  id: string
  provider: string
  status: 'queued' | 'running' | 'succeeded' | 'failed'
  recordsProcessed: number
  errorMessage: string | null
  createdAt: string
  finishedAt: string | null
}

export interface QuickBooksConnection {
  configured: boolean
  connected: boolean
  environment: 'sandbox'
  companyName?: string
  realmId?: string
  connectedAt?: string
}

export interface QuickBooksAuthorization {
  authorizationUrl: string
}
