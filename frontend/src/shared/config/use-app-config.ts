import { useQuery } from '@tanstack/react-query'
import { apiRequest } from '@/shared/api/client'

export interface Features {
  reference_cms: boolean
  email_notifications: boolean
  participant_cabinet: boolean
  attendance: boolean
  points: boolean
  merch: boolean
  predictions: boolean
  jury: boolean
}

export interface AppConfig {
  app_name: string
  env: string
  features: Features
}

/** Публичная конфигурация из /api/v1/config (feature flags — источник истины backend). */
export function useAppConfig() {
  return useQuery({
    queryKey: ['app-config'],
    queryFn: () => apiRequest<AppConfig>('/config'),
    staleTime: Infinity,
  })
}
