import { apiRequest } from './client'
import type {
  CheckDTO,
  IncidentDTO,
  MonitorDTO,
  MonitorDetailDTO,
  MonitorListDTO,
  MonitorRequest,
} from './types'

export function listMonitors(): Promise<MonitorListDTO[]> {
  return apiRequest<MonitorListDTO[]>('/api/monitors', {}, { authenticated: true })
}

export function createMonitor(input: MonitorRequest): Promise<MonitorDTO> {
  return apiRequest<MonitorDTO>('/api/monitors', {
    method: 'POST',
    body: JSON.stringify(input),
  }, { authenticated: true })
}

export function getMonitor(id: string): Promise<MonitorDetailDTO> {
  return apiRequest<MonitorDetailDTO>(monitorPath(id), {}, { authenticated: true })
}

export function updateMonitor(id: string, input: MonitorRequest): Promise<MonitorDTO> {
  return apiRequest<MonitorDTO>(monitorPath(id), {
    method: 'PUT',
    body: JSON.stringify(input),
  }, { authenticated: true })
}

export function deleteMonitor(id: string): Promise<void> {
  return apiRequest<void>(monitorPath(id), { method: 'DELETE' }, { authenticated: true })
}

export function pauseMonitor(id: string): Promise<MonitorDTO> {
  return apiRequest<MonitorDTO>(`${monitorPath(id)}/pause`, { method: 'POST' }, { authenticated: true })
}

export function resumeMonitor(id: string): Promise<MonitorDTO> {
  return apiRequest<MonitorDTO>(`${monitorPath(id)}/resume`, { method: 'POST' }, { authenticated: true })
}

export function listMonitorChecks(id: string, limit?: number): Promise<CheckDTO[]> {
  return apiRequest<CheckDTO[]>(withLimit(`${monitorPath(id)}/checks`, limit), {}, { authenticated: true })
}

export function listMonitorIncidents(id: string, limit?: number): Promise<IncidentDTO[]> {
  return apiRequest<IncidentDTO[]>(withLimit(`${monitorPath(id)}/incidents`, limit), {}, { authenticated: true })
}

function monitorPath(id: string): string {
  return `/api/monitors/${encodeURIComponent(id)}`
}

function withLimit(path: string, limit?: number): string {
  if (limit === undefined) {
    return path
  }
  const query = new URLSearchParams({ limit: String(limit) })
  return `${path}?${query}`
}
