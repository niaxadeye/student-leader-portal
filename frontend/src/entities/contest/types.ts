export type ContestStatus = 'DRAFT' | 'ACTIVE' | 'FINISHED' | 'ARCHIVED'

/** Конкурс из админ-API (бэкенд: modules/contests, SITE.md §9, §21.6). */
export interface AdminContest {
  id: string
  name: string
  slug: string
  description: string | null
  status: ContestStatus
  start_at: string | null
  end_at: string | null
  timezone: string
  participants_count: number
  challenges_count: number
  created_at: string
  updated_at: string
  archived_at: string | null
}

/** Тело создания конкурса (SUPER_ADMIN). */
export interface CreateContestInput {
  name: string
  slug?: string
  description?: string
  start_at?: string | null
  end_at?: string | null
  timezone?: string
}

/** Тело редактирования конкурса. */
export interface UpdateContestInput {
  name: string
  description?: string
  start_at?: string | null
  end_at?: string | null
  timezone?: string
}
