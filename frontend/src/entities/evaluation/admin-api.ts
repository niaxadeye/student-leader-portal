import { apiRequest } from '@/shared/api/client'
import type { AdminScoreboard, CriterionInput, EvaluationCriterion, EvaluationPayload, LiveSnapshot, SchemeInput, ScoreOverrideInput } from './types'

export function getEvaluation(challengeId: string): Promise<EvaluationPayload> {
  return apiRequest<EvaluationPayload>(`/admin/challenges/${challengeId}/evaluation`)
}

export function getAdminScoreboard(challengeId: string): Promise<AdminScoreboard> {
  return apiRequest<AdminScoreboard>(`/admin/challenges/${challengeId}/evaluation/scores`)
}

export function putEvaluation(challengeId: string, input: SchemeInput): Promise<EvaluationPayload> {
  return apiRequest<EvaluationPayload>(`/admin/challenges/${challengeId}/evaluation`, {
    method: 'PUT',
    body: input,
  })
}

export function addCriterion(challengeId: string, input: CriterionInput): Promise<EvaluationCriterion> {
  return apiRequest<EvaluationCriterion>(`/admin/challenges/${challengeId}/evaluation/criteria`, {
    method: 'POST',
    body: input,
  })
}

export function updateCriterion(
  challengeId: string,
  criterionId: string,
  input: CriterionInput,
): Promise<EvaluationCriterion> {
  return apiRequest<EvaluationCriterion>(
    `/admin/challenges/${challengeId}/evaluation/criteria/${criterionId}`,
    { method: 'PATCH', body: input },
  )
}

export function deleteCriterion(challengeId: string, criterionId: string): Promise<void> {
  return apiRequest(`/admin/challenges/${challengeId}/evaluation/criteria/${criterionId}`, {
    method: 'DELETE',
  })
}

export function getAdminLive(challengeId: string): Promise<LiveSnapshot> {
  return apiRequest<LiveSnapshot>(`/admin/challenges/${challengeId}/live`)
}

function liveCmd(challengeId: string, action: string, body: Record<string, unknown>): Promise<LiveSnapshot> {
  return apiRequest<LiveSnapshot>(`/admin/challenges/${challengeId}/live/${action}`, {
    method: 'POST',
    body,
  })
}

export function startLive(challengeId: string, baseRevision: number): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'start', { base_revision: baseRevision })
}

export function pauseLive(challengeId: string, baseRevision: number): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'pause', { base_revision: baseRevision })
}

export function finishLive(challengeId: string, baseRevision: number): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'finish', { base_revision: baseRevision })
}

export function restartLive(challengeId: string, baseRevision: number): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'restart', { base_revision: baseRevision })
}

export function restartLiveTimer(challengeId: string, baseRevision: number): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'restart-timer', { base_revision: baseRevision })
}

export function completeLiveContestant(challengeId: string, baseRevision: number): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'complete-contestant', { base_revision: baseRevision })
}

export function endSpeechLive(challengeId: string, baseRevision: number): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'end-speech', { base_revision: baseRevision })
}

export function setLiveContestant(
  challengeId: string,
  baseRevision: number,
  contestantUserId: string,
): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'current-contestant', {
    base_revision: baseRevision,
    contestant_user_id: contestantUserId,
  })
}

export function setLivePhase(challengeId: string, baseRevision: number, phaseId: string): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'phase', { base_revision: baseRevision, phase_id: phaseId })
}

export function stepLiveQuestion(challengeId: string, baseRevision: number, delta: number): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'question', { base_revision: baseRevision, delta })
}

export function setLiveCorrectAnswer(challengeId: string, answer: 'YES' | 'NO'): Promise<LiveSnapshot> {
  return apiRequest<LiveSnapshot>(`/admin/challenges/${challengeId}/live/correct-answer`, {
    method: 'POST',
    body: { correct_answer: answer },
  })
}

export function setLiveQuestionPlan(
  challengeId: string,
  questionCount: number,
  answers: { question_number: number; correct_answer: 'YES' | 'NO' }[],
): Promise<LiveSnapshot> {
  return apiRequest<LiveSnapshot>(`/admin/challenges/${challengeId}/live/question-plan`, {
    method: 'PUT',
    body: { question_count: questionCount, answers },
  })
}

export function setLiveDurations(
  challengeId: string,
  baseRevision: number,
  speechSeconds: number,
  questionsSeconds: number,
): Promise<LiveSnapshot> {
  return liveCmd(challengeId, 'durations', {
    base_revision: baseRevision,
    speech_seconds: speechSeconds,
    questions_seconds: questionsSeconds,
  })
}

export function shuffleLiveDraw(challengeId: string): Promise<LiveSnapshot> {
  return apiRequest<LiveSnapshot>(`/admin/challenges/${challengeId}/live/draw`, { method: 'POST', body: {} })
}

export function reorderLiveDraw(challengeId: string, contestantUserIds: string[]): Promise<LiveSnapshot> {
  return apiRequest<LiveSnapshot>(`/admin/challenges/${challengeId}/live/draw`, {
    method: 'PUT',
    body: { contestant_user_ids: contestantUserIds },
  })
}

export function setNumericResult(
  challengeId: string,
  contestantUserId: string,
  score: number | null,
): Promise<AdminScoreboard> {
  return apiRequest<AdminScoreboard>(`/admin/challenges/${challengeId}/evaluation/numeric-results`, {
    method: 'PUT',
    body: { contestant_user_id: contestantUserId, score },
  })
}

export function resetEvaluationResults(challengeId: string, password: string): Promise<{ ok: boolean }> {
  return apiRequest(`/admin/challenges/${challengeId}/evaluation/reset-results`, {
    method: 'POST',
    body: { password },
  })
}

export function replaceEvaluationJury(
  challengeId: string,
  password: string,
  juryUserIds: string[],
): Promise<{ ok: boolean }> {
  return apiRequest(`/admin/challenges/${challengeId}/evaluation/replace-jury`, {
    method: 'POST',
    body: { password, jury_user_ids: juryUserIds },
  })
}

export function putRemoteJury(challengeId: string, juryUserIds: string[]): Promise<EvaluationPayload> {
  return apiRequest<EvaluationPayload>(`/admin/challenges/${challengeId}/evaluation/remote-jury`, {
    method: 'PUT',
    body: { jury_user_ids: juryUserIds },
  })
}

export function putStageLink(
  challengeId: string,
  input: {
    remote_challenge_id: string | null
    main_weight: number
    remote_weight: number
    combine_mode: 'RANK_SUM' | 'SCORE_SUM'
  },
): Promise<EvaluationPayload> {
  return apiRequest<EvaluationPayload>(`/admin/challenges/${challengeId}/evaluation/stage-link`, {
    method: 'PUT',
    body: input,
  })
}

export function overrideScore(challengeId: string, input: ScoreOverrideInput): Promise<AdminScoreboard> {
  return apiRequest<AdminScoreboard>(`/admin/challenges/${challengeId}/evaluation/score-override`, {
    method: 'PUT',
    body: input,
  })
}
