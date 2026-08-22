import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getJuryBriefing,
  getJuryLive,
  getJuryScorecard,
  getJurySubmission,
  juryRestoreLife,
  jurySetAnswer,
  listJuryContests,
  listJurySubmissions,
  putJuryScore,
} from './jury-api'
import type { JuryScorecard, LiveSnapshot } from './types'

export function useJuryContests() {
  return useQuery({
    queryKey: ['jury', 'contests'],
    queryFn: listJuryContests,
  })
}

export function useJuryLive(challengeId: string | undefined) {
  return useQuery({
    queryKey: ['jury', 'live', challengeId ?? ''],
    queryFn: () => getJuryLive(challengeId!),
    enabled: !!challengeId,
    refetchInterval: (q) => (q.state.data?.scheme_type === 'REMOTE_CRITERIA' ? false : 2000),
  })
}

export function useJuryBriefing(challengeId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ['jury', 'briefing', challengeId ?? ''],
    queryFn: () => getJuryBriefing(challengeId!),
    enabled: !!challengeId && enabled,
  })
}

export function useJurySubmissions(challengeId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ['jury', 'submissions', challengeId ?? ''],
    queryFn: () => listJurySubmissions(challengeId!),
    enabled: !!challengeId && enabled,
  })
}

export function useJurySubmission(submissionId: string | undefined) {
  return useQuery({
    queryKey: ['jury', 'submission', submissionId ?? ''],
    queryFn: () => getJurySubmission(submissionId!),
    enabled: !!submissionId,
  })
}

export function useJuryScorecard(
  challengeId: string | undefined,
  contestantUserId: string | null | undefined,
  sessionState: string | undefined,
) {
  return useQuery({
    queryKey: ['jury', 'scorecard', challengeId ?? '', contestantUserId ?? '', sessionState ?? ''],
    queryFn: () => getJuryScorecard(challengeId!, contestantUserId),
    enabled: !!challengeId,
  })
}

export function usePutJuryScore(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: {
      performance_id: string
      criterion_id: string
      score: number
      mutation_id: string
    }) => putJuryScore(challengeId, body),
    onSuccess: (res) => {
      qc.setQueriesData<JuryScorecard>({ queryKey: ['jury', 'scorecard', challengeId] }, (prev) => {
        if (!prev) return prev
        let filled = 0
        const criteria = prev.criteria.map((c) => {
          const next =
            c.id === res.criterion_id ? { ...c, score: res.score, revision: res.revision } : c
          if (next.score != null) filled++
          return next
        })
        return { ...prev, criteria, filled, total: res.total }
      })
    },
  })
}

const juryLiveKey = (challengeId: string) => ['jury', 'live', challengeId]

export function useJurySetAnswer(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contestantUserId, answer }: { contestantUserId: string; answer: 'YES' | 'NO' }) =>
      jurySetAnswer(challengeId, contestantUserId, answer),
    onSuccess: (data: LiveSnapshot) => qc.setQueryData(juryLiveKey(challengeId), data),
  })
}

export function useJuryRestoreLife(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contestantUserId, questionNumber }: { contestantUserId: string; questionNumber: number }) =>
      juryRestoreLife(challengeId, contestantUserId, questionNumber),
    onSuccess: (data: LiveSnapshot) => qc.setQueryData(juryLiveKey(challengeId), data),
  })
}
