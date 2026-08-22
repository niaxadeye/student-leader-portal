import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  addCriterion,
  completeLiveContestant,
  deleteCriterion,
  endSpeechLive,
  finishLive,
  getAdminLive,
  getAdminScoreboard,
  getEvaluation,
  pauseLive,
  putEvaluation,
  reorderLiveDraw,
  replaceEvaluationJury,
  putRemoteJury,
  putStageLink,
  overrideScore,
  resetEvaluationResults,
  restartLive,
  restartLiveTimer,
  setLiveContestant,
  setLiveDurations,
  setLivePhase,
  setLiveQuestionPlan,
  setNumericResult,
  shuffleLiveDraw,
  startLive,
  stepLiveQuestion,
  updateCriterion,
} from './admin-api'
import type { CriterionInput, LiveSnapshot, SchemeInput, ScoreOverrideInput } from './types'

const evalKey = (challengeId: string) => ['admin', 'evaluation', challengeId]

export function useEvaluation(challengeId: string | undefined) {
  return useQuery({
    queryKey: evalKey(challengeId ?? ''),
    queryFn: () => getEvaluation(challengeId!),
    enabled: !!challengeId,
  })
}

export function useAdminScoreboard(challengeId: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'evaluation', 'scores', challengeId ?? ''],
    queryFn: () => getAdminScoreboard(challengeId!),
    enabled: !!challengeId,
    refetchInterval: 4000,
  })
}

export function useSaveEvaluation(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: SchemeInput) => putEvaluation(challengeId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: evalKey(challengeId) })
      qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId] })
      qc.invalidateQueries({ queryKey: ['admin', 'challenges'] })
      qc.invalidateQueries({ queryKey: ['admin', 'evaluation', 'scores', challengeId] })
    },
  })
}

export function useAddCriterion(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CriterionInput) => addCriterion(challengeId, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: evalKey(challengeId) }),
  })
}

export function useUpdateCriterion(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: CriterionInput }) =>
      updateCriterion(challengeId, id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: evalKey(challengeId) }),
  })
}

export function useDeleteCriterion(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (criterionId: string) => deleteCriterion(challengeId, criterionId),
    onSuccess: () => qc.invalidateQueries({ queryKey: evalKey(challengeId) }),
  })
}

const liveKey = (challengeId: string) => ['admin', 'live', challengeId]

export function useAdminLive(challengeId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: liveKey(challengeId ?? ''),
    queryFn: () => getAdminLive(challengeId!),
    enabled: !!challengeId && enabled,
    refetchInterval: 2000,
  })
}

function useLiveMutation(
  challengeId: string,
  fn: (baseRevision: number, extra?: string) => Promise<LiveSnapshot>,
) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ baseRevision, extra }: { baseRevision: number; extra?: string }) =>
      fn(baseRevision, extra),
    onSuccess: (data) => qc.setQueryData(liveKey(challengeId), data),
  })
}

export function useStartLive(challengeId: string) {
  return useLiveMutation(challengeId, (rev) => startLive(challengeId, rev))
}

export function usePauseLive(challengeId: string) {
  return useLiveMutation(challengeId, (rev) => pauseLive(challengeId, rev))
}

export function useFinishLive(challengeId: string) {
  return useLiveMutation(challengeId, (rev) => finishLive(challengeId, rev))
}

export function useRestartLive(challengeId: string) {
  return useLiveMutation(challengeId, (rev) => restartLive(challengeId, rev))
}

export function useRestartLiveTimer(challengeId: string) {
  return useLiveMutation(challengeId, (rev) => restartLiveTimer(challengeId, rev))
}

export function useCompleteLiveContestant(challengeId: string) {
  return useLiveMutation(challengeId, (rev) => completeLiveContestant(challengeId, rev))
}

export function useEndSpeechLive(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ baseRevision }: { baseRevision: number }) => endSpeechLive(challengeId, baseRevision),
    onSuccess: (data) => {
      qc.setQueryData(liveKey(challengeId), data)
      qc.invalidateQueries({ queryKey: ['admin', 'evaluation', 'scores', challengeId] })
    },
  })
}

export function useSetLiveContestant(challengeId: string) {
  return useLiveMutation(challengeId, (rev, extra) => setLiveContestant(challengeId, rev, extra ?? ''))
}

export function useSetLivePhase(challengeId: string) {
  return useLiveMutation(challengeId, (rev, extra) => setLivePhase(challengeId, rev, extra ?? ''))
}

export function useStepLiveQuestion(challengeId: string) {
  return useLiveMutation(challengeId, (rev, extra) => stepLiveQuestion(challengeId, rev, Number(extra ?? '0')))
}

export function useSetLiveQuestionPlan(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      questionCount,
      answers,
    }: {
      questionCount: number
      answers: { question_number: number; correct_answer: 'YES' | 'NO' }[]
    }) => setLiveQuestionPlan(challengeId, questionCount, answers),
    onSuccess: (data: LiveSnapshot) => {
      qc.setQueryData(liveKey(challengeId), data)
      qc.invalidateQueries({ queryKey: ['admin', 'evaluation', 'scores', challengeId] })
    },
  })
}

export function useSetLiveDurations(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      baseRevision,
      speechSeconds,
      questionsSeconds,
    }: {
      baseRevision: number
      speechSeconds: number
      questionsSeconds: number
    }) => setLiveDurations(challengeId, baseRevision, speechSeconds, questionsSeconds),
    onSuccess: (data: LiveSnapshot) => qc.setQueryData(liveKey(challengeId), data),
  })
}

export function useSetNumericResult(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contestantUserId, score }: { contestantUserId: string; score: number | null }) =>
      setNumericResult(challengeId, contestantUserId, score),
    onSuccess: (data) => {
      qc.setQueryData(['admin', 'evaluation', 'scores', challengeId], data)
      qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId] })
      qc.invalidateQueries({ queryKey: ['admin', 'challenges'] })
    },
  })
}

export function useOverrideScore(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ScoreOverrideInput) => overrideScore(challengeId, input),
    onSuccess: (data) => {
      qc.setQueryData(['admin', 'evaluation', 'scores', challengeId], data)
      qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId] })
      qc.invalidateQueries({ queryKey: ['admin', 'challenges'] })
    },
  })
}

function invalidateTrial(qc: ReturnType<typeof useQueryClient>, challengeId: string) {
  qc.invalidateQueries({ queryKey: evalKey(challengeId) })
  qc.invalidateQueries({ queryKey: ['admin', 'evaluation', 'scores', challengeId] })
  qc.invalidateQueries({ queryKey: liveKey(challengeId) })
  qc.invalidateQueries({ queryKey: ['admin', 'challenge', challengeId] })
  qc.invalidateQueries({ queryKey: ['admin', 'challenges'] })
}

export function useResetEvaluationResults(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (password: string) => resetEvaluationResults(challengeId, password),
    onSuccess: () => invalidateTrial(qc, challengeId),
  })
}

export function useReplaceEvaluationJury(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ password, juryUserIds }: { password: string; juryUserIds: string[] }) =>
      replaceEvaluationJury(challengeId, password, juryUserIds),
    onSuccess: () => invalidateTrial(qc, challengeId),
  })
}

export function usePutRemoteJury(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (juryUserIds: string[]) => putRemoteJury(challengeId, juryUserIds),
    onSuccess: () => invalidateTrial(qc, challengeId),
  })
}

export function usePutStageLink(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: {
      remote_challenge_id: string | null
      main_weight: number
      remote_weight: number
      combine_mode: 'RANK_SUM' | 'SCORE_SUM'
    }) => putStageLink(challengeId, input),
    onSuccess: () => invalidateTrial(qc, challengeId),
  })
}

export function useShuffleLiveDraw(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => shuffleLiveDraw(challengeId),
    onSuccess: (data) => qc.setQueryData(liveKey(challengeId), data),
  })
}

export function useReorderLiveDraw(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (contestantUserIds: string[]) => reorderLiveDraw(challengeId, contestantUserIds),
    onSuccess: (data: LiveSnapshot) => qc.setQueryData(liveKey(challengeId), data),
  })
}
