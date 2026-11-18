import { Dialog, DialogBackdrop, DialogPanel, DialogTitle } from '@headlessui/react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowDownUp, BadgeCheck, CalendarClock, FilePlus2, History, Plus, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useAuth } from '@/features/auth/AuthProvider'
import { formatDate, formatMoney, formatRelativeTime, titleCase } from '@/shared/lib/format'
import { Button } from '@/shared/ui/Button'
import { Input, Textarea } from '@/shared/ui/FormControls'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ErrorState, LoadingState } from '@/shared/ui/ViewState'
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
  const detail = useQuery({ queryKey: ['projects', projectId], queryFn: () => projectsApi.get(projectId!), enabled: Boolean(projectId) })
  const changeForm = useForm<ChangeOrderFormValues>({ resolver: zodResolver(changeOrderFormSchema), mode: 'onChange', defaultValues: { changeOrderNumber: '', title: '', description: '', amount: 0 } })
  const milestoneForm = useForm<MilestoneFormValues>({ resolver: zodResolver(milestoneFormSchema), mode: 'onChange', defaultValues: { name: '', amount: 0, dueOn: today() } })
  const decisionForm = useForm<DecisionFormValues>({ resolver: zodResolver(decisionFormSchema), mode: 'onChange', defaultValues: { note: '' } })
  const invoiceForm = useForm<InvoiceMilestoneFormValues>({ resolver: zodResolver(invoiceMilestoneFormSchema), mode: 'onChange', defaultValues: { invoiceNumber: '', issuedOn: today(), dueOn: today() } })

  useEffect(() => {
    setShowChangeOrder(false); setShowMilestone(false); setDecisionTarget(null); setInvoiceTarget(null)
  }, [projectId])

  const acceptDetail = async (value: ProjectDetail) => {
    queryClient.setQueryData(['projects', projectId], value)
    await Promise.all([queryClient.invalidateQueries({ queryKey: ['projects'] }), queryClient.invalidateQueries({ queryKey: ['invoices'] })])
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
  const submitDecision = (decision: 'approved' | 'rejected') => decisionForm.handleSubmit((values) => {
    if (decisionTarget) decide.mutate({ target: decisionTarget, input: { decision, note: values.note.trim() } })
  })()
  const beginInvoice = (milestone: BillingMilestone) => {
    invoiceForm.reset({ invoiceNumber: '', issuedOn: today(), dueOn: milestone.dueOn >= today() ? milestone.dueOn : today() })
    setInvoiceTarget(milestone)
  }

  const canEdit = session?.user.role === 'admin' || session?.user.role === 'controller'
  const canApprove = session?.user.role === 'admin'
  const data = detail.data
  const project = data?.project
  const contract = data?.contract

  return <Dialog className="invoice-dialog" open={Boolean(projectId)} onClose={onClose}>
    <DialogBackdrop className="invoice-dialog-backdrop" />
    <div className="invoice-dialog-positioner">
      <DialogPanel className="invoice-dialog-panel project-detail-panel">
        <div className="invoice-dialog-heading">
          <div className="invoice-dialog-title-icon"><BadgeCheck size={18} /></div>
          <div><span className="eyebrow">Contract controls</span><DialogTitle>{project?.projectCode ?? 'Project details'}</DialogTitle></div>
          <Button aria-label="Close project details" className="invoice-dialog-close" variant="ghost" onClick={onClose}><X size={24} /></Button>
        </div>
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
            {showChangeOrder ? <form className="project-inline-form" onSubmit={changeForm.handleSubmit((values) => createChangeOrder.mutate(values))} noValidate>
              <Input label="Change-order number" error={changeForm.formState.errors.changeOrderNumber?.message} {...changeForm.register('changeOrderNumber')} />
              <Input label="Title" error={changeForm.formState.errors.title?.message} {...changeForm.register('title')} />
              <Input label="Amount (+/−)" type="number" step="0.01" error={changeForm.formState.errors.amount?.message} {...changeForm.register('amount', { valueAsNumber: true })} />
              <Textarea className="col-span-full" label="Scope justification" rows={3} error={changeForm.formState.errors.description?.message} {...changeForm.register('description')} />
              {createChangeOrder.error ? <p className="form-error col-span-full">{createChangeOrder.error.message}</p> : null}
              <div className="invoice-dialog-actions col-span-full"><Button type="button" variant="ghost" onClick={() => setShowChangeOrder(false)}>Cancel</Button><Button type="submit" disabled={createChangeOrder.isPending}>Submit for approval</Button></div>
            </form> : null}
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
            {showMilestone ? <form className="project-inline-form" onSubmit={milestoneForm.handleSubmit((values) => createMilestone.mutate(values))} noValidate>
              <Input label="Milestone name" error={milestoneForm.formState.errors.name?.message} {...milestoneForm.register('name')} />
              <Input label="Amount" type="number" min="0.01" step="0.01" error={milestoneForm.formState.errors.amount?.message} {...milestoneForm.register('amount', { valueAsNumber: true })} />
              <Input label="Due date" type="date" error={milestoneForm.formState.errors.dueOn?.message} {...milestoneForm.register('dueOn')} />
              {createMilestone.error ? <p className="form-error col-span-full">{createMilestone.error.message}</p> : null}
              <div className="invoice-dialog-actions col-span-full"><Button type="button" variant="ghost" onClick={() => setShowMilestone(false)}>Cancel</Button><Button type="submit" disabled={createMilestone.isPending}>Add milestone</Button></div>
            </form> : null}
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
      </DialogPanel>
    </div>

    <Dialog className="project-nested-dialog" open={Boolean(decisionTarget)} onClose={() => setDecisionTarget(null)}>
      <DialogBackdrop className="project-nested-backdrop" /><div className="project-nested-positioner"><DialogPanel className="project-nested-panel"><DialogTitle>Review {decisionTarget?.label}</DialogTitle><p>Approval is final and creates an immutable financial record.</p><Textarea label="Review note" rows={4} error={decisionForm.formState.errors.note?.message} {...decisionForm.register('note')} />{decide.error ? <p className="form-error">{decide.error.message}</p> : null}<div className="invoice-dialog-actions"><Button variant="ghost" onClick={() => setDecisionTarget(null)}>Cancel</Button><Button variant="secondary" disabled={decide.isPending} onClick={() => void submitDecision('rejected')}>Reject</Button><Button disabled={decide.isPending} onClick={() => void submitDecision('approved')}>Approve</Button></div></DialogPanel></div>
    </Dialog>

    <Dialog className="project-nested-dialog" open={Boolean(invoiceTarget)} onClose={() => setInvoiceTarget(null)}>
      <DialogBackdrop className="project-nested-backdrop" /><div className="project-nested-positioner"><DialogPanel className="project-nested-panel"><DialogTitle>Invoice {invoiceTarget?.name}</DialogTitle><p>The invoice amount is fixed at {invoiceTarget && project ? formatMoney(invoiceTarget.amountMinor, project.currencyCode) : ''}.</p><form onSubmit={invoiceForm.handleSubmit((values) => invoiceMilestone.mutate(values))} noValidate><div className="project-invoice-grid"><Input label="Invoice number" error={invoiceForm.formState.errors.invoiceNumber?.message} {...invoiceForm.register('invoiceNumber')} /><Input label="Issue date" type="date" error={invoiceForm.formState.errors.issuedOn?.message} {...invoiceForm.register('issuedOn')} /><Input label="Due date" type="date" error={invoiceForm.formState.errors.dueOn?.message} {...invoiceForm.register('dueOn')} /></div>{invoiceMilestone.error ? <p className="form-error">{invoiceMilestone.error.message}</p> : null}<div className="invoice-dialog-actions"><Button type="button" variant="ghost" onClick={() => setInvoiceTarget(null)}>Cancel</Button><Button type="submit" disabled={invoiceMilestone.isPending}>Create invoice</Button></div></form></DialogPanel></div>
    </Dialog>
  </Dialog>
}
