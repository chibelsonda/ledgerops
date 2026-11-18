export interface ReconciliationItem {
  id: string
  kind: string
  severity: 'low' | 'medium' | 'high'
  title: string
  description: string
  amountMinor: number | null
  currencyCode: string | null
  status: 'open' | 'resolved' | 'dismissed'
  createdAt: string
}

