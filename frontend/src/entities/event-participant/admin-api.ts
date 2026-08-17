import { apiGetBlob, apiPostForm, apiRequest, apiRequestFull } from '@/shared/api/client'
import type { EventParticipant } from './types'
import type {
  AdminParticipantFilters,
  AdminParticipantInput,
  AdminParticipantList,
  ParticipantExportFormat,
  ParticipantImportResult,
  ParticipantStatusAction,
} from './admin-types'

function participantsPath(contestId: string): string {
  return `/admin/contests/${encodeURIComponent(contestId)}/participants`
}

export async function listAdminParticipants(
  contestId: string,
  filters: AdminParticipantFilters,
): Promise<AdminParticipantList> {
  const query = new URLSearchParams({
    limit: String(filters.limit),
    offset: String(filters.offset),
  })
  if (filters.search) query.set('search', filters.search)
  if (filters.status) query.set('status', filters.status)

  const response = await apiRequestFull<EventParticipant[]>(
    `${participantsPath(contestId)}?${query.toString()}`,
  )
  return {
    participants: response.data,
    total: Number(response.meta?.total ?? response.data.length),
    limit: Number(response.meta?.limit ?? filters.limit),
    offset: Number(response.meta?.offset ?? filters.offset),
  }
}

export function createAdminParticipant(
  contestId: string,
  input: AdminParticipantInput,
): Promise<EventParticipant> {
  return apiRequest(participantsPath(contestId), { method: 'POST', body: input })
}

export function updateAdminParticipant(
  contestId: string,
  participantId: string,
  input: AdminParticipantInput,
): Promise<EventParticipant> {
  return apiRequest(`${participantsPath(contestId)}/${encodeURIComponent(participantId)}`, {
    method: 'PATCH',
    body: input,
  })
}

export function changeAdminParticipantStatus(
  contestId: string,
  participantId: string,
  action: ParticipantStatusAction,
): Promise<EventParticipant> {
  return apiRequest(
    `${participantsPath(contestId)}/${encodeURIComponent(participantId)}/${action}`,
    { method: 'POST' },
  )
}

export function importAdminParticipants(
  contestId: string,
  file: File,
): Promise<ParticipantImportResult> {
  const form = new FormData()
  form.set('file', file)
  return apiPostForm(`${participantsPath(contestId)}/import`, form)
}

export function exportAdminParticipants(
  contestId: string,
  format: ParticipantExportFormat,
): Promise<Blob> {
  return apiGetBlob(`${participantsPath(contestId)}/export?format=${format}`)
}
