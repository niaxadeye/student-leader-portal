import { LayoutDashboard, Trophy, Users, Shield } from 'lucide-react'
import type { RoleCode } from '@/entities/auth/types'

export interface AdminNavItem {
  to: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  end?: boolean
  /** Если задано — пункт виден только при наличии одной из ролей. */
  roles?: RoleCode[]
}

// Пункты меню админки. «Пользователи» — только SUPER_ADMIN (SITE.md §5.1).
const items: AdminNavItem[] = [
  { to: '/admin', label: 'Обзор', icon: LayoutDashboard, end: true },
  { to: '/admin/contests', label: 'Конкурсы', icon: Trophy },
  { to: '/admin/users', label: 'Пользователи', icon: Users, roles: ['SUPER_ADMIN'] },
]

export function navForRoles(roles: RoleCode[]): AdminNavItem[] {
  return items.filter((i) => !i.roles || i.roles.some((r) => roles.includes(r)))
}

/** Иконка-бейдж роли для хедера. */
export const roleMeta: Record<RoleCode, { label: string; icon: typeof Shield }> = {
  SUPER_ADMIN: { label: 'Суперадмин', icon: Shield },
  ADMIN: { label: 'Администратор', icon: Shield },
  CONTESTANT: { label: 'Конкурсант', icon: Users },
}
