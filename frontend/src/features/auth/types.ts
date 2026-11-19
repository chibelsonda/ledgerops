export type Role = 'admin' | 'controller' | 'reviewer'

export interface AuthenticatedUser {
  userId: string
  organizationId: string
  organizationName: string
  email: string
  displayName: string
  role: Role
}

export interface Session {
  user: AuthenticatedUser
  csrfToken: string
  expiresAt: string
}

export interface LoginCredentials {
  email: string
  password: string
}

export interface RegistrationInput {
  organizationName: string
  displayName: string
  email: string
  password: string
}

export interface PasswordResetRequestInput {
  email: string
}

export interface PasswordResetConfirmInput {
  token: string
  newPassword: string
}

export interface PasswordResetRequestResponse {
  message: string
  resetUrl?: string
}
