import type { Role } from '@/features/auth/types'

export const capabilities = {
  operate: (role?: Role) => role === 'admin' || role === 'controller',
  administer: (role?: Role) => role === 'admin',
  createClosePeriod: (role?: Role) => role === 'controller',
} as const
