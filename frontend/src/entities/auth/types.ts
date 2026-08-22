export type RoleCode = 'MEGA_ADMIN' | 'SUPER_ADMIN' | 'ADMIN' | 'STAFF' | 'JURY' | 'REMOTE_JURY' | 'CONTESTANT'

export type StaffPermission =
  | 'event.participants.manage'
  | 'event.attendance.scan'
  | 'event.attendance.manage'
  | 'event.tasks.manage'
  | 'event.tasks.moderate'
  | 'event.merch.manage'
  | 'event.merch.orders.manage'
  | 'event.points.manage'

export interface StaffGrant {
  contest_id: string
  permissions: StaffPermission[]
}

/** Ответ /api/v1/auth/me (SITE.md §20). */
export interface CurrentUser {
  id: string
  login: string
  full_name: string
  roles: RoleCode[]
  must_change_password: boolean
  staff_permissions?: StaffGrant[]
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
  audience?: 'admin' | 'contestant' | 'jury'
}

export interface ChangePasswordInput {
  old_password: string
  new_password: string
}
