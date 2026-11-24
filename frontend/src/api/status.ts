import { apiRequest } from './client'
import type { PublicStatusDTO } from './types'

export function getPublicStatus(slug: string): Promise<PublicStatusDTO> {
  return apiRequest<PublicStatusDTO>(`/api/public/status/${encodeURIComponent(slug)}`)
}
