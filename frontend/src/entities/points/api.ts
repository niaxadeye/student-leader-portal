import { participantApiRequest } from '@/entities/event-participant/api'
import { apiRequest } from '@/shared/api/client'
import type { AdminAdjustmentInput, AdminAdjustmentResult, PointsOverview } from './types'

export function getParticipantPoints(limit = 10, offset = 0): Promise<PointsOverview> {
  return participantApiRequest(`/participant/points?limit=${limit}&offset=${offset}`)
}

function adminPointsPath(contestId: string, participantId: string): string {
  return `/admin/contests/${encodeURIComponent(contestId)}/participants/${encodeURIComponent(participantId)}/points`
}

export function getAdminParticipantPoints(
  contestId: string,
  participantId: string,
  limit = 20,
): Promise<PointsOverview> {
  return apiRequest(`${adminPointsPath(contestId, participantId)}?limit=${limit}`)
}

export function adjustAdminParticipantPoints(
  contestId: string,
  participantId: string,
  input: AdminAdjustmentInput,
): Promise<AdminAdjustmentResult> {
  return apiRequest(`${adminPointsPath(contestId, participantId)}/adjustments`, {
    method: 'POST',
    headers: { 'Idempotency-Key': input.idempotency_key },
    body: input,
  })
}
