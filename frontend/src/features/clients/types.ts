export interface Client {
  id: string; name: string; paymentTermsDays: number; remindersEnabled: boolean; reminderDelayDays: number
  primaryContact: string; primaryEmail: string; contactCount: number; invoiceCount: number; outstandingMinor: number
  currencyCode: string; createdAt: string; updatedAt: string
}
export interface Contact { id: string; clientId: string; name: string; email: string; phone: string; roleTitle: string; isPrimary: boolean; createdAt: string; updatedAt: string }
export interface ClientPage { items: Client[]; pagination: { page: number; pageSize: number; totalItems: number; totalPages: number } }
export interface ClientInput { name: string; paymentTermsDays: number; remindersEnabled: boolean; reminderDelayDays: number; auditNote?: string; primaryContact?: { name: string; email: string; phone: string; roleTitle: string } }
export interface ContactInput { name: string; email: string; phone: string; roleTitle: string; isPrimary: boolean }
