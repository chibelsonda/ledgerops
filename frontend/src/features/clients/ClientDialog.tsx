import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Building2, Pencil, Plus, Star } from 'lucide-react'
import { Controller, useForm, type UseFormReturn } from 'react-hook-form'
import { useEffect, useMemo, useState } from 'react'
import { queryKeys } from '@/shared/api/queryKeys'
import { Button } from '@/shared/ui/Button'
import { Input, Textarea } from '@/shared/ui/FormControls'
import { SlideOver } from '@/shared/ui/SlideOver'
import { Toggle } from '@/shared/ui/Toggle'
import { clientsApi } from './api'
import { clientFormSchema, contactFormSchema, type ClientFormValues, type ContactFormValues } from './schema'
import type { Client, ClientInput, Contact, ContactInput } from './types'

const contactDefaults: ContactFormValues = {
  contactName: '', contactEmail: '', contactPhone: '', contactRole: '', isPrimary: false,
}

function clientDefaults(client: Client | null): ClientFormValues {
  return {
    name: client?.name ?? '',
    paymentTermsDays: client?.paymentTermsDays ?? 30,
    remindersEnabled: client?.remindersEnabled ?? true,
    reminderDelayDays: client?.reminderDelayDays ?? 3,
    contactName: client?.primaryContact ?? '',
    contactEmail: client?.primaryEmail ?? '',
    contactPhone: '',
    contactRole: 'Billing contact',
    auditNote: '',
  }
}

export function ClientDialog({ open, client, onClose }: { open: boolean; client: Client | null; onClose: () => void }) {
  const queryClient = useQueryClient()
  const editing = Boolean(client)
  const schema = useMemo(() => clientFormSchema(editing), [editing])
  const form = useForm<ClientFormValues>({
    resolver: zodResolver(schema), mode: 'onChange', defaultValues: clientDefaults(client),
  })

  useEffect(() => {
    if (open) form.reset(clientDefaults(client))
  }, [client, form, open])

  const save = useMutation({
    mutationFn: (values: ClientFormValues) => {
      const input: ClientInput = {
        name: values.name.trim(),
        paymentTermsDays: values.paymentTermsDays,
        remindersEnabled: values.remindersEnabled,
        reminderDelayDays: values.reminderDelayDays,
        ...(editing
          ? { auditNote: values.auditNote?.trim() }
          : { primaryContact: contactInput(values) }),
      }
      return client ? clientsApi.update(client.id, input) : clientsApi.create(input)
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.clients.all }),
        queryClient.invalidateQueries({ queryKey: queryKeys.invoices.clients }),
      ])
      onClose()
    },
  })

  return (
    <SlideOver
      open={open}
      eyebrow="Client records"
      title={client ? `Manage ${client.name}` : 'Add client'}
      icon={<Building2 size={18} />}
      closeLabel="Close client form"
      panelClassName="client-dialog-panel"
      onClose={onClose}
    >
      <form className="invoice-form" onSubmit={form.handleSubmit((values) => save.mutate(values))} noValidate>
        <div className="client-form-grid">
          <Input label="Client name" error={form.formState.errors.name?.message} {...form.register('name')} />
          <Input label="Payment terms (days)" type="number" error={form.formState.errors.paymentTermsDays?.message} {...form.register('paymentTermsDays', { valueAsNumber: true })} />
          <Controller control={form.control} name="remindersEnabled" render={({ field }) => <Toggle label="Automatic reminders" description="Queue overdue collection reminders" checked={field.value} onChange={field.onChange} />} />
          <Input label="Days after due date" type="number" disabled={!form.watch('remindersEnabled')} error={form.formState.errors.reminderDelayDays?.message} {...form.register('reminderDelayDays', { valueAsNumber: true })} />
        </div>
        {!editing ? <PrimaryContactFields form={form} /> : <Textarea label="Audit note" rows={2} error={form.formState.errors.auditNote?.message} {...form.register('auditNote')} />}
        {save.error ? <p className="form-error">{save.error.message}</p> : null}
        <div className="invoice-dialog-actions">
          <Button variant="ghost" type="button" onClick={onClose}>Cancel</Button>
          <Button type="submit" disabled={save.isPending}>{save.isPending ? 'Saving…' : editing ? 'Save client' : 'Create client'}</Button>
        </div>
      </form>
      {client ? <ContactManager client={client} /> : null}
    </SlideOver>
  )
}

function PrimaryContactFields({ form }: { form: UseFormReturn<ClientFormValues> }) {
  return <>
    <h3 className="form-section-title">Primary billing contact</h3>
    <div className="client-form-grid">
      <Input label="Contact name" error={form.formState.errors.contactName?.message} {...form.register('contactName')} />
      <Input label="Email address" type="email" error={form.formState.errors.contactEmail?.message} {...form.register('contactEmail')} />
      <Input label="Phone" error={form.formState.errors.contactPhone?.message} {...form.register('contactPhone')} />
      <Input label="Role" error={form.formState.errors.contactRole?.message} {...form.register('contactRole')} />
    </div>
  </>
}

function ContactManager({ client }: { client: Client }) {
  const queryClient = useQueryClient()
  const [editingContact, setEditingContact] = useState<Contact | null>(null)
  const form = useForm<ContactFormValues>({ resolver: zodResolver(contactFormSchema), mode: 'onChange', defaultValues: contactDefaults })
  const contacts = useQuery({ queryKey: queryKeys.clients.contacts(client.id), queryFn: () => clientsApi.contacts(client.id) })
  useEffect(() => {
    form.reset(contactDefaults)
    setEditingContact(null)
  }, [client.id, form])
  const save = useMutation({
    mutationFn: (values: ContactFormValues) => {
      const input = contactInput(values)
      return editingContact ? clientsApi.updateContact(editingContact.id, { ...input, isPrimary: values.isPrimary }) : clientsApi.createContact(client.id, { ...input, isPrimary: values.isPrimary })
    },
    onSuccess: async () => {
      form.reset(contactDefaults)
      setEditingContact(null)
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.clients.all }),
        queryClient.invalidateQueries({ queryKey: queryKeys.clients.contacts(client.id) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.invoices.clients }),
      ])
    },
  })

  function edit(contact: Contact) {
    setEditingContact(contact)
    form.reset({ contactName: contact.name, contactEmail: contact.email, contactPhone: contact.phone, contactRole: contact.roleTitle, isPrimary: contact.isPrimary })
  }

  return <section className="contact-panel">
    <div className="contact-heading"><strong>Contacts</strong><span>{contacts.data?.length ?? 0} records</span></div>
    {contacts.data?.map((contact) => <button type="button" className="contact-row" key={contact.id} onClick={() => edit(contact)}><span>{contact.isPrimary ? <Star size={13} /> : null}<strong>{contact.name}</strong></span><span>{contact.email}</span><small>{contact.roleTitle || 'Contact'}</small><Pencil size={13} /></button>)}
    <form className="contact-form" onSubmit={form.handleSubmit((values) => save.mutate(values))} noValidate>
      <div className="client-form-grid">
        <Input label="Contact name" error={form.formState.errors.contactName?.message} {...form.register('contactName')} />
        <Input label="Contact email" type="email" error={form.formState.errors.contactEmail?.message} {...form.register('contactEmail')} />
        <Input label="Phone" error={form.formState.errors.contactPhone?.message} {...form.register('contactPhone')} />
        <Input label="Role" error={form.formState.errors.contactRole?.message} {...form.register('contactRole')} />
      </div>
      <Controller control={form.control} name="isPrimary" render={({ field }) => <Toggle label="Primary billing contact" checked={field.value} onChange={field.onChange} />} />
      {save.error ? <p className="form-error">{save.error.message}</p> : null}
      <div className="invoice-dialog-actions">
        {editingContact ? <Button type="button" variant="ghost" onClick={() => { setEditingContact(null); form.reset(contactDefaults) }}>Cancel edit</Button> : null}
        <Button type="submit" disabled={save.isPending}><Plus size={14} /> {editingContact ? 'Update contact' : 'Add contact'}</Button>
      </div>
    </form>
  </section>
}

function contactInput(values: Pick<ClientFormValues, 'contactName' | 'contactEmail' | 'contactPhone' | 'contactRole'>): Omit<ContactInput, 'isPrimary'> {
  return {
    name: values.contactName.trim(),
    email: values.contactEmail.trim(),
    phone: values.contactPhone.trim(),
    roleTitle: values.contactRole.trim(),
  }
}
