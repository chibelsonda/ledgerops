import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { Building2, ChevronLeft, ChevronRight, Pencil, Plus, Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useAuth } from '@/features/auth/AuthProvider'
import { formatMoney } from '@/shared/lib/format'
import { capabilities } from '@/shared/lib/capabilities'
import { queryKeys } from '@/shared/api/queryKeys'
import { Button } from '@/shared/ui/Button'
import { Input } from '@/shared/ui/FormControls'
import { PageHeader } from '@/shared/ui/PageHeader'
import { ErrorState, LoadingState } from '@/shared/ui/ViewState'
import { clientsApi } from './api'
import { ClientDialog } from './ClientDialog'
import type { Client } from './types'

export function ClientsPage() {
  const { session } = useAuth(); const canEdit = capabilities.operate(session?.user.role); const [search, setSearch] = useState(''); const [query, setQuery] = useState(''); const [page, setPage] = useState(1); const [selected, setSelected] = useState<Client | null | undefined>(undefined)
  useEffect(() => { const id = window.setTimeout(() => { setQuery(search.trim()); setPage(1) }, 300); return () => window.clearTimeout(id) }, [search])
  const result = useQuery({ queryKey: queryKeys.clients.list(query, page), queryFn: () => clientsApi.list(query, page, 10), placeholderData: keepPreviousData })
  if (result.isLoading) return <LoadingState label="Loading clients" />
  if (result.error) return <ErrorState message={result.error.message} />
  const data = result.data!
  return <div className="page"><PageHeader eyebrow="Accounts receivable" title="Clients" description="Manage billing contacts, payment terms, and automated reminder policies." action={canEdit ? <Button onClick={() => setSelected(null)}><Plus size={16} /> Add client</Button> : undefined} />
    <section className="panel table-panel"><div className="table-tools"><div className="invoice-search"><Input hideLabel label="Search clients" icon={<Search size={16} />} value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search client, contact, or email…" /></div><span className="record-count">{data.pagination.totalItems} clients</span></div><div className="table-scroll"><table><thead><tr><th>Client</th><th>Primary contact</th><th>Policy</th><th className="numeric">Invoices</th><th className="numeric">Outstanding</th>{canEdit ? <th><span className="sr-only">Actions</span></th> : null}</tr></thead><tbody>{data.items.map((client) => <tr key={client.id}><td><strong>{client.name}</strong><div className="table-subtext"><Building2 size={11} /> {client.contactCount} contact{client.contactCount === 1 ? '' : 's'}</div></td><td><strong>{client.primaryContact || 'No primary contact'}</strong><div className="table-subtext">{client.primaryEmail || 'Reminder unavailable'}</div></td><td>{client.remindersEnabled ? `${client.reminderDelayDays} days after due` : 'Disabled'}</td><td className="numeric">{client.invoiceCount}</td><td className="numeric outstanding">{formatMoney(client.outstandingMinor, client.currencyCode)}</td>{canEdit ? <td className="row-action"><Button variant="ghost" onClick={() => setSelected(client)}><Pencil size={14} /> Manage</Button></td> : null}</tr>)}</tbody></table></div>{data.items.length === 0 ? <div className="empty-table">No clients match that search.</div> : null}<div className="table-pagination"><span>Page {data.pagination.page} of {Math.max(data.pagination.totalPages, 1)}</span><Button aria-label="Previous client page" variant="secondary" disabled={page <= 1} onClick={() => setPage((v) => v - 1)}><ChevronLeft size={15} /></Button><Button aria-label="Next client page" variant="secondary" disabled={page >= data.pagination.totalPages} onClick={() => setPage((v) => v + 1)}><ChevronRight size={15} /></Button></div></section>
    {canEdit ? <ClientDialog open={selected !== undefined} client={selected ?? null} onClose={() => setSelected(undefined)} /> : null}
  </div>
}
