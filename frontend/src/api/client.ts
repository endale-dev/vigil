import type { ErrorDTO } from './types'

type ApiRequestOptions = {
  authenticated?: boolean
}

type UnauthorizedHandler = () => void

let unauthorizedHandler: UnauthorizedHandler | undefined
let redirectingToLogin = false

export class ApiError extends Error {
  readonly status: number
  readonly field?: string

  constructor(status: number, message: string, field?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.field = field
  }
}

export function setUnauthorizedHandler(handler: UnauthorizedHandler): () => void {
  unauthorizedHandler = handler
  return () => {
    if (unauthorizedHandler === handler) {
      unauthorizedHandler = undefined
    }
  }
}

export async function apiRequest<T>(
  path: string,
  init: RequestInit = {},
  options: ApiRequestOptions = {},
): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Accept')) {
    headers.set('Accept', 'application/json')
  }
  if (init.body !== undefined && init.body !== null && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  let response: Response
  try {
    response = await fetch(path, {
      ...init,
      headers,
      credentials: 'same-origin',
    })
  } catch (error) {
    if (error instanceof Error && error.name === 'AbortError') {
      throw error
    }
    throw new ApiError(0, 'Unable to connect to the server.')
  }

  if (response.status === 204) {
    return undefined as T
  }

  const body = await response.text().catch(() => {
    throw new ApiError(0, 'The connection was interrupted. Please try again.')
  })
  let payload: unknown
  if (body.trim() !== '') {
    try {
      payload = JSON.parse(body) as unknown
    } catch {
      if (response.ok) {
        throw new ApiError(response.status, 'Vigil could not complete this request. Please try again.')
      }
      payload = undefined
    }
  }

  if (!response.ok) {
    if (response.status === 401 && options.authenticated) {
      handleUnauthorized()
    }
    throw publicError(response.status, asErrorDTO(payload))
  }

  if (payload === undefined) {
    throw new ApiError(response.status, 'Vigil could not complete this request. Please try again.')
  }

  return payload as T
}

function publicError(status: number, detail?: ErrorDTO): ApiError {
  if (status === 422 && detail?.field) return new ApiError(status, detail.error, detail.field)
  const messages: Record<number, string> = {
    400: 'Check your entries and try again.',
    401: detail?.error === 'invalid credentials' ? 'Check your email and password.' : 'Please sign in again.',
    403: detail?.error === 'limit reached' ? 'You have reached your monitor limit.' : 'You do not have permission to do that.',
    404: 'This item is unavailable.',
    409: detail?.error === 'email already taken' ? 'This email is already registered.' : 'This value is already in use.',
    413: 'This request is too large. Shorten your entry and try again.',
    429: 'Please wait a moment before trying again.',
  }
  return new ApiError(status, messages[status] ?? 'Vigil could not complete this request. Please try again.')
}

function handleUnauthorized(): void {
  unauthorizedHandler?.()
  if (typeof window !== 'undefined' && window.location.pathname !== '/login' && !redirectingToLogin) {
    redirectingToLogin = true
    window.location.replace('/login')
  }
}

function asErrorDTO(value: unknown): ErrorDTO | undefined {
  if (typeof value !== 'object' || value === null || !('error' in value) || typeof value.error !== 'string') {
    return undefined
  }
  const field = 'field' in value && typeof value.field === 'string' ? value.field : undefined
  return { error: value.error, field }
}
