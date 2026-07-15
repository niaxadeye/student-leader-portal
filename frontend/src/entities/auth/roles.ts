import type { CurrentUser, RoleCode } from './types'

/** Стартовый маршрут после логина по набору ролей. */
export function landingPath(roles: RoleCode[]): string {
  if (roles.includes('MEGA_ADMIN') || roles.includes('ADMIN') || roles.includes('SUPER_ADMIN'))
    return '/admin'
  if (roles.includes('CONTESTANT')) return '/contestant'
  return '/contestant'
}

export const isAdmin = (u: CurrentUser | null) =>
  !!u &&
  (u.roles.includes('MEGA_ADMIN') ||
    u.roles.includes('ADMIN') ||
    u.roles.includes('SUPER_ADMIN'))

/** MEGA_ADMIN — платформенный админ (кросс-арендный доступ). */
export const isMega = (u: CurrentUser | null) => !!u && u.roles.includes('MEGA_ADMIN')

/** SUPER_ADMIN — организатор (владелец своих данных). */
export const isSuper = (u: CurrentUser | null) => !!u && u.roles.includes('SUPER_ADMIN')
