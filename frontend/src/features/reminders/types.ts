export type ReminderStatus = 'queued' | 'sent' | 'cancelled' | 'failed'
export type DeliveryEvent = 'delivered' | 'bounced' | 'complained' | 'deferred'

export interface Reminder {
  id: string
  invoiceId: string
  invoiceNumber: string
  clientId: string
  clientName: string
  contactName: string
  recipientEmail: string
  status: ReminderStatus
  scheduledFor: string
  subject: string
  message: string
  outstandingMinor: number
  currencyCode: string
  createdAt: string
  attempts: number
  maxAttempts: number
  lastError: string | null
  providerMessageId: string | null
  deliveryEvent: DeliveryEvent | null
}

export interface DeliveryStatus {
  configured: boolean
  fromAddress?: string
}
