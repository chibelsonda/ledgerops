import { zodResolver } from '@hookform/resolvers/zod'
import { LockKeyhole } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Link } from 'react-router-dom'
import { Button } from '@/shared/ui/Button'
import { Input, PasswordInput } from '@/shared/ui/FormControls'
import { AuthLayout } from './AuthLayout'
import { useAuth } from './AuthProvider'
import { loginSchema, type LoginFormValues } from './schema'

export function LoginPage() {
  const { login } = useAuth()
  const { register, handleSubmit, setError, formState: { errors, isSubmitting } } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  })

  const submit = handleSubmit(async (values) => {
    try {
      await login(values)
    } catch (error) {
      setError('root', { message: error instanceof Error ? error.message : 'Sign in failed.' })
    }
  })

  return (
    <AuthLayout eyebrow="Welcome back" title="Sign in to LedgerOps" description="Use your workspace account to continue.">
      <form onSubmit={submit} className="login-form" noValidate>
            <Input label="Email address" type="email" autoComplete="email" error={errors.email?.message} {...register('email')} />
            <PasswordInput label="Password" autoComplete="current-password" error={errors.password?.message} {...register('password')} />
            <div className="auth-form-link"><Link to="/forgot-password">Forgot password?</Link></div>
            {errors.root?.message && <p className="login-error" role="alert">{errors.root.message}</p>}
            <Button type="submit" full disabled={isSubmitting}>
              {isSubmitting ? <LockKeyhole size={15} className="spin" /> : <LockKeyhole size={15} />}
              {isSubmitting ? 'Signing in' : 'Sign in securely'}
            </Button>
          </form>
      <p className="auth-switch">New to LedgerOps? <Link to="/register">Create an account</Link></p>
    </AuthLayout>
  )
}
