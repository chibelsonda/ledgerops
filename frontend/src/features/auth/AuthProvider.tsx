import { useQuery, useQueryClient } from '@tanstack/react-query'
import { createContext, type ReactNode, useCallback, useContext, useEffect } from 'react'
import { ApiError, setCSRFToken } from '../../shared/api/http'
import { authApi } from './api'
import type { LoginCredentials, Session } from './types'

const sessionQueryKey = ['auth', 'session'] as const

interface AuthContextValue {
  session: Session | null
  isLoading: boolean
  login: (credentials: LoginCredentials) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const sessionQuery = useQuery({
    queryKey: sessionQueryKey,
    queryFn: async () => {
      try {
        return await authApi.session()
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) return null
        throw error
      }
    },
    retry: false,
    staleTime: 5 * 60_000,
  })

  const login = useCallback(async (credentials: LoginCredentials) => {
    const session = await authApi.login(credentials)
    queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== 'auth' })
    queryClient.setQueryData(sessionQueryKey, session)
  }, [queryClient])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } finally {
      queryClient.setQueryData(sessionQueryKey, null)
      queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== 'auth' })
    }
  }, [queryClient])

  useEffect(() => {
    const handleUnauthorized = () => {
      setCSRFToken('')
      queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== 'auth' })
      queryClient.setQueryData(sessionQueryKey, null)
    }
    window.addEventListener('ledgerops:unauthorized', handleUnauthorized)
    return () => window.removeEventListener('ledgerops:unauthorized', handleUnauthorized)
  }, [queryClient])

  return (
    <AuthContext.Provider value={{ session: sessionQuery.data ?? null, isLoading: sessionQuery.isLoading, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) throw new Error('useAuth must be used inside AuthProvider')
  return context
}
