export type CloseStatus = 'draft' | 'in_review' | 'closed'
export type CloseTaskStatus = 'pending' | 'completed'

export interface ClosePeriodSummary {
  id: string
  monthStart: string
  dueOn: string
  status: CloseStatus
  createdById: string
  createdBy: string
  submittedById?: string
  submittedBy?: string
  reviewedBy?: string
  reviewNote?: string
  totalTasks: number
  completedTasks: number
  progress: number
  openExceptions: number
  submittedAt?: string
  reviewedAt?: string
  closedAt?: string
  createdAt: string
}

export interface CloseTask {
  id: string
  title: string
  category: string
  position: number
  status: CloseTaskStatus
  evidenceNote?: string
  completedBy?: string
  completedAt?: string
}

export interface CloseAuditEvent { id: string; actor: string; action: string; metadata: Record<string, unknown>; createdAt: string }
export interface CloseDetail { period: ClosePeriodSummary; tasks: CloseTask[]; auditEvents: CloseAuditEvent[] }
export interface CreateClosePeriodInput { monthStart: string; dueOn: string }
export interface UpdateCloseTaskInput { status: CloseTaskStatus; evidenceNote: string }
export interface CloseDecisionInput { decision: 'approved' | 'rejected'; note: string }
