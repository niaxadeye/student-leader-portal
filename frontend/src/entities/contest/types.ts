export type ContestStatus = 'DRAFT' | 'ACTIVE' | 'FINISHED' | 'ARCHIVED'

/** Уровень доступа актора к конкурсу (§4): OWNER — владелец/мега, иначе назначенный ADMIN. */
export type ContestAccessLevel = 'OWNER' | 'EDIT' | 'VIEW'

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
  access_level: ContestAccessLevel | null
  created_at: string
  updated_at: string
  archived_at: string | null
}

/** Право редактировать контент конкурса (испытания/форма/статус): OWNER или EDIT. */
export const canEditContest = (level: ContestAccessLevel | null | undefined) =>
  level === 'OWNER' || level === 'EDIT'

/** Право управлять составом участников — только владелец/мега (§3.6). */
export const canManageParticipants = (level: ContestAccessLevel | null | undefined) =>
  level === 'OWNER'

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
