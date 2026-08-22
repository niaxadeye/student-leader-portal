import { apiRequest, apiRequestFull } from '@/shared/api/client'
import type { ChallengeBriefing } from '@/entities/challenge/types'
import type { AdminSubmissionDetail, AdminSubmissionRow } from '@/entities/submission/admin-api'
import type { JuryContest, JuryScorecard, JuryScoreWrite, LiveSnapshot } from './types'

export function listJuryContests(): Promise<JuryContest[]> {
  return apiRequest<JuryContest[]>('/jury/contests')
}

export function getJuryLive(challengeId: string): Promise<LiveSnapshot> {
  return apiRequest<LiveSnapshot>(`/jury/challenges/${challengeId}/live`)
}

export function getJuryScorecard(challengeId: string, contestantUserId?: string | null): Promise<JuryScorecard> {
  const q = contestantUserId
    ? `?contestant_user_id=${encodeURIComponent(contestantUserId)}`
    : ''
  return apiRequest<JuryScorecard>(`/jury/challenges/${challengeId}/scorecard${q}`)
}

export function putJuryScore(
  challengeId: string,
  body: { performance_id: string; criterion_id: string; score: number; mutation_id: string },
): Promise<JuryScoreWrite> {
  return apiRequest<JuryScoreWrite>(`/jury/challenges/${challengeId}/scorecard`, {
    method: 'PUT',
    body,
  })
}

export function getJuryBriefing(challengeId: string): Promise<ChallengeBriefing> {
  return apiRequest<ChallengeBriefing>(`/jury/challenges/${challengeId}/briefing`)
}

export async function listJurySubmissions(
  challengeId: string,
): Promise<{ rows: AdminSubmissionRow[]; total: number }> {
  const { data, meta } = await apiRequestFull<AdminSubmissionRow[]>(
    `/jury/challenges/${challengeId}/submissions`,
  )
  return { rows: data, total: (meta?.total as number) ?? data.length }
}

export function getJurySubmission(submissionId: string): Promise<AdminSubmissionDetail> {
  return apiRequest<AdminSubmissionDetail>(`/jury/submissions/${submissionId}`)
}

export async function getJuryFileDownloadUrl(submissionId: string, fileId: string): Promise<string> {
  const { download_url } = await apiRequest<{ download_url: string }>(
    `/jury/submissions/${submissionId}/files/${fileId}`,
  )
  return download_url
}

export function jurySetAnswer(
  challengeId: string,
  contestantUserId: string,
  answer: 'YES' | 'NO',
): Promise<LiveSnapshot> {
  return apiRequest<LiveSnapshot>(`/jury/challenges/${challengeId}/lives/answer`, {
    method: 'POST',
    body: { contestant_user_id: contestantUserId, answer },
  })
}

export function juryRestoreLife(
  challengeId: string,
  contestantUserId: string,
  questionNumber: number,
): Promise<LiveSnapshot> {
  return apiRequest<LiveSnapshot>(`/jury/challenges/${challengeId}/lives/restore`, {
    method: 'POST',
    body: { contestant_user_id: contestantUserId, question_number: questionNumber },
  })
}
