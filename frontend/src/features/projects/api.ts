import { apiRequest } from '@/shared/api/http'
import type { ChangeOrderInput, ClientOption, CreateProjectInput, DecisionInput, MilestoneInput, MilestoneInvoiceInput, ProjectDetail, ProjectSummary } from './types'

export const projectsApi = {
  list: async () => (await apiRequest<{ items: ProjectSummary[] }>('/api/v1/projects')).items,
  clients: async () => (await apiRequest<{ items: ClientOption[] }>('/api/v1/projects/clients')).items,
  get: (projectId: string) => apiRequest<ProjectDetail>(`/api/v1/projects/${projectId}`),
  create: (input: CreateProjectInput) => apiRequest<ProjectDetail>('/api/v1/projects', { method: 'POST', body: JSON.stringify(input), successMessage: 'Project created successfully.' }),
  decideContract: (contractId: string, input: DecisionInput) => apiRequest<ProjectDetail>(`/api/v1/contracts/${contractId}/decision`, { method: 'PATCH', body: JSON.stringify(input), successMessage: 'Contract decision saved.' }),
  createChangeOrder: (contractId: string, input: ChangeOrderInput) => apiRequest<ProjectDetail>(`/api/v1/contracts/${contractId}/change-orders`, { method: 'POST', body: JSON.stringify(input), successMessage: 'Change order created successfully.' }),
  decideChangeOrder: (changeOrderId: string, input: DecisionInput) => apiRequest<ProjectDetail>(`/api/v1/change-orders/${changeOrderId}/decision`, { method: 'PATCH', body: JSON.stringify(input), successMessage: 'Change-order decision saved.' }),
  createMilestone: (contractId: string, input: MilestoneInput) => apiRequest<ProjectDetail>(`/api/v1/contracts/${contractId}/milestones`, { method: 'POST', body: JSON.stringify(input), successMessage: 'Milestone created successfully.' }),
  invoiceMilestone: (milestoneId: string, input: MilestoneInvoiceInput) => apiRequest<ProjectDetail>(`/api/v1/milestones/${milestoneId}/invoice`, { method: 'POST', body: JSON.stringify(input), successMessage: 'Milestone invoice created successfully.' }),
}
