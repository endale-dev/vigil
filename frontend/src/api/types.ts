export type MonitorType = 'http' | 'tcp' | 'tls'

export type MonitorStatus = 'unknown' | 'up' | 'down'

export type OverallStatus = 'operational' | 'degraded' | 'unknown'

export type NotificationChannel = 'email' | 'telegram'

export interface RegisterRequest {
  name: string
  email: string
  password: string
}

export interface LoginRequest {
  email: string
  password: string
}

export interface MonitorRequest {
  name: string
  type: MonitorType
  target: string
  intervalSeconds: number
  timeoutSeconds: number
  expectedStatus: number
  tlsWarnDays: number
  public: boolean
}

export interface ErrorDTO {
  error: string
  field?: string
}

export interface CapabilitiesDTO {
  google: boolean
  telegram: boolean
}

export interface NotifyDTO {
  emailAlerts: boolean
  telegramChatId: string
}

export interface UserDTO {
  id: string
  name: string
  email: string
  hasPassword: boolean
  googleLinked: boolean
  statusSlug: string
  notify: NotifyDTO
  capabilities: CapabilitiesDTO
}

export interface MonitorDTO {
  id: string
  name: string
  type: MonitorType
  target: string
  intervalSeconds: number
  timeoutSeconds: number
  expectedStatus: number
  tlsWarnDays: number
  public: boolean
  enabled: boolean
  status: MonitorStatus
  lastCheckedAt: string | null
  createdAt: string
}

export type MonitorListDTO = MonitorDTO & {
  uptime24Hours: UptimeSummaryDTO
}

export type MonitorDetailDTO = MonitorDTO & {
  uptime: MonitorUptimeDTO
}

export interface CheckDTO {
  id: string
  checkedAt: string
  up: boolean
  latencyMs: number
  detail?: string
}

export interface IncidentDTO {
  id: string
  startedAt: string
  resolvedAt: string | null
  cause: string
}

export interface UptimeSummaryDTO {
  up: number
  total: number
  ratio: number | null
}

export interface MonitorUptimeDTO {
  last24Hours: UptimeSummaryDTO
  last7Days: UptimeSummaryDTO
  last30Days: UptimeSummaryDTO
}

export interface PublicStatusDTO {
  status: OverallStatus
  monitors: PublicStatusMonitorDTO[]
}

export interface PublicStatusMonitorDTO {
  name: string
  type: MonitorType
  status: MonitorStatus
  lastCheckedAt: string | null
  uptime: MonitorUptimeDTO
  checks: PublicCheckDTO[]
}

export interface PublicCheckDTO {
  up: boolean
  latencyMs: number
}
