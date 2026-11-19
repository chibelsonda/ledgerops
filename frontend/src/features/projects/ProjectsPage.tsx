import { useQuery } from '@tanstack/react-query'
import { BriefcaseBusiness, Eye, Plus } from 'lucide-react'
import { useState } from 'react'
import { useAuth } from '@/features/auth/AuthProvider'
import { formatDate, formatMoney } from '@/shared/lib/format'
import { capabilities } from '@/shared/lib/capabilities'
import { queryKeys } from '@/shared/api/queryKeys'
import { Button } from '@/shared/ui/Button'
import { PageHeader } from '@/shared/ui/PageHeader'
import { StatusBadge } from '@/shared/ui/StatusBadge'
import { ErrorState, LoadingState } from '@/shared/ui/ViewState'
import { projectsApi } from './api'
import { ProjectDetailDialog } from './components/ProjectDetailDialog'
import { ProjectDialog } from './components/ProjectDialog'

export function ProjectsPage() {
  const { session } = useAuth()
  const canEdit = capabilities.operate(session?.user.role)
  const [creating, setCreating] = useState(false)
  const [selectedID, setSelectedID] = useState<string>()
  const projects = useQuery({ queryKey: queryKeys.projects.all, queryFn: projectsApi.list })
  const clients = useQuery({ queryKey: queryKeys.projects.clients, queryFn: projectsApi.clients, enabled: canEdit })
  if (projects.isLoading) return <LoadingState label="Loading projects" />
  if (projects.error) return <ErrorState message={projects.error.message} />

  return <div className="page">
    <PageHeader
      eyebrow="Project operations"
      title="Projects & contracts"
      description="Control contract value, approved scope changes, billing allocations, and milestone invoices from one auditable workflow."
      action={canEdit ? <Button onClick={() => setCreating(true)}><Plus size={16} /> New project</Button> : undefined}
    />
    <section className="project-overview-grid">
      <div className="panel"><span>Active projects</span><strong>{projects.data?.filter((project) => project.status === 'active').length ?? 0}</strong></div>
      <div className="panel"><span>Pending contracts</span><strong>{projects.data?.filter((project) => project.contractStatus === 'pending').length ?? 0}</strong></div>
      <div className="panel"><span>Total revised value</span><strong>{formatMoney(projects.data?.reduce((total, project) => total + project.revisedValueMinor, 0) ?? 0)}</strong></div>
      <div className="panel"><span>Billed to date</span><strong>{formatMoney(projects.data?.reduce((total, project) => total + project.billedMinor, 0) ?? 0)}</strong></div>
    </section>
    <section className="panel table-panel">
      <div className="table-scroll"><table>
        <thead><tr><th>Project</th><th>Client</th><th>Contract</th><th>Status</th><th className="numeric">Revised value</th><th className="numeric">Billed</th><th className="numeric">Unallocated</th><th><span className="sr-only">Actions</span></th></tr></thead>
        <tbody>{projects.data?.map((project) => <tr key={project.id}>
          <td><strong>{project.projectCode}</strong><div className="table-subtext"><BriefcaseBusiness size={11} /> {project.name}</div></td>
          <td><strong>{project.clientName}</strong><div className="table-subtext">Started {formatDate(project.startOn)}</div></td>
          <td><strong>{project.contractNumber}</strong><div className="table-subtext">{formatMoney(project.originalValueMinor, project.currencyCode)} original</div></td>
          <td><StatusBadge status={project.contractStatus} /></td>
          <td className="numeric">{formatMoney(project.revisedValueMinor, project.currencyCode)}</td>
          <td className="numeric outstanding">{formatMoney(project.billedMinor, project.currencyCode)}</td>
          <td className="numeric">{formatMoney(project.unallocatedMinor, project.currencyCode)}</td>
          <td className="row-action"><Button aria-label={`View ${project.projectCode}`} variant="ghost" onClick={() => setSelectedID(project.id)}><Eye size={14} /> View</Button></td>
        </tr>)}</tbody>
      </table></div>
      {projects.data?.length === 0 ? <div className="empty-table">No projects yet. Create one to submit its contract for approval.</div> : null}
    </section>
    {canEdit ? <ProjectDialog open={creating} clients={clients.data ?? []} onClose={() => setCreating(false)} onCreated={setSelectedID} /> : null}
    <ProjectDetailDialog projectId={selectedID} onClose={() => setSelectedID(undefined)} />
  </div>
}
