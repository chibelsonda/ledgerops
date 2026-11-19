import { z } from 'zod'

export const loginSchema = z.object({
  email: z.email('Enter a valid email address.'),
  password: z.string().min(12, 'Password must contain at least 12 characters.').max(128, 'Password must not exceed 128 characters.'),
})

export type LoginFormValues = z.infer<typeof loginSchema>

const passwordSchema = z.string().min(12, 'Password must contain at least 12 characters.').max(128, 'Password must not exceed 128 characters.')

export const registrationSchema = z.object({
  organizationName: z.string().trim().min(2, 'Workspace name must contain at least 2 characters.').max(120),
  displayName: z.string().trim().min(2, 'Your name must contain at least 2 characters.').max(120),
  email: z.email('Enter a valid email address.'),
  password: passwordSchema,
  confirmPassword: z.string(),
}).refine((values) => values.password === values.confirmPassword, {
  path: ['confirmPassword'],
  message: 'Passwords do not match.',
})

export const passwordResetRequestSchema = z.object({
  email: z.email('Enter a valid email address.'),
})

export const passwordResetSchema = z.object({
  password: passwordSchema,
  confirmPassword: z.string(),
}).refine((values) => values.password === values.confirmPassword, {
  path: ['confirmPassword'],
  message: 'Passwords do not match.',
})

export type RegistrationFormValues = z.infer<typeof registrationSchema>
export type PasswordResetRequestFormValues = z.infer<typeof passwordResetRequestSchema>
export type PasswordResetFormValues = z.infer<typeof passwordResetSchema>
