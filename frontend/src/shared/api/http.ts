import { emitToast } from '@/shared/lib/toast'

const API_BASE_PATH = '/api/v1'

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
  suppressErrorToast?: boolean
}

type FormRequestOptions = Omit<ApiRequestOptions, 'suppressErrorToast'>

function defaultSuccessMessage(method: string) {
  if (method === 'DELETE') return 'Deleted successfully.'
  if (method === 'PATCH' || method === 'PUT') return 'Changes saved successfully.'
  return 'Request completed successfully.'
}

function apiPath(path: string) {
  return `${API_BASE_PATH}${path.startsWith('/') ? path : `/${path}`}`
}

export async function apiRequest<T>(path: string, options?: ApiRequestOptions): Promise<T> {
  const method = options?.method?.toUpperCase() ?? 'GET'
  const requiresCSRF = !['GET', 'HEAD', 'OPTIONS'].includes(method)
  const { successMessage, suppressToast, suppressErrorToast, ...requestOptions } = options ?? {}
  let response: Response
  try {
    response = await fetch(apiPath(path), {
      ...requestOptions,
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
        ...(requiresCSRF && csrfToken ? { 'X-CSRF-Token': csrfToken } : {}),
        ...requestOptions.headers,
      },
    })
  } catch (error) {
    if (!(error instanceof DOMException && error.name === 'AbortError') && !suppressToast && !suppressErrorToast) {
      emitToast({ tone: 'error', message: 'Unable to reach the server. Check your connection and try again.' })
    }
    throw error
  }
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ApiErrorBody
    const message = body.error?.message ?? `Request failed with status ${response.status}`
    if (!suppressToast && !suppressErrorToast) emitToast({ tone: 'error', message })
    if (response.status === 401 && !path.startsWith('/auth/')) {
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

export function apiFormRequest<T>(path: string, options?: FormRequestOptions): Promise<T> {
  return apiRequest<T>(path, { ...options, suppressErrorToast: true })
}
