import { apiFormRequest, apiRequest, setCSRFToken } from '@/shared/api/http'
import type { LoginCredentials, Session } from './types'

function rememberCSRF(session: Session) {
  setCSRFToken(session.csrfToken)
  return session
}

export const authApi = {
  session: async () => rememberCSRF(await apiRequest<Session>('/auth/session', { suppressToast: true })),
  login: async (credentials: LoginCredentials) => rememberCSRF(await apiFormRequest<Session>('/auth/login', {
    method: 'POST',
    body: JSON.stringify(credentials),
    successMessage: 'Signed in successfully.',
  })),
  logout: async () => {
    await apiRequest<void>('/auth/logout', { method: 'POST', successMessage: 'Signed out successfully.' })
    setCSRFToken('')
  },
}
