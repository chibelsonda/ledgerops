import { Dialog, DialogBackdrop, DialogPanel, DialogTitle } from '@headlessui/react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarCheck, Check, ClipboardCheck, Clock3, LockKeyhole, Plus, RotateCcw, Send, ShieldCheck, X } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { formatDate, formatRelativeTime, titleCase } from '../../shared/lib/format'
import { Button } from '../../shared/ui/Button'
import { Input, Textarea } from '../../shared/ui/FormControls'
import { PageHeader } from '../../shared/ui/PageHeader'
import { StatusBadge } from '../../shared/ui/StatusBadge'
import { ErrorState, LoadingState } from '../../shared/ui/ViewState'
import { useAuth } from '../auth/AuthProvider'
import { closeApi } from './api'
import { closeDecisionSchema, closeEvidenceSchema, closePeriodSchema, type CloseDecisionForm, type CloseEvidenceForm, type ClosePeriodForm } from './schema'
import type { CloseDetail, CloseTask } from './types'

function nextMonthDefaults() {
  const now = new Date()
  const month = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 1))
  const due = new Date(Date.UTC(month.getUTCFullYear(), month.getUTCMonth() + 1, 5))
  return { monthStart: month.toISOString().slice(0, 10), dueOn: due.toISOString().slice(0, 10) }
}

export function ClosePage() {
  const { session } = useAuth()
  const queryClient = useQueryClient()
  const [selectedId, setSelectedId] = useState<string>()
  const [creating, setCreating] = useState(false)
  const [task, setTask] = useState<CloseTask>()
  const [reviewing, setReviewing] = useState(false)
  const periods = useQuery({ queryKey: ['close-periods'], queryFn: closeApi.list })
  useEffect(() => { if (!selectedId && periods.data?.[0]) setSelectedId(periods.data[0].id) }, [periods.data, selectedId])
  const detail = useQuery({ queryKey: ['close-periods', selectedId], queryFn: () => closeApi.get(selectedId!), enabled: Boolean(selectedId) })

  const syncDetail = async (value: CloseDetail) => {
    queryClient.setQueryData(['close-periods', value.period.id], value)
    await queryClient.invalidateQueries({ queryKey: ['close-periods'], exact: true })
  }
  const submit = useMutation({ mutationFn: () => closeApi.submit(selectedId!), onSuccess: syncDetail })
  const reopen = useMutation({ mutationFn: (taskId: string) => closeApi.updateTask(taskId, { status: 'pending', evidenceNote: '' }), onSuccess: syncDetail })

  if (periods.isLoading) return <LoadingState label="Loading close workspace" />
  if (periods.error) return <ErrorState message={periods.error.message} />
  const current = detail.data
  const canCreate = session?.user.role === 'controller'
  const canWork = (session?.user.role === 'admin' || session?.user.role === 'controller') && current?.period.status === 'draft'
  const canReview = session?.user.role === 'admin' && current?.period.status === 'in_review' && current.period.submittedById !== session.user.userId

  return <div className="page">
    <PageHeader eyebrow="Accounting operations" title="Month-end close" description="Run a repeatable close checklist, attach evidence, and require an independent final sign-off before the period is locked." action={canCreate ? <Button onClick={() => setCreating(true)}><Plus size={16} /> New close period</Button> : undefined} />
    <div className="close-layout">
      <aside className="panel close-period-list">
        <div className="panel-heading"><div><span className="eyebrow">Close calendar</span><h2>Accounting periods</h2></div></div>
        {periods.data?.map((period) => <button className={`close-period-option ${period.id === selectedId ? 'selected' : ''}`} key={period.id} onClick={() => setSelectedId(period.id)}>
          <span><strong>{new Intl.DateTimeFormat('en-US', { month: 'long', year: 'numeric', timeZone: 'UTC' }).format(new Date(`${period.monthStart}T00:00:00Z`))}</strong><small>Due {formatDate(period.dueOn)}</small></span>
          <span><StatusBadge status={period.status} /><small>{period.completedTasks}/{period.totalTasks} complete</small></span>
        </button>)}
        {periods.data?.length === 0 ? <p className="project-empty">No close periods have been created.</p> : null}
      </aside>

      <section className="panel close-workspace">
        {detail.isLoading ? <LoadingState label="Loading close period" /> : detail.error ? <ErrorState message={detail.error.message} /> : current ? <>
          <div className="close-hero">
            <div><span className="eyebrow">{formatDate(current.period.monthStart)}</span><h2>Close control center</h2><p>Owned by {current.period.createdBy} · due {formatDate(current.period.dueOn)}</p></div>
            <div className="close-hero-actions"><StatusBadge status={current.period.status} />{canReview ? <Button onClick={() => setReviewing(true)}><ShieldCheck size={15} /> Review close</Button> : null}</div>
          </div>
          <div className="close-summary-grid">
            <div><ClipboardCheck /><span>Checklist progress</span><strong>{current.period.progress}%</strong></div>
            <div><Check /><span>Completed tasks</span><strong>{current.period.completedTasks} of {current.period.totalTasks}</strong></div>
            <div><ShieldCheck /><span>Open exceptions</span><strong>{current.period.openExceptions}</strong></div>
          </div>
          <div className="close-progress"><span style={{ width: `${current.period.progress}%` }} /></div>
          {current.period.reviewNote ? <p className="close-review-note"><strong>{current.period.status === 'closed' ? 'Final review' : 'Returned for changes'}:</strong> {current.period.reviewNote}</p> : null}

          <section className="close-section">
            <div className="project-section-heading"><div><CalendarCheck size={16} /><strong>Close checklist</strong></div><span>Evidence is required for completion</span></div>
            <div className="close-task-list">{current.tasks.map((item) => <article className={`close-task ${item.status === 'completed' ? 'completed' : ''}`} key={item.id}>
              <span className="close-task-check">{item.status === 'completed' ? <Check size={15} /> : item.position}</span>
              <div><small>{item.category}</small><strong>{item.title}</strong>{item.evidenceNote ? <p>{item.evidenceNote}</p> : null}{item.completedBy ? <span>{item.completedBy} · {formatRelativeTime(item.completedAt!)}</span> : null}</div>
              {canWork && item.status === 'pending' ? <Button variant="secondary" onClick={() => setTask(item)}>Complete</Button> : null}
              {canWork && item.status === 'completed' ? <Button aria-label={`Reopen ${item.title}`} variant="ghost" onClick={() => reopen.mutate(item.id)}><RotateCcw size={14} /> Reopen</Button> : null}
              {current.period.status === 'closed' ? <LockKeyhole size={16} /> : null}
            </article>)}</div>
            {canWork ? <div className="close-submit"><span>{current.period.completedTasks === current.period.totalTasks ? 'Checklist complete and ready for independent review.' : `${current.period.totalTasks - current.period.completedTasks} tasks remain before submission.`}</span><Button disabled={current.period.progress !== 100 || submit.isPending} onClick={() => submit.mutate()}><Send size={15} /> Submit for review</Button></div> : null}
          </section>
          <section className="close-section close-audit"><div className="project-section-heading"><div><Clock3 size={16} /><strong>Close audit trail</strong></div></div>{current.auditEvents.map((event) => <div className="project-audit-row" key={event.id}><span className="project-audit-action">{titleCase(event.action.replace('close_period.', '').replace('close_task.', 'Task '))}</span><strong>{event.actor}</strong><span>{formatRelativeTime(event.createdAt)}</span></div>)}</section>
        </> : <div className="select-prompt"><CalendarCheck /><strong>Select a period</strong><span>Choose a close period to inspect its checklist.</span></div>}
      </section>
    </div>
    <CreatePeriodDialog open={creating} onClose={() => setCreating(false)} onCreated={(value) => { void syncDetail(value); setSelectedId(value.period.id); setCreating(false) }} />
    <EvidenceDialog task={task} onClose={() => setTask(undefined)} onCompleted={(value) => { void syncDetail(value); setTask(undefined) }} />
    {current ? <ReviewDialog open={reviewing} periodId={current.period.id} onClose={() => setReviewing(false)} onDecided={(value) => { void syncDetail(value); setReviewing(false) }} /> : null}
  </div>
}

function CreatePeriodDialog({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: (value: CloseDetail) => void }) {
  const form = useForm<ClosePeriodForm>({ resolver: zodResolver(closePeriodSchema), defaultValues: nextMonthDefaults() })
  const mutation = useMutation({ mutationFn: closeApi.create, onSuccess: onCreated })
  return <Modal open={open} title="Create close period" onClose={onClose}><form onSubmit={form.handleSubmit((value) => mutation.mutate(value))} className="close-dialog-form"><Input label="Close month" type="date" error={form.formState.errors.monthStart?.message} {...form.register('monthStart')} /><Input label="Close due date" type="date" error={form.formState.errors.dueOn?.message} {...form.register('dueOn')} /><div className="invoice-dialog-actions"><Button type="button" variant="ghost" onClick={onClose}>Cancel</Button><Button type="submit" disabled={mutation.isPending}>Create checklist</Button></div></form></Modal>
}

function EvidenceDialog({ task, onClose, onCompleted }: { task?: CloseTask; onClose: () => void; onCompleted: (value: CloseDetail) => void }) {
  const form = useForm<CloseEvidenceForm>({ resolver: zodResolver(closeEvidenceSchema), defaultValues: { evidenceNote: '' } })
  const mutation = useMutation({ mutationFn: (value: CloseEvidenceForm) => closeApi.updateTask(task!.id, { status: 'completed', ...value }), onSuccess: onCompleted })
  return <Modal open={Boolean(task)} title={task?.title ?? 'Complete task'} onClose={onClose}><form onSubmit={form.handleSubmit((value) => mutation.mutate(value))} className="close-dialog-form"><p>Record what was verified so the close binder has a durable audit trail.</p><Textarea label="Completion evidence" rows={4} error={form.formState.errors.evidenceNote?.message} {...form.register('evidenceNote')} /><div className="invoice-dialog-actions"><Button type="button" variant="ghost" onClick={onClose}>Cancel</Button><Button type="submit" disabled={mutation.isPending}>Complete task</Button></div></form></Modal>
}

function ReviewDialog({ open, periodId, onClose, onDecided }: { open: boolean; periodId: string; onClose: () => void; onDecided: (value: CloseDetail) => void }) {
  const form = useForm<CloseDecisionForm>({ resolver: zodResolver(closeDecisionSchema), defaultValues: { note: '' } })
  const mutation = useMutation({ mutationFn: ({ decision, note }: CloseDecisionForm & { decision: 'approved' | 'rejected' }) => closeApi.decide(periodId, { decision, note }), onSuccess: onDecided })
  return <Modal open={open} title="Independent close review" onClose={onClose}><form className="close-dialog-form"><p>Approval permanently locks this accounting period and its evidence.</p><Textarea label="Review note" rows={4} error={form.formState.errors.note?.message} {...form.register('note')} /><div className="invoice-dialog-actions"><Button type="button" variant="secondary" onClick={form.handleSubmit((value) => mutation.mutate({ ...value, decision: 'rejected' }))}>Return for changes</Button><Button type="button" onClick={form.handleSubmit((value) => mutation.mutate({ ...value, decision: 'approved' }))}>Approve and close</Button></div></form></Modal>
}

function Modal({ open, title, onClose, children }: { open: boolean; title: string; onClose: () => void; children: ReactNode }) {
  return <Dialog className="project-nested-dialog" open={open} onClose={onClose}><DialogBackdrop className="project-nested-backdrop" /><div className="project-nested-positioner"><DialogPanel className="project-nested-panel"><div className="close-modal-title"><DialogTitle>{title}</DialogTitle><Button aria-label={`Close ${title}`} variant="ghost" onClick={onClose}><X size={18} /></Button></div>{children}</DialogPanel></div></Dialog>
}
