export type RoleCode = 'SUPER_ADMIN' | 'ADMIN' | 'CONTESTANT'

/** Ответ /api/v1/auth/me (SITE.md §20). */
export interface CurrentUser {
  id: string
  login: string
  full_name: string
  roles: RoleCode[]
  must_change_password: boolean
}

/** Ответ /api/v1/auth/login. */
export interface LoginResult {
  access_token: string
  expires_at: string
  must_change_password: boolean
}

export interface LoginInput {
  login: string
  password: string
}

export interface ChangePasswordInput {
  old_password: string
  new_password: string
}
