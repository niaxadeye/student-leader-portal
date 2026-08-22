export type EvaluationType =
  | 'CRITERIA_SCORING'
  | 'REMOTE_CRITERIA'
  | 'NUMERIC_RESULT'
  | 'QUESTION_SCORING'
  | 'ELIMINATION_LIVES'
  | 'HEAD_TO_HEAD_VOTING'
  | 'COMPOSITE_SCORING'
  | 'FOCUS_GROUP_SCORING'

export type ScoringUnit = 'POINTS' | 'LIVES' | 'VOTES' | 'PERCENTAGE' | 'SECONDS'
export type CorridorMode = 'NONE' | 'WARN' | 'STRICT'
export type ResultVisibility = 'ADMIN_ONLY' | 'JURY' | 'PUBLIC'
export type EditPolicy = 'WHILE_TRIAL_ACTIVE' | 'UNTIL_LOCK' | 'ALWAYS'

export interface ScaleBand {
  id?: string
  min_score: number
  max_score: number
  description: string
  sort_order?: number
}

export interface EvaluationCriterion {
  id: string
  title: string
  description: string | null
  min_score: number
  max_score: number
  weight: number
  is_required: boolean
  sort_order: number
  bands: ScaleBand[]
}

export interface EvaluationScheme {
  id: string
  challenge_id: string
  contest_id: string
  name: string
  type: EvaluationType
  scoring_unit: ScoringUnit
  min_score: number | null
  max_score: number | null
  corridor_mode: CorridorMode
  result_visibility: ResultVisibility
  edit_policy: EditPolicy
  starting_lives: number | null
  operator_user_id: string | null
  settings?: { starting_lives?: number; operator_user_id?: string }
  active: boolean
  created_at: string
  updated_at: string
  criteria: EvaluationCriterion[]
}

export interface EvaluationPayload {
  configured: boolean
  scheme: EvaluationScheme | null
  jury: AdminScoreboardJury[]
  contest_jury?: AdminScoreboardJury[]
  remote_jury?: AdminScoreboardJury[]
  jury_scope?: 'CONTEST' | 'CHALLENGE'
  stage_link?: StageLink | null
  linked_from?: { id: string; title: string } | null
  remote_stage_options?: Array<{ id: string; title: string }>
}

export type CombineMode = 'RANK_SUM' | 'SCORE_SUM'

export interface StageLink {
  remote_challenge_id: string
  remote_challenge_title: string
  main_weight: number
  remote_weight: number
  combine_mode: CombineMode
}

export interface CombinedRankingRow {
  user_id: string
  full_name: string
  main_score: number | null
  main_rank: number | null
  remote_score: number | null
  remote_rank: number | null
  combined: number | null
  rank: number | null
}

export interface CombinedRanking {
  remote_challenge_id: string
  remote_challenge_title: string
  main_weight: number
  remote_weight: number
  combine_mode: CombineMode
  rows: CombinedRankingRow[]
}

export interface SchemeInput {
  name?: string
  type: EvaluationType
  scoring_unit?: ScoringUnit
  min_score?: number | null
  max_score?: number | null
  corridor_mode?: CorridorMode
  result_visibility?: ResultVisibility
  edit_policy?: EditPolicy
  starting_lives?: number | null
  operator_user_id?: string | null
}

export interface CriterionInput {
  title: string
  description?: string | null
  min_score?: number
  max_score?: number
  weight?: number
  is_required?: boolean
  bands?: Array<{ min_score: number; max_score: number; description: string }>
}

export interface JuryChallenge {
  id: string
  title: string
  slug: string
  status: string
  has_scheme: boolean
  scheme_type?: EvaluationType | ''
}

export interface JuryContest {
  id: string
  name: string
  slug: string
  challenges: JuryChallenge[]
}

export type LiveState =
  | 'NOT_STARTED'
  | 'PREPARING'
  | 'LIVE'
  | 'QUESTIONS'
  | 'DISCUSSION'
  | 'SCORING'
  | 'POST_SCORING'
  | 'PAUSED'
  | 'APPLAUSE'
  | 'FINISHED'

export interface LiveContestant {
  user_id: string
  login: string
  full_name: string
  organization: string | null
  performance_id: string | null
  performance_status: string | null
  draw_number: number | null
  speech_duration_seconds: number | null
  avatar_url: string | null
  lives?: number | null
  eliminated?: boolean
  eliminated_question?: number | null
  rank?: number | null
  restore_questions?: number[]
  answer?: 'YES' | 'NO' | null
  mismatch?: boolean
  can_reveal?: boolean
}

export interface LivePhase {
  id: string
  title: string
  duration_seconds: number | null
  scoring_allowed: boolean
  maps_to_state: LiveState
  sort_order: number
}

export interface LiveSnapshot {
  challenge_id: string
  contest_id: string
  challenge_title: string
  session_revision: number
  state: LiveState
  current_contestant_user_id: string | null
  current_performance_id: string | null
  current_phase_id: string | null
  started_at: string | null
  finished_at: string | null
  phase_started_at: string | null
  phase_duration_seconds: number | null
  paused_at: string | null
  accumulated_pause_seconds: number
  timer_remaining_seconds: number | null
  jury_online: number
  server_time: string
  current: LiveContestant | null
  performance: {
    id: string
    contestant_user_id: string
    status: string
    sequence_number: number | null
    started_at: string | null
    finished_at: string | null
  } | null
  phases: LivePhase[]
  contestants: LiveContestant[]
  drawn: boolean
  draw_locked: boolean
  scheme_type?: EvaluationType | ''
  starting_lives?: number | null
  current_question_number?: number
  operator_user_id?: string | null
  lives?: LivesBoard | null
  correct_answer?: 'YES' | 'NO' | null
  question_count?: number
  question_keys?: LiveQuestionKey[]
}

export interface LiveQuestionKey {
  question_number: number
  correct_answer: 'YES' | 'NO'
}

export interface LivesQuestionLoss {
  contestant_user_id: string
  full_name: string
  organization: string | null
  avatar_url: string | null
}

export interface LivesQuestionAnswer {
  contestant_user_id: string
  full_name: string
  answer: 'YES' | 'NO'
  mismatch: boolean
}

export interface LivesQuestionLog {
  question_number: number
  current: boolean
  correct_answer?: 'YES' | 'NO' | null
  losses: LivesQuestionLoss[]
  answers?: LivesQuestionAnswer[]
}

export interface LivesRow {
  user_id: string
  lives: number
  eliminated: boolean
  eliminated_question: number | null
  rank: number | null
  restore_questions: number[]
  answer?: 'YES' | 'NO' | null
  mismatch?: boolean
  can_reveal?: boolean
}

export interface LivesBoard {
  starting_lives: number
  current_question: number
  operator_user_id: string | null
  viewer_user_id: string | null
  official: boolean
  correct_answer?: 'YES' | 'NO' | null
  questions: LivesQuestionLog[]
  rows: LivesRow[]
}

export interface JuryLifeLog {
  jury_user_id: string
  official: boolean
  questions: LivesQuestionLog[]
  rows: LivesRow[]
}

export interface ChallengeDrawEntry {
  draw_number: number
  full_name: string
  organization: string | null
  is_me: boolean
}

export interface ChallengeDraw {
  drawn: boolean
  my_draw_number: number | null
  total: number
  order: ChallengeDrawEntry[]
}

export interface JuryScorecardCriterion extends EvaluationCriterion {
  score: number | null
  revision: number
}

export interface JuryScorecard {
  configured: boolean
  scheme_type: EvaluationType | ''
  scoring_ui: 'CRITERIA' | 'LIVES' | 'NONE' | ''
  editable: boolean
  performance_id: string | null
  contestant: LiveContestant | null
  criteria: JuryScorecardCriterion[]
  filled: number
  total: number | null
}

export interface JuryScoreWrite {
  criterion_id: string
  score: number
  revision: number
  total: number
}

export interface AdminScoreboardCriterion {
  id: string
  title: string
  min_score: number
  max_score: number
  weight: number
}

export interface AdminScoreboardJury {
  user_id: string
  login: string
  full_name: string
}

export interface AdminScoreboardValue {
  criterion_id: string
  score: number | null
}

export interface AdminScoreboardSheet {
  jury_user_id: string
  filled: number
  total: number | null
  values: AdminScoreboardValue[]
}

export interface AdminScoreboardContestant extends LiveContestant {
  sheets: AdminScoreboardSheet[]
  average: number | null
  sum: number | null
  rank: number | null
  lives?: number | null
  eliminated?: boolean
  eliminated_question?: number | null
  numeric_score?: number | null
}

export interface AdminScoreboard {
  configured: boolean
  scheme_type: EvaluationType | ''
  scoring_ui: 'CRITERIA' | 'LIVES' | 'NUMERIC' | 'NONE' | ''
  current_contestant_user_id: string | null
  criteria: AdminScoreboardCriterion[]
  jury: AdminScoreboardJury[]
  contestants: AdminScoreboardContestant[]
  starting_lives?: number | null
  operator_user_id?: string | null
  current_question_number?: number
  question_count?: number
  life_logs?: JuryLifeLog[]
  min_score?: number | null
  max_score?: number | null
  combined?: CombinedRanking | null
  can_override?: boolean
  corrections?: ScoreCorrection[]
}

export interface ScoreCorrection {
  id: string
  kind: 'CRITERION' | 'NUMERIC' | string
  actor_user_id: string
  actor_name: string
  contestant_user_id: string
  contestant_name: string
  jury_user_id: string | null
  jury_name: string | null
  criterion_id: string | null
  criterion_title: string
  old_score: number | null
  new_score: number | null
  reason: string
  created_at: string
}

export interface ScoreOverrideInput {
  kind: 'CRITERION' | 'NUMERIC'
  contestant_user_id: string
  jury_user_id?: string
  criterion_id?: string
  score: number | null
  reason: string
}

export interface ContestDrawSummary {
  challenge_id: string
  my_draw_number: number | null
  total: number
}

export const liveStateLabels: Record<LiveState, string> = {
  NOT_STARTED: 'Не начато',
  PREPARING: 'Подготовка',
  LIVE: 'Выступление',
  QUESTIONS: 'Вопросы',
  DISCUSSION: 'Обсуждение',
  SCORING: 'Оценивание',
  POST_SCORING: 'После оценок',
  PAUSED: 'Пауза',
  APPLAUSE: 'Аплодисменты',
  FINISHED: 'Завершено',
}

export const evaluationTypeLabels: Record<EvaluationType, string> = {
  CRITERIA_SCORING: 'Критерии',
  REMOTE_CRITERIA: 'Заочное оценивание',
  ELIMINATION_LIVES: '2 к 1',
  HEAD_TO_HEAD_VOTING: 'Прямое голосование',
  NUMERIC_RESULT: 'Числовой результат',
  QUESTION_SCORING: 'Вопросы',
  COMPOSITE_SCORING: 'Составной результат',
  FOCUS_GROUP_SCORING: 'Фокус-группа',
}

export const evaluationTypeDescriptions: Record<EvaluationType, string> = {
  CRITERIA_SCORING: 'Жюри ставит баллы по критериям во время выступления.',
  REMOTE_CRITERIA:
    'Заочное жюри оценивает сданные файлы и материалы по критериям. Основные жюри конкурса в этом испытании не участвуют.',
  ELIMINATION_LIVES: 'У каждого конкурсанта есть жизни. Неверный ответ снимает жизнь, при нуле — выбывание.',
  HEAD_TO_HEAD_VOTING: 'Прямое сравнение и голосование жюри. Настройки появятся позже.',
  NUMERIC_RESULT: 'Администратор задаёт максимальный балл и выставляет результат каждому конкурсанту. Live и жеребьёвки нет.',
  QUESTION_SCORING: 'Оценивание ответов на вопросы. Настройки появятся позже.',
  COMPOSITE_SCORING: 'Составной результат из нескольких частей. Настройки появятся позже.',
  FOCUS_GROUP_SCORING: 'Оценка фокус-группой. Настройки появятся позже.',
}

export const evaluationTypeOrder: EvaluationType[] = [
  'CRITERIA_SCORING',
  'REMOTE_CRITERIA',
  'ELIMINATION_LIVES',
  'HEAD_TO_HEAD_VOTING',
  'NUMERIC_RESULT',
  'QUESTION_SCORING',
  'COMPOSITE_SCORING',
  'FOCUS_GROUP_SCORING',
]

export function schemeHasLive(type?: string | null): boolean {
  return type !== 'NUMERIC_RESULT' && type !== 'REMOTE_CRITERIA'
}

export function contestantLabel(c: {
  draw_number?: number | null
  full_name: string
  organization?: string | null
}): string {
  const num = c.draw_number != null ? `№${c.draw_number} ` : ''
  const org = c.organization?.trim() ? ` · ${c.organization.trim()}` : ''
  return `${num}${c.full_name}${org}`
}

export function formatLiveTimer(seconds: number | null): string {
  if (seconds == null) return '—'
  const neg = seconds < 0
  const s = Math.floor(Math.abs(seconds))
  const m = Math.floor(s / 60)
  const body = `${String(m).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`
  return neg ? `−${body}` : body
}
