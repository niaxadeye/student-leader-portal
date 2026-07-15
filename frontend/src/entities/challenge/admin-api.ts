// Админ-API конструктора испытаний (бэкенд: modules/challenges).
import { apiRequest } from '@/shared/api/client'
import type { AdminChallenge, AdminField, ChallengeInput, FieldInput } from './admin-types'

// ── Испытания ────────────────────────────────────────────────────────────
export function listChallenges(contestId: string): Promise<AdminChallenge[]> {
  return apiRequest<AdminChallenge[]>(`/admin/contests/${contestId}/challenges`)
}

export function getChallenge(challengeId: string): Promise<AdminChallenge> {
  return apiRequest<AdminChallenge>(`/admin/challenges/${challengeId}`)
}

export function createChallenge(contestId: string, input: ChallengeInput): Promise<AdminChallenge> {
  return apiRequest<AdminChallenge>(`/admin/contests/${contestId}/challenges`, {
    method: 'POST',
    body: input,
  })
}

export function updateChallenge(challengeId: string, input: ChallengeInput): Promise<AdminChallenge> {
  return apiRequest<AdminChallenge>(`/admin/challenges/${challengeId}`, {
    method: 'PATCH',
    body: input,
  })
}

export function duplicateChallenge(challengeId: string): Promise<AdminChallenge> {
  return apiRequest<AdminChallenge>(`/admin/challenges/${challengeId}/duplicate`, { method: 'POST' })
}

export function transitionChallenge(
  challengeId: string,
  action: 'publish' | 'close' | 'archive',
): Promise<AdminChallenge> {
  return apiRequest<AdminChallenge>(`/admin/challenges/${challengeId}/${action}`, { method: 'POST' })
}

// ── Поля ─────────────────────────────────────────────────────────────────
export function listFields(challengeId: string): Promise<AdminField[]> {
  return apiRequest<AdminField[]>(`/admin/challenges/${challengeId}/fields`)
}

export function addField(challengeId: string, input: FieldInput): Promise<AdminField> {
  return apiRequest<AdminField>(`/admin/challenges/${challengeId}/fields`, {
    method: 'POST',
    body: input,
  })
}

export function updateField(challengeId: string, fieldId: string, input: FieldInput): Promise<void> {
  return apiRequest(`/admin/challenges/${challengeId}/fields/${fieldId}`, {
    method: 'PATCH',
    body: input,
  })
}

export function deleteField(challengeId: string, fieldId: string): Promise<void> {
  return apiRequest(`/admin/challenges/${challengeId}/fields/${fieldId}`, { method: 'DELETE' })
}

export function reorderFields(challengeId: string, fieldIds: string[]): Promise<void> {
  return apiRequest(`/admin/challenges/${challengeId}/fields/reorder`, {
    method: 'PATCH',
    body: { field_ids: fieldIds },
  })
}
