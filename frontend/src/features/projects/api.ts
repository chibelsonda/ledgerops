import { apiFormRequest, apiRequest } from '@/shared/api/http'
import type { ChangeOrderInput, ClientOption, CreateProjectInput, DecisionInput, MilestoneInput, MilestoneInvoiceInput, ProjectDetail, ProjectSummary } from './types'

export const projectsApi = {
  list: async () => (await apiRequest<{ items: ProjectSummary[] }>('/projects')).items,
  clients: async () => (await apiRequest<{ items: ClientOption[] }>('/projects/clients')).items,
  get: (projectId: string) => apiRequest<ProjectDetail>(`/projects/${projectId}`),
  create: (input: CreateProjectInput) => apiFormRequest<ProjectDetail>('/projects', { method: 'POST', body: JSON.stringify(input), successMessage: 'Project created successfully.' }),
  decideContract: (contractId: string, input: DecisionInput) => apiFormRequest<ProjectDetail>(`/contracts/${contractId}/decision`, { method: 'PATCH', body: JSON.stringify(input), successMessage: 'Contract decision saved.' }),
  createChangeOrder: (contractId: string, input: ChangeOrderInput) => apiFormRequest<ProjectDetail>(`/contracts/${contractId}/change-orders`, { method: 'POST', body: JSON.stringify(input), successMessage: 'Change order created successfully.' }),
  decideChangeOrder: (changeOrderId: string, input: DecisionInput) => apiFormRequest<ProjectDetail>(`/change-orders/${changeOrderId}/decision`, { method: 'PATCH', body: JSON.stringify(input), successMessage: 'Change-order decision saved.' }),
  createMilestone: (contractId: string, input: MilestoneInput) => apiFormRequest<ProjectDetail>(`/contracts/${contractId}/milestones`, { method: 'POST', body: JSON.stringify(input), successMessage: 'Milestone created successfully.' }),
  invoiceMilestone: (milestoneId: string, input: MilestoneInvoiceInput) => apiFormRequest<ProjectDetail>(`/milestones/${milestoneId}/invoice`, { method: 'POST', body: JSON.stringify(input), successMessage: 'Milestone invoice created successfully.' }),
}
