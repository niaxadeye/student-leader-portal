import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { LayoutDashboard, BookOpen, LogOut } from 'lucide-react'
import { useAuth } from '@/entities/auth/auth-context'
import { cn } from '@/shared/lib/cn'

const nav = [
  { to: '/contestant', label: 'Кабинет', icon: LayoutDashboard, end: true },
  { to: '/reference', label: 'Справка', icon: BookOpen, end: false },
]

/** Инициалы из ФИО: «Иванова Анна» → «ИА». */
function initials(fullName: string): string {
  return fullName
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? '')
    .join('')
}

export function ContestantLayout() {
  const navigate = useNavigate()
  const { user, logout } = useAuth()

  async function handleLogout() {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen">
      <header className="sticky top-0 z-30 border-b border-border bg-surface/80 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-4">
          <div className="flex items-center gap-6">
            <div className="flex items-center gap-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-btn bg-brand text-[14px] font-bold text-white">
                SL
              </div>
              <span className="hidden text-[15px] font-semibold text-ink sm:block">
                Студенческий лидер
              </span>
            </div>
            <nav className="flex items-center gap-1">
              {nav.map(({ to, label, icon: Icon, end }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={end}
                  className={({ isActive }) =>
                    cn(
                      'flex items-center gap-2 rounded-btn px-3 py-2 text-[14px] font-medium transition-colors',
                      isActive ? 'bg-brand-subtle text-brand' : 'text-muted hover:text-ink',
                    )
                  }
                >
                  <Icon className="h-4 w-4" />
                  <span className="hidden xs:block">{label}</span>
                </NavLink>
              ))}
            </nav>
          </div>
          <div className="flex items-center gap-3">
            <div className="hidden text-right sm:block">
              <p className="text-[13px] font-medium text-ink">{user?.full_name ?? '—'}</p>
              <p className="text-[12px] text-muted-2">{user?.login}</p>
            </div>
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-subtle text-[13px] font-semibold text-brand">
              {user ? initials(user.full_name) : '—'}
            </div>
            <button
              aria-label="Выйти"
              onClick={handleLogout}
              className="text-muted-2 hover:text-danger"
            >
              <LogOut className="h-5 w-5" />
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-8">
        <Outlet />
      </main>
    </div>
  )
}
