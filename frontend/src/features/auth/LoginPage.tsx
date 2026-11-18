import { zodResolver } from '@hookform/resolvers/zod'
import { LockKeyhole, ShieldCheck } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { Button } from '@/shared/ui/Button'
import { Input } from '@/shared/ui/FormControls'
import { LedgerOpsMark } from '@/shared/ui/LedgerOpsMark'
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
    <main className="login-page">
      <section className="login-story" aria-label="LedgerOps introduction">
        <div className="login-brand"><span className="brand-mark"><LedgerOpsMark /></span><span>LedgerOps</span></div>
        <div>
          <span className="eyebrow">Financial command center</span>
          <h1>Decisions start with numbers you can trust.</h1>
          <p>Review cash, receivables, budgets, and exceptions from one focused workspace.</p>
        </div>
        <div className="login-security"><ShieldCheck size={18} /><span>Protected by server-side sessions and role-based access.</span></div>
      </section>
      <section className="login-panel">
        <div className="login-card">
          <span className="eyebrow">Welcome back</span>
          <h2>Sign in to LedgerOps</h2>
          <p>Use your assigned workspace account.</p>
          <form onSubmit={submit} className="login-form" noValidate>
            <Input label="Email address" type="email" autoComplete="email" error={errors.email?.message} {...register('email')} />
            <Input label="Password" type="password" autoComplete="current-password" error={errors.password?.message} {...register('password')} />
            {errors.root?.message && <p className="login-error" role="alert">{errors.root.message}</p>}
            <Button type="submit" full disabled={isSubmitting}>
              {isSubmitting ? <LockKeyhole size={15} className="spin" /> : <LockKeyhole size={15} />}
              {isSubmitting ? 'Signing in' : 'Sign in securely'}
            </Button>
          </form>
        </div>
      </section>
    </main>
  )
}
