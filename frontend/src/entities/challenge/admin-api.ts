// Админ-API конструктора испытаний (бэкенд: modules/challenges).
import { apiPostForm, apiRequest } from '@/shared/api/client'
import type {
  AdminBriefing,
  AdminChallenge,
  AdminField,
  BriefingInput,
  ChallengeInput,
  FieldInput,
  OverrideInput,
} from './admin-types'

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

export function getChallengeBriefing(challengeId: string): Promise<AdminBriefing> {
  return apiRequest<AdminBriefing>(`/admin/challenges/${challengeId}/briefing`)
}

export function saveChallengeBriefing(challengeId: string, input: BriefingInput): Promise<AdminBriefing> {
  return apiRequest<AdminBriefing>(`/admin/challenges/${challengeId}/briefing`, {
    method: 'PUT',
    body: input,
  })
}

export function uploadBriefingFile(challengeId: string, file: File): Promise<AdminBriefing> {
  const form = new FormData()
  form.append('file', file)
  return apiPostForm<AdminBriefing>(`/admin/challenges/${challengeId}/briefing/files`, form)
}

export function deleteBriefingFile(challengeId: string, fileId: string): Promise<AdminBriefing> {
  return apiRequest<AdminBriefing>(`/admin/challenges/${challengeId}/briefing/files/${fileId}`, {
    method: 'DELETE',
  })
}

export function saveBriefingOverride(
  challengeId: string,
  userId: string,
  input: OverrideInput,
): Promise<AdminBriefing> {
  return apiRequest<AdminBriefing>(`/admin/challenges/${challengeId}/briefing/contestants/${userId}`, {
    method: 'PUT',
    body: input,
  })
}

export function clearBriefingOverride(challengeId: string, userId: string): Promise<AdminBriefing> {
  return apiRequest<AdminBriefing>(`/admin/challenges/${challengeId}/briefing/contestants/${userId}`, {
    method: 'DELETE',
  })
}

export function uploadOverrideFile(challengeId: string, userId: string, file: File): Promise<AdminBriefing> {
  const form = new FormData()
  form.append('file', file)
  return apiPostForm<AdminBriefing>(
    `/admin/challenges/${challengeId}/briefing/contestants/${userId}/files`,
    form,
  )
}
