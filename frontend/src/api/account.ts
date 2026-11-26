import { apiRequest } from './client'
import type { NotificationChannel, NotifyDTO, UserDTO } from './types'

export function updateNotifications(input: NotifyDTO & { telegramStartNonce?: string }): Promise<UserDTO> {
  return apiRequest<UserDTO>('/api/me/notifications', {
    method: 'PUT',
    body: JSON.stringify(input),
  }, { authenticated: true })
}

export function updateProfile(name: string): Promise<UserDTO> {
  return apiRequest<UserDTO>('/api/me/profile', {
    method: 'PUT',
    body: JSON.stringify({ name }),
  }, { authenticated: true })
}

export function updateStatusSlug(statusSlug: string): Promise<UserDTO> {
  return apiRequest<UserDTO>('/api/me/status-slug', {
    method: 'PUT',
    body: JSON.stringify({ statusSlug }),
  }, { authenticated: true })
}

export function sendTestAlert(channel: NotificationChannel): Promise<void> {
  return apiRequest<void>(`/api/me/notifications/test/${channel}`, {
    method: 'POST',
  }, { authenticated: true })
}
