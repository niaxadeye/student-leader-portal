import { apiRequest } from '@/shared/api/client'
import type {
  AdminContest,
  ContestStatus,
  CreateContestInput,
  UpdateContestInput,
} from './types'

/** Список конкурсов в области видимости (scope применяется на бэкенде). */
export function listContests(status?: string): Promise<AdminContest[]> {
  const qs = status ? `?status=${encodeURIComponent(status)}` : ''
  return apiRequest<AdminContest[]>(`/admin/contests${qs}`)
}

export function getContest(id: string): Promise<AdminContest> {
  return apiRequest<AdminContest>(`/admin/contests/${id}`)
}

export function createContest(input: CreateContestInput): Promise<AdminContest> {
  return apiRequest<AdminContest>('/admin/contests', { method: 'POST', body: input })
}

export function updateContest(id: string, input: UpdateContestInput): Promise<AdminContest> {
  return apiRequest<AdminContest>(`/admin/contests/${id}`, { method: 'PATCH', body: input })
}

/** Переход статуса: publish | finish | archive. */
export function transitionContest(
  id: string,
  action: 'publish' | 'finish' | 'archive',
): Promise<AdminContest> {
  return apiRequest<AdminContest>(`/admin/contests/${id}/${action}`, { method: 'POST' })
}

/** Целевой статус для каждого перехода (для оптимистичных подписей). */
export const transitionTarget: Record<string, ContestStatus> = {
  publish: 'ACTIVE',
  finish: 'FINISHED',
  archive: 'ARCHIVED',
}
