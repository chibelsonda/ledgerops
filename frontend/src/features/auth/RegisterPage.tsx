import { zodResolver } from '@hookform/resolvers/zod'
import { UserPlus } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Link } from 'react-router-dom'
import { Button } from '@/shared/ui/Button'
import { Input, PasswordInput } from '@/shared/ui/FormControls'
import { AuthLayout } from './AuthLayout'
import { useAuth } from './AuthProvider'
import { registrationSchema, type RegistrationFormValues } from './schema'

export function RegisterPage() {
  const { registerAccount } = useAuth()
  const form = useForm<RegistrationFormValues>({
    resolver: zodResolver(registrationSchema),
    defaultValues: { organizationName: '', displayName: '', email: '', password: '', confirmPassword: '' },
  })
  const submit = form.handleSubmit(async ({ confirmPassword: _confirmPassword, ...values }) => {
    try {
      await registerAccount(values)
    } catch (error) {
      form.setError('root', { message: error instanceof Error ? error.message : 'Account creation failed.' })
    }
  })

  return (
    <AuthLayout eyebrow="Get started" title="Create your account" description="Create a secure workspace for your financial operations.">
      <form onSubmit={submit} className="login-form" noValidate>
        <Input label="Workspace name" autoComplete="organization" error={form.formState.errors.organizationName?.message} {...form.register('organizationName')} />
        <Input label="Your name" autoComplete="name" error={form.formState.errors.displayName?.message} {...form.register('displayName')} />
        <Input label="Email address" type="email" autoComplete="email" error={form.formState.errors.email?.message} {...form.register('email')} />
        <PasswordInput label="Password" autoComplete="new-password" error={form.formState.errors.password?.message} {...form.register('password')} />
        <PasswordInput label="Confirm password" autoComplete="new-password" error={form.formState.errors.confirmPassword?.message} {...form.register('confirmPassword')} />
        {form.formState.errors.root?.message && <p className="login-error" role="alert">{form.formState.errors.root.message}</p>}
        <Button type="submit" full disabled={form.formState.isSubmitting}><UserPlus size={15} />{form.formState.isSubmitting ? 'Creating account' : 'Create account'}</Button>
      </form>
      <p className="auth-switch">Already have an account? <Link to="/login">Sign in</Link></p>
    </AuthLayout>
  )
}
