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

  const body = await response.text()
  let payload: unknown
  if (body.trim() !== '') {
    try {
      payload = JSON.parse(body) as unknown
    } catch {
      if (response.ok) {
        throw new ApiError(response.status, 'The server returned an invalid response.')
      }
      payload = undefined
    }
  }

  if (!response.ok) {
    if (response.status === 401 && options.authenticated) {
      handleUnauthorized()
    }
    const error = asErrorDTO(payload)
    throw new ApiError(
      response.status,
      error?.error ?? (body.trim() || `Request failed with status ${response.status}.`),
      error?.field,
    )
  }

  if (payload === undefined) {
    throw new ApiError(response.status, 'The server returned an empty response.')
  }

  return payload as T
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
