import { zodResolver } from '@hookform/resolvers/zod'
import { MailCheck } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link } from 'react-router-dom'
import { Button } from '@/shared/ui/Button'
import { Input } from '@/shared/ui/FormControls'
import { authApi } from './api'
import { AuthLayout } from './AuthLayout'
import { passwordResetRequestSchema, type PasswordResetRequestFormValues } from './schema'
import type { PasswordResetRequestResponse } from './types'

export function ForgotPasswordPage() {
  const [result, setResult] = useState<PasswordResetRequestResponse | null>(null)
  const form = useForm<PasswordResetRequestFormValues>({ resolver: zodResolver(passwordResetRequestSchema), defaultValues: { email: '' } })
  const submit = form.handleSubmit(async (values) => {
    try {
      setResult(await authApi.requestPasswordReset(values))
    } catch (error) {
      form.setError('root', { message: error instanceof Error ? error.message : 'Password reset request failed.' })
    }
  })

  return (
    <AuthLayout eyebrow="Account recovery" title="Reset your password" description="Enter your email and we’ll send you a secure, single-use reset link.">
      {result ? (
        <div className="auth-result" role="status">
          <MailCheck size={24} />
          <p>{result.message}</p>
          {result.resetUrl && <a className="auth-reset-link" href={result.resetUrl}>Open local reset link</a>}
        </div>
      ) : (
        <form onSubmit={submit} className="login-form" noValidate>
          <Input label="Email address" type="email" autoComplete="email" error={form.formState.errors.email?.message} {...form.register('email')} />
          {form.formState.errors.root?.message && <p className="login-error" role="alert">{form.formState.errors.root.message}</p>}
          <Button type="submit" full disabled={form.formState.isSubmitting}>{form.formState.isSubmitting ? 'Sending reset link' : 'Send reset link'}</Button>
        </form>
      )}
      <p className="auth-switch"><Link to="/login">Back to sign in</Link></p>
    </AuthLayout>
  )
}
