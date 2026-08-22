import type { CurrentUser, RoleCode } from './types'

/** Стартовый маршрут после логина по набору ролей. */
export function landingPath(roles: RoleCode[]): string {
  if (
    roles.includes('MEGA_ADMIN') ||
    roles.includes('ADMIN') ||
    roles.includes('SUPER_ADMIN') ||
    roles.includes('STAFF')
  )
    return '/admin'
  if (roles.includes('JURY') || roles.includes('REMOTE_JURY')) return '/jury'
  if (roles.includes('CONTESTANT')) return '/contestant'
  return '/contestant'
}

export const isAdmin = (u: CurrentUser | null) =>
  !!u &&
  (u.roles.includes('MEGA_ADMIN') ||
    u.roles.includes('ADMIN') ||
    u.roles.includes('SUPER_ADMIN') ||
    u.roles.includes('STAFF'))

/** MEGA_ADMIN — платформенный админ (кросс-арендный доступ). */
export const isMega = (u: CurrentUser | null) => !!u && u.roles.includes('MEGA_ADMIN')

/** SUPER_ADMIN — организатор (владелец своих данных). */
export const isSuper = (u: CurrentUser | null) => !!u && u.roles.includes('SUPER_ADMIN')

/** STAFF — сотрудник мероприятия без прав ADMIN. */
export const isStaff = (u: CurrentUser | null) => !!u && u.roles.includes('STAFF')

export function staffPermissionsForContest(
  user: CurrentUser | null,
  contestId: string,
): string[] {
  return user?.staff_permissions?.find((g) => g.contest_id === contestId)?.permissions ?? []
}

export function hasStaffPermission(
  user: CurrentUser | null,
  contestId: string,
  ...needed: string[]
): boolean {
  const granted = staffPermissionsForContest(user, contestId)
  return needed.some((p) => granted.includes(p))
}
