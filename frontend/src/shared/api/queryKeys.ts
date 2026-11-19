export const queryKeys = {
  dashboard: ['dashboard'] as const,
  invoices: {
    all: ['invoices'] as const,
    list: (filters: object) => ['invoices', filters] as const,
    clients: ['invoices', 'clients'] as const,
    audit: (invoiceId?: string) => ['invoices', invoiceId, 'audit-events'] as const,
    payments: (invoiceId: string) => ['invoices', invoiceId, 'payments'] as const,
    adjustments: (invoiceId: string) => ['invoices', invoiceId, 'adjustments'] as const,
  },
  clients: {
    all: ['clients'] as const,
    list: (query: string, page: number) => ['clients', query, page] as const,
    contacts: (clientId?: string) => ['clients', clientId, 'contacts'] as const,
  },
  projects: {
    all: ['projects'] as const,
    clients: ['projects', 'clients'] as const,
    detail: (projectId?: string) => ['projects', projectId] as const,
  },
  close: {
    all: ['close-periods'] as const,
    detail: (periodId?: string) => ['close-periods', periodId] as const,
  },
  forecast: {
    scenarios: ['cash-flow-scenarios'] as const,
    detail: (scenarioId: string) => ['cash-flow-forecast', scenarioId] as const,
  },
  reminders: {
    all: ['reminders'] as const,
    deliveryStatus: ['reminders', 'delivery-status'] as const,
  },
  reconciliation: ['reconciliation'] as const,
  syncRuns: ['sync-runs'] as const,
  quickBooks: ['integrations', 'quickbooks'] as const,
} as const
