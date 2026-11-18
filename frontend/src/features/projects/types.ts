export type ApprovalStatus = 'pending' | 'approved' | 'rejected'

export interface ClientOption { id: string; name: string }

export interface ProjectSummary {
  id: string
  projectCode: string
  name: string
  clientId: string
  clientName: string
  status: 'active' | 'on_hold' | 'completed' | 'cancelled'
  currencyCode: string
  startOn: string
  endOn?: string
  contractId: string
  contractNumber: string
  contractStatus: ApprovalStatus
  originalValueMinor: number
  approvedChangesMinor: number
  revisedValueMinor: number
  allocatedMinor: number
  billedMinor: number
  unallocatedMinor: number
  createdAt: string
}

export interface Contract {
  id: string
  contractNumber: string
  title: string
  originalValueMinor: number
  status: ApprovalStatus
  requestedById: string
  requestedBy: string
  reviewedBy?: string
  reviewNote?: string
  requestedAt: string
  reviewedAt?: string
}

export interface ChangeOrder {
  id: string
  changeOrderNumber: string
  title: string
  description: string
  amountMinor: number
  status: ApprovalStatus
  requestedById: string
  requestedBy: string
  reviewedBy?: string
  reviewNote?: string
  requestedAt: string
  reviewedAt?: string
}

export interface BillingMilestone {
  id: string
  name: string
  amountMinor: number
  dueOn: string
  status: 'planned' | 'invoiced'
  invoiceId?: string
  invoiceNumber?: string
  createdBy: string
  createdAt: string
  invoicedAt?: string
}

export interface ProjectAuditEvent {
  id: string
  actor: string
  action: string
  metadata: Record<string, unknown>
  createdAt: string
}

export interface ProjectDetail {
  project: ProjectSummary
  contract: Contract
  changeOrders: ChangeOrder[]
  milestones: BillingMilestone[]
  auditEvents: ProjectAuditEvent[]
}

export interface CreateProjectInput {
  clientId: string
  projectCode: string
  name: string
  currencyCode: string
  startOn: string
  endOn?: string
  contractNumber: string
  contractTitle: string
  originalValueMinor: number
}

export interface DecisionInput { decision: 'approved' | 'rejected'; note: string }
export interface ChangeOrderInput { changeOrderNumber: string; title: string; description: string; amountMinor: number }
export interface MilestoneInput { name: string; amountMinor: number; dueOn: string }
export interface MilestoneInvoiceInput { invoiceNumber: string; issuedOn: string; dueOn: string }
