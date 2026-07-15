import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { LogOut } from 'lucide-react'
import { useAuth } from '@/entities/auth/auth-context'
import { cn } from '@/shared/lib/cn'
import { navForRoles, roleMeta, type AdminNavItem } from './admin-nav'
import type { RoleCode, CurrentUser } from '@/entities/auth/types'

function initials(fullName: string): string {
  return fullName
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? '')
    .join('')
}

/** Наивысшая роль для бейджа: MEGA_ADMIN > SUPER_ADMIN > ADMIN > CONTESTANT. */
function topRole(roles: RoleCode[]): RoleCode {
  if (roles.includes('MEGA_ADMIN')) return 'MEGA_ADMIN'
  if (roles.includes('SUPER_ADMIN')) return 'SUPER_ADMIN'
  if (roles.includes('ADMIN')) return 'ADMIN'
  return 'CONTESTANT'
}

export function AdminLayout() {
  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const roles = user?.roles ?? []
  const nav = navForRoles(roles)
  const role = roleMeta[topRole(roles)]

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="flex min-h-screen bg-surface-2">
      <AdminSidebar nav={nav} />
      <div className="flex min-w-0 flex-1 flex-col">
        <AdminHeader user={user} role={role} onLogout={handleLogout} initials={initials} />
        <main className="mx-auto w-full max-w-6xl flex-1 px-6 py-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}

function AdminSidebar({ nav }: { nav: AdminNavItem[] }) {
  return (
    <aside className="hidden w-60 shrink-0 flex-col border-r border-border bg-surface md:flex">
      <div className="flex h-16 items-center gap-2 border-b border-border px-5">
        <div className="flex h-8 w-8 items-center justify-center rounded-btn bg-brand text-[14px] font-bold text-white">
          SL
        </div>
        <span className="text-[15px] font-semibold text-ink">Админ-панель</span>
      </div>
      <nav className="flex flex-col gap-1 p-3">
        {nav.map(({ to, label, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-btn px-3 py-2 text-[14px] font-medium transition-colors',
                isActive ? 'bg-brand-subtle text-brand' : 'text-muted hover:bg-muted/10 hover:text-ink',
              )
            }
          >
            <Icon className="h-[18px] w-[18px]" />
            {label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}

function AdminHeader({
  user,
  role,
  onLogout,
  initials,
}: {
  user: CurrentUser | null
  role: { label: string; icon: React.ComponentType<{ className?: string }> }
  onLogout: () => void
  initials: (n: string) => string
}) {
  const RoleIcon = role.icon
  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-end gap-3 border-b border-border bg-surface/80 px-6 backdrop-blur">
      <div className="flex items-center gap-1 rounded-badge bg-brand-subtle px-2 py-1 text-[12px] font-medium text-brand">
        <RoleIcon className="h-3.5 w-3.5" />
        {role.label}
      </div>
      <div className="hidden text-right sm:block">
        <p className="text-[13px] font-medium text-ink">{user?.full_name ?? '—'}</p>
        <p className="text-[12px] text-muted-2">{user?.login}</p>
      </div>
      <div className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-subtle text-[13px] font-semibold text-brand">
        {user ? initials(user.full_name) : '—'}
      </div>
      <button aria-label="Выйти" onClick={onLogout} className="text-muted-2 hover:text-danger">
        <LogOut className="h-5 w-5" />
      </button>
    </header>
  )
}
