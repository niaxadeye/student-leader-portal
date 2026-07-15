import { LayoutDashboard, Trophy, Users, Shield, Building2, ShieldPlus } from 'lucide-react'
import type { RoleCode } from '@/entities/auth/types'

export interface AdminNavItem {
  to: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  end?: boolean
  /** Если задано — пункт виден только при наличии одной из ролей. */
  roles?: RoleCode[]
}

// Пункты меню админки. «Организаторы» — только MEGA_ADMIN; «Пользователи» —
// SUPER_ADMIN и MEGA_ADMIN (docs/RBAC_MULTITENANCY.md §4).
const items: AdminNavItem[] = [
  { to: '/admin', label: 'Обзор', icon: LayoutDashboard, end: true },
  { to: '/admin/contests', label: 'Конкурсы', icon: Trophy },
  { to: '/admin/organizers', label: 'Организаторы', icon: Building2, roles: ['MEGA_ADMIN'] },
  { to: '/admin/users', label: 'Пользователи', icon: Users, roles: ['SUPER_ADMIN', 'MEGA_ADMIN'] },
]

export function navForRoles(roles: RoleCode[]): AdminNavItem[] {
  return items.filter((i) => !i.roles || i.roles.some((r) => roles.includes(r)))
}

/** Иконка-бейдж роли для хедера. */
export const roleMeta: Record<RoleCode, { label: string; icon: typeof Shield }> = {
  MEGA_ADMIN: { label: 'Мегаадмин', icon: ShieldPlus },
  SUPER_ADMIN: { label: 'Суперадмин', icon: Shield },
  ADMIN: { label: 'Администратор', icon: Shield },
  CONTESTANT: { label: 'Конкурсант', icon: Users },
}
