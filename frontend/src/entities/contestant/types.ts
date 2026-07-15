export type ContestantStatus = 'ACTIVE' | 'BLOCKED'

/** Конкурсант из админ-API (бэкенд: contests.Participant, SITE.md §21.7).
 *  Submission-метрики (формы/испытания) появятся на Этапе 3. */
export interface Contestant {
  id: string
  user_id: string
  type: string
  login: string
  full_name: string
  organization: string | null
  user_status: ContestantStatus
  joined_at: string
}

export interface AddContestantInput {
  login: string
  full_name: string
  organization?: string
}

/** Ответ добавления: временный пароль показать один раз. */
export interface AddContestantResult {
  user_id: string
  login: string
  temp_password: string
}

/** Одна строка результата импорта CSV. */
export interface ImportRow {
  line: number
  login: string
  status: 'created' | 'error'
  temp_password?: string
  error?: string
}

export interface ImportResult {
  created: number
  failed: number
  rows: ImportRow[]
}
