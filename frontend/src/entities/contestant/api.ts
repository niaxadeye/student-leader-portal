import { apiRequest, apiGetText, apiPostText } from '@/shared/api/client'
import type {
  AddContestantInput,
  AddContestantResult,
  Contestant,
  ImportResult,
} from './types'

export function listContestants(contestId: string): Promise<Contestant[]> {
  return apiRequest<Contestant[]>(`/admin/contests/${contestId}/contestants`)
}

export function addContestant(
  contestId: string,
  input: AddContestantInput,
): Promise<AddContestantResult> {
  return apiRequest<AddContestantResult>(`/admin/contests/${contestId}/contestants`, {
    method: 'POST',
    body: input,
  })
}

export function removeContestant(contestId: string, userId: string): Promise<void> {
  return apiRequest(`/admin/contests/${contestId}/contestants/${userId}`, { method: 'DELETE' })
}

/** Импорт CSV (login,full_name,organization). Возвращает построчную сводку. */
export function importContestants(contestId: string, csv: string): Promise<ImportResult> {
  return apiPostText<ImportResult>(`/admin/contests/${contestId}/contestants/import`, csv)
}

/** Экспорт активных конкурсантов в CSV (текст). */
export function exportContestants(contestId: string): Promise<string> {
  return apiGetText(`/admin/contests/${contestId}/contestants/export`)
}
