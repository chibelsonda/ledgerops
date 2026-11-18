import { Dialog, DialogBackdrop, DialogPanel, DialogTitle } from '@headlessui/react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { BriefcaseBusiness, X } from 'lucide-react'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { Button } from '../../../shared/ui/Button'
import { Input, Select } from '../../../shared/ui/FormControls'
import { projectsApi } from '../api'
import { projectFormSchema, toMinor, type ProjectFormValues } from '../schema'
import type { ClientOption, CreateProjectInput } from '../types'

const today = () => new Date().toISOString().slice(0, 10)
const defaults = (): ProjectFormValues => ({ clientId: '', projectCode: '', name: '', currencyCode: 'USD', startOn: today(), endOn: '', contractNumber: '', contractTitle: '', originalValue: 0 })

export function ProjectDialog({ open, clients, onClose, onCreated }: { open: boolean; clients: ClientOption[]; onClose: () => void; onCreated: (projectId: string) => void }) {
  const queryClient = useQueryClient()
  const form = useForm<ProjectFormValues>({ resolver: zodResolver(projectFormSchema), mode: 'onChange', defaultValues: defaults() })
  useEffect(() => { if (open) form.reset(defaults()) }, [form, open])
  const create = useMutation({
    mutationFn: (values: ProjectFormValues) => {
      const input: CreateProjectInput = {
        clientId: values.clientId,
        projectCode: values.projectCode.trim(),
        name: values.name.trim(),
        currencyCode: values.currencyCode.trim().toUpperCase(),
        startOn: values.startOn,
        endOn: values.endOn || undefined,
        contractNumber: values.contractNumber.trim(),
        contractTitle: values.contractTitle.trim(),
        originalValueMinor: toMinor(values.originalValue),
      }
      return projectsApi.create(input)
    },
    onSuccess: async (detail) => {
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
      onClose()
      onCreated(detail.project.id)
    },
  })

  return <Dialog className="invoice-dialog" open={open} onClose={onClose}>
    <DialogBackdrop className="invoice-dialog-backdrop" />
    <div className="invoice-dialog-positioner">
      <DialogPanel className="invoice-dialog-panel project-create-panel">
        <div className="invoice-dialog-heading">
          <div className="invoice-dialog-title-icon"><BriefcaseBusiness size={18} /></div>
          <div><span className="eyebrow">Project billing</span><DialogTitle>New project and contract</DialogTitle></div>
          <Button aria-label="Close project form" className="invoice-dialog-close" variant="ghost" onClick={onClose}><X size={24} /></Button>
        </div>
        <form className="invoice-form" onSubmit={form.handleSubmit((values) => create.mutate(values))} noValidate>
          <h3 className="form-section-title">Project information</h3>
          <div className="invoice-form-grid">
            <Select label="Client" error={form.formState.errors.clientId?.message} {...form.register('clientId')}><option value="">Select a client</option>{clients.map((client) => <option key={client.id} value={client.id}>{client.name}</option>)}</Select>
            <Input label="Project code" error={form.formState.errors.projectCode?.message} {...form.register('projectCode')} />
            <Input label="Project name" error={form.formState.errors.name?.message} {...form.register('name')} />
            <Input label="Currency" maxLength={3} error={form.formState.errors.currencyCode?.message} {...form.register('currencyCode')} />
            <Input label="Start date" type="date" error={form.formState.errors.startOn?.message} {...form.register('startOn')} />
            <Input label="End date (optional)" type="date" error={form.formState.errors.endOn?.message} {...form.register('endOn')} />
          </div>
          <h3 className="form-section-title">Contract intake</h3>
          <div className="invoice-form-grid">
            <Input label="Contract number" error={form.formState.errors.contractNumber?.message} {...form.register('contractNumber')} />
            <Input label="Original contract value" type="number" min="0.01" step="0.01" error={form.formState.errors.originalValue?.message} {...form.register('originalValue', { valueAsNumber: true })} />
            <Input className="col-span-2" label="Contract title" error={form.formState.errors.contractTitle?.message} {...form.register('contractTitle')} />
          </div>
          <p className="project-policy-note">The contract is submitted for independent approval immediately. Its value and terms become immutable after review.</p>
          {create.error ? <p className="form-error">{create.error.message}</p> : null}
          <div className="invoice-dialog-actions"><Button type="button" variant="ghost" onClick={onClose}>Cancel</Button><Button type="submit" disabled={create.isPending}>{create.isPending ? 'Creating…' : 'Create and submit'}</Button></div>
        </form>
      </DialogPanel>
    </div>
  </Dialog>
}
