import { apiRequest } from './client'
import type { LoginRequest, RegisterRequest, UserDTO } from './types'

export function register(input: RegisterRequest): Promise<UserDTO> {
  return apiRequest<UserDTO>('/api/auth/register', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function login(input: LoginRequest): Promise<UserDTO> {
  return apiRequest<UserDTO>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function logout(): Promise<void> {
  return apiRequest<void>('/api/auth/logout', { method: 'POST' })
}

export function getCurrentUser(): Promise<UserDTO> {
  return apiRequest<UserDTO>('/api/me')
}

export function startGoogleLogin(): void {
  window.location.assign('/api/auth/google/start')
}

export async function requestPasswordReset(email: string): Promise<void> {
  await apiRequest<unknown>('/api/auth/password/forgot', {
    method: 'POST',
    body: JSON.stringify({ email }),
  })
}

export function resetPassword(token: string, newPassword: string): Promise<void> {
  return apiRequest<void>('/api/auth/password/reset', {
    method: 'POST',
    body: JSON.stringify({ token, newPassword }),
  })
}

export function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  return apiRequest<void>(
    '/api/auth/password/change',
    {
      method: 'POST',
      body: JSON.stringify({ currentPassword, newPassword }),
    },
    { authenticated: true },
  )
}
