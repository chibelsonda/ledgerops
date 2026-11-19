import { zodResolver } from '@hookform/resolvers/zod'
import { KeyRound } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useSearchParams } from 'react-router-dom'
import { Button } from '@/shared/ui/Button'
import { PasswordInput } from '@/shared/ui/FormControls'
import { authApi } from './api'
import { AuthLayout } from './AuthLayout'
import { passwordResetSchema, type PasswordResetFormValues } from './schema'

export function ResetPasswordPage() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const [complete, setComplete] = useState(false)
  const form = useForm<PasswordResetFormValues>({ resolver: zodResolver(passwordResetSchema), defaultValues: { password: '', confirmPassword: '' } })
  const submit = form.handleSubmit(async (values) => {
    try {
      await authApi.resetPassword({ token, newPassword: values.password })
      setComplete(true)
    } catch (error) {
      form.setError('root', { message: error instanceof Error ? error.message : 'Password reset failed.' })
    }
  })

  return (
    <AuthLayout eyebrow="Account recovery" title="Choose a new password" description="Use at least 12 characters. This reset link can only be used once.">
      {!token ? (
        <p className="login-error" role="alert">This password reset link is missing its security token.</p>
      ) : complete ? (
        <div className="auth-result" role="status"><KeyRound size={24} /><p>Your password has been reset. You can now sign in.</p></div>
      ) : (
        <form onSubmit={submit} className="login-form" noValidate>
          <PasswordInput label="New password" autoComplete="new-password" error={form.formState.errors.password?.message} {...form.register('password')} />
          <PasswordInput label="Confirm new password" autoComplete="new-password" error={form.formState.errors.confirmPassword?.message} {...form.register('confirmPassword')} />
          {form.formState.errors.root?.message && <p className="login-error" role="alert">{form.formState.errors.root.message}</p>}
          <Button type="submit" full disabled={form.formState.isSubmitting}>{form.formState.isSubmitting ? 'Resetting password' : 'Reset password'}</Button>
        </form>
      )}
      <p className="auth-switch"><Link to="/login">Back to sign in</Link></p>
    </AuthLayout>
  )
}
