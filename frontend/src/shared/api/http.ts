import { emitToast } from '@/shared/lib/toast'

let csrfToken = ''

export function setCSRFToken(value: string) {
  csrfToken = value
}

export class ApiError extends Error {
  constructor(message: string, readonly status: number) {
    super(message)
    this.name = 'ApiError'
  }
}

interface ApiErrorBody {
  error?: { message?: string }
}

interface ApiRequestOptions extends RequestInit {
  successMessage?: string
  suppressToast?: boolean
}

function defaultSuccessMessage(method: string) {
  if (method === 'DELETE') return 'Deleted successfully.'
  if (method === 'PATCH' || method === 'PUT') return 'Changes saved successfully.'
  return 'Request completed successfully.'
}

export async function apiRequest<T>(path: string, options?: ApiRequestOptions): Promise<T> {
  const method = options?.method?.toUpperCase() ?? 'GET'
  const requiresCSRF = !['GET', 'HEAD', 'OPTIONS'].includes(method)
  const { successMessage, suppressToast, ...requestOptions } = options ?? {}
  let response: Response
  try {
    response = await fetch(path, {
      ...requestOptions,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...(requiresCSRF && csrfToken ? { 'X-CSRF-Token': csrfToken } : {}),
        ...requestOptions.headers,
      },
    })
  } catch (error) {
    if (!(error instanceof DOMException && error.name === 'AbortError') && !suppressToast) {
      emitToast({ tone: 'error', message: 'Unable to reach the server. Check your connection and try again.' })
    }
    throw error
  }
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ApiErrorBody
    const message = body.error?.message ?? `Request failed with status ${response.status}`
    if (!suppressToast) emitToast({ tone: 'error', message })
    if (response.status === 401 && !path.startsWith('/api/v1/auth/')) {
      setCSRFToken('')
      window.dispatchEvent(new Event('ledgerops:unauthorized'))
    }
    throw new ApiError(message, response.status)
  }
  const result = response.status === 204 ? undefined as T : await response.json() as T
  if (requiresCSRF && !suppressToast) {
    emitToast({ tone: 'success', message: successMessage ?? defaultSuccessMessage(method) })
  }
  return result
}
