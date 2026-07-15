import type { RoleCode } from '@/entities/auth/types'

export type UserStatus = 'ACTIVE' | 'BLOCKED'

/** Назначение роли со scope (бэкенд: useradmin.RoleAssignment). */
export interface RoleAssignment {
  code: RoleCode
  scope_type: 'GLOBAL' | 'CONTEST'
  scope_id: string
}

/** Пользователь реестра SUPER_ADMIN (SITE.md §19). */
export interface AdminUser {
  id: string
  login: string
  full_name: string
  email: string | null
  organization: string | null
  status: UserStatus
  must_change_password: boolean
  last_login_at: string | null
  created_at: string
  roles: RoleAssignment[]
}

/** Страница реестра с метаданными пагинации. */
export interface UsersPage {
  users: AdminUser[]
  total: number
  limit: number
  offset: number
}

/** Фильтры списка (хранятся в URL, SITE.md §43). */
export interface UsersFilter {
  search?: string
  role?: string
  status?: string
  limit?: number
  offset?: number
}

export interface CreateUserInput {
  login: string
  full_name: string
  email?: string
  organization?: string
  role?: RoleCode
  scope_type?: 'GLOBAL' | 'CONTEST'
  scope_id?: string
}

/** Ответ создания: временный пароль показать один раз. */
export interface CreateUserResult {
  user_id: string
  login: string
  temp_password: string
}
