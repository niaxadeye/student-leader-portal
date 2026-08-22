import { apiRequest, ApiRequestError } from '@/shared/api/client'
import type { ChallengeDraw, ContestDrawSummary } from './types'

const emptyDraw: ChallengeDraw = { drawn: false, my_draw_number: null, total: 0, order: [] }

export async function getChallengeDraw(challengeId: string): Promise<ChallengeDraw> {
  try {
    return await apiRequest<ChallengeDraw>(`/challenges/${challengeId}/draw`)
  } catch (err) {
    if (err instanceof ApiRequestError && (err.status === 404 || err.code === 'NOT_FOUND')) {
      return emptyDraw
    }
    throw err
  }
}

export async function listContestDraws(contestId: string): Promise<ContestDrawSummary[]> {
  try {
    return await apiRequest<ContestDrawSummary[]>(`/contests/${contestId}/draws`)
  } catch (err) {
    if (err instanceof ApiRequestError && (err.status === 404 || err.code === 'NOT_FOUND')) {
      return []
    }
    throw err
  }
}
