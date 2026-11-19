import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowDownUp, BadgeCheck, CalendarClock, FilePlus2, History, Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useAuth } from '@/features/auth/AuthProvider'
import { capabilities } from '@/shared/lib/capabilities'
import { queryKeys } from '@/shared/api/queryKeys'
import { formatDate, formatMoney, formatRelativeTime, titleCase } from '@/shared/lib/format'
import { Button } from '@/shared/ui/Button'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ErrorState, LoadingState } from '@/shared/ui/ViewState'
import { SlideOver } from '@/shared/ui/SlideOver'
import { ChangeOrderForm, DecisionDialog, MilestoneForm, MilestoneInvoiceDialog } from './ProjectWorkflowForms'
import { projectsApi } from '@/features/projects/api'
import { changeOrderFormSchema, decisionFormSchema, invoiceMilestoneFormSchema, milestoneFormSchema, toMinor, type ChangeOrderFormValues, type DecisionFormValues, type InvoiceMilestoneFormValues, type MilestoneFormValues } from '@/features/projects/schema'
import type { BillingMilestone, DecisionInput, ProjectDetail } from '@/features/projects/types'

type DecisionTarget = { kind: 'contract' | 'changeOrder'; id: string; label: string } | null
const today = () => new Date().toISOString().slice(0, 10)

export function ProjectDetailDialog({ projectId, onClose }: { projectId?: string; onClose: () => void }) {
  const { session } = useAuth()
  const queryClient = useQueryClient()
  const [showChangeOrder, setShowChangeOrder] = useState(false)
  const [showMilestone, setShowMilestone] = useState(false)
  const [decisionTarget, setDecisionTarget] = useState<DecisionTarget>(null)
  const [invoiceTarget, setInvoiceTarget] = useState<BillingMilestone | null>(null)
  const detail = useQuery({ queryKey: queryKeys.projects.detail(projectId), queryFn: () => projectsApi.get(projectId!), enabled: Boolean(projectId) })
  const changeForm = useForm<ChangeOrderFormValues>({ resolver: zodResolver(changeOrderFormSchema), mode: 'onChange', defaultValues: { changeOrderNumber: '', title: '', description: '', amount: 0 } })
  const milestoneForm = useForm<MilestoneFormValues>({ resolver: zodResolver(milestoneFormSchema), mode: 'onChange', defaultValues: { name: '', amount: 0, dueOn: today() } })
  const decisionForm = useForm<DecisionFormValues>({ resolver: zodResolver(decisionFormSchema), mode: 'onChange', defaultValues: { note: '' } })
  const invoiceForm = useForm<InvoiceMilestoneFormValues>({ resolver: zodResolver(invoiceMilestoneFormSchema), mode: 'onChange', defaultValues: { invoiceNumber: '', issuedOn: today(), dueOn: today() } })

  useEffect(() => {
    setShowChangeOrder(false); setShowMilestone(false); setDecisionTarget(null); setInvoiceTarget(null)
  }, [projectId])

  const acceptDetail = async (value: ProjectDetail) => {
    queryClient.setQueryData(queryKeys.projects.detail(projectId), value)
    await Promise.all([queryClient.invalidateQueries({ queryKey: queryKeys.projects.all }), queryClient.invalidateQueries({ queryKey: queryKeys.invoices.all })])
  }
  const decide = useMutation({
    mutationFn: ({ target, input }: { target: NonNullable<DecisionTarget>; input: DecisionInput }) => target.kind === 'contract' ? projectsApi.decideContract(target.id, input) : projectsApi.decideChangeOrder(target.id, input),
    onSuccess: async (value) => { await acceptDetail(value); setDecisionTarget(null); decisionForm.reset({ note: '' }) },
  })
  const createChangeOrder = useMutation({
    mutationFn: (values: ChangeOrderFormValues) => projectsApi.createChangeOrder(detail.data!.contract.id, { changeOrderNumber: values.changeOrderNumber.trim(), title: values.title.trim(), description: values.description.trim(), amountMinor: toMinor(values.amount) }),
    onSuccess: async (value) => { await acceptDetail(value); setShowChangeOrder(false); changeForm.reset({ changeOrderNumber: '', title: '', description: '', amount: 0 }) },
  })
  const createMilestone = useMutation({
    mutationFn: (values: MilestoneFormValues) => projectsApi.createMilestone(detail.data!.contract.id, { name: values.name.trim(), amountMinor: toMinor(values.amount), dueOn: values.dueOn }),
    onSuccess: async (value) => { await acceptDetail(value); setShowMilestone(false); milestoneForm.reset({ name: '', amount: 0, dueOn: today() }) },
  })
  const invoiceMilestone = useMutation({
    mutationFn: (values: InvoiceMilestoneFormValues) => projectsApi.invoiceMilestone(invoiceTarget!.id, { invoiceNumber: values.invoiceNumber.trim(), issuedOn: values.issuedOn, dueOn: values.dueOn }),
    onSuccess: async (value) => { await acceptDetail(value); setInvoiceTarget(null); invoiceForm.reset({ invoiceNumber: '', issuedOn: today(), dueOn: today() }) },
  })

  const beginDecision = (target: NonNullable<DecisionTarget>) => { decisionForm.reset({ note: '' }); setDecisionTarget(target) }
  const beginInvoice = (milestone: BillingMilestone) => {
    invoiceForm.reset({ invoiceNumber: '', issuedOn: today(), dueOn: milestone.dueOn >= today() ? milestone.dueOn : today() })
    setInvoiceTarget(milestone)
  }

  const canEdit = capabilities.operate(session?.user.role)
  const canApprove = capabilities.administer(session?.user.role)
  const data = detail.data
  const project = data?.project
  const contract = data?.contract

  return <SlideOver open={Boolean(projectId)} eyebrow="Contract controls" title={project?.projectCode ?? 'Project details'} icon={<BadgeCheck size={18} />} closeLabel="Close project details" panelClassName="project-detail-panel" onClose={onClose}>
        {detail.isLoading ? <LoadingState label="Loading project" /> : null}
        {detail.error ? <ErrorState message={detail.error.message} /> : null}
        {data && project && contract ? <div className="project-detail-content">
          <section className="project-hero">
            <div><span className="eyebrow">{project.clientName}</span><h2>{project.name}</h2><p>{contract.contractNumber} · {contract.title}</p></div>
            <StatusBadge status={contract.status} />
          </section>
          <section className="project-value-grid">
            <div><span>Original</span><strong>{formatMoney(project.originalValueMinor, project.currencyCode)}</strong></div>
            <div><span>Approved changes</span><strong>{formatMoney(project.approvedChangesMinor, project.currencyCode)}</strong></div>
            <div><span>Revised value</span><strong>{formatMoney(project.revisedValueMinor, project.currencyCode)}</strong></div>
            <div><span>Billed</span><strong>{formatMoney(project.billedMinor, project.currencyCode)}</strong></div>
            <div><span>Unallocated</span><strong>{formatMoney(project.unallocatedMinor, project.currencyCode)}</strong></div>
          </section>

          <section className="project-section">
            <div className="project-section-heading"><div><BadgeCheck size={16} /><strong>Contract approval</strong></div><span>Requested by {contract.requestedBy}</span></div>
            {contract.reviewNote ? <p className="project-review-note">{contract.reviewNote} — {contract.reviewedBy}</p> : null}
            {contract.status === 'pending' && canApprove && contract.requestedById !== session?.user.userId
              ? <Button variant="secondary" onClick={() => beginDecision({ kind: 'contract', id: contract.id, label: contract.contractNumber })}>Review contract</Button>
              : null}
            {contract.status === 'pending' && contract.requestedById === session?.user.userId ? <small className="separation-note">Separation of duties prevents you from reviewing your own contract.</small> : null}
          </section>

          <section className="project-section">
            <div className="project-section-heading"><div><ArrowDownUp size={16} /><strong>Change orders</strong></div>{canEdit && contract.status === 'approved' ? <Button variant="ghost" onClick={() => setShowChangeOrder((value) => !value)}><Plus size={14} /> Add change order</Button> : null}</div>
            {showChangeOrder ? <ChangeOrderForm form={changeForm} pending={createChangeOrder.isPending} error={createChangeOrder.error} onCancel={() => setShowChangeOrder(false)} onSubmit={(values) => createChangeOrder.mutate(values)} /> : null}
            <div className="project-record-list">{data.changeOrders.map((order) => <article className="project-record" key={order.id}>
              <div><strong>{order.changeOrderNumber} · {order.title}</strong><span>{order.description}</span><small>{order.requestedBy}</small></div>
              <strong className={order.amountMinor < 0 ? 'negative-money' : ''}>{formatMoney(order.amountMinor, project.currencyCode)}</strong>
              <StatusBadge status={order.status} />
              {order.status === 'pending' && canApprove && order.requestedById !== session?.user.userId ? <Button variant="ghost" onClick={() => beginDecision({ kind: 'changeOrder', id: order.id, label: order.changeOrderNumber })}>Review</Button> : null}
            </article>)}</div>
            {data.changeOrders.length === 0 ? <p className="project-empty">No change orders.</p> : null}
          </section>

          <section className="project-section">
            <div className="project-section-heading"><div><CalendarClock size={16} /><strong>Billing milestones</strong></div>{canEdit && contract.status === 'approved' && project.unallocatedMinor > 0 ? <Button variant="ghost" onClick={() => setShowMilestone((value) => !value)}><Plus size={14} /> Add milestone</Button> : null}</div>
            {showMilestone ? <MilestoneForm form={milestoneForm} pending={createMilestone.isPending} error={createMilestone.error} onCancel={() => setShowMilestone(false)} onSubmit={(values) => createMilestone.mutate(values)} /> : null}
            <div className="project-record-list">{data.milestones.map((milestone) => <article className="project-record" key={milestone.id}>
              <div><strong>{milestone.name}</strong><span>Due {formatDate(milestone.dueOn)}{milestone.invoiceNumber ? ` · ${milestone.invoiceNumber}` : ''}</span><small>{milestone.createdBy}</small></div>
              <strong>{formatMoney(milestone.amountMinor, project.currencyCode)}</strong>
              <StatusBadge status={milestone.status} />
              {milestone.status === 'planned' && canEdit ? <Button variant="ghost" onClick={() => beginInvoice(milestone)}><FilePlus2 size={13} /> Invoice</Button> : null}
            </article>)}</div>
            {data.milestones.length === 0 ? <p className="project-empty">No billing milestones.</p> : null}
          </section>

          <section className="project-section project-audit">
            <div className="project-section-heading"><div><History size={16} /><strong>Audit history</strong></div><span>{data.auditEvents.length} events</span></div>
            {data.auditEvents.map((event) => <div className="project-audit-row" key={event.id}><span className="project-audit-action">{titleCase(event.action.replace('.', ' '))}</span><strong>{event.actor}</strong><span>{formatRelativeTime(event.createdAt)}</span></div>)}
          </section>
        </div> : null}
    <DecisionDialog label={decisionTarget?.label ?? ''} form={decisionForm} open={Boolean(decisionTarget)} pending={decide.isPending} error={decide.error} onClose={() => setDecisionTarget(null)} onDecision={(decision, values) => { if (decisionTarget) decide.mutate({ target: decisionTarget, input: { decision, note: values.note.trim() } }) }} />
    <MilestoneInvoiceDialog target={invoiceTarget} currencyCode={project?.currencyCode} form={invoiceForm} pending={invoiceMilestone.isPending} error={invoiceMilestone.error} onClose={() => setInvoiceTarget(null)} onSubmit={(values) => invoiceMilestone.mutate(values)} />
  </SlideOver>
}
