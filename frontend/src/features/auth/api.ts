import { apiFormRequest, apiRequest, setCSRFToken } from '@/shared/api/http'
import type { LoginCredentials, PasswordResetConfirmInput, PasswordResetRequestInput, PasswordResetRequestResponse, RegistrationInput, Session } from './types'

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
  register: async (input: RegistrationInput) => rememberCSRF(await apiFormRequest<Session>('/auth/register', {
    method: 'POST',
    body: JSON.stringify(input),
    successMessage: 'Your account and workspace are ready.',
  })),
  requestPasswordReset: (input: PasswordResetRequestInput) => apiFormRequest<PasswordResetRequestResponse>('/auth/password-reset/request', {
    method: 'POST',
    body: JSON.stringify(input),
    successMessage: 'Password reset request received.',
  }),
  resetPassword: (input: PasswordResetConfirmInput) => apiFormRequest<void>('/auth/password-reset/confirm', {
    method: 'POST',
    body: JSON.stringify(input),
    successMessage: 'Password reset successfully.',
  }),
  logout: async () => {
    await apiRequest<void>('/auth/logout', { method: 'POST', successMessage: 'Signed out successfully.' })
    setCSRFToken('')
  },
}
