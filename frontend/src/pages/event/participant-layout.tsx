import { useState } from 'react'
import { ClipboardList, LayoutDashboard, LogOut, QrCode, ShoppingBag, Target } from 'lucide-react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import { cn } from '@/shared/lib/cn'

function initials(fullName: string): string {
  return fullName
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('')
}

export function ParticipantLayout() {
  const navigate = useNavigate()
  const { session, logout } = useParticipantAuth()
  const [isLoggingOut, setIsLoggingOut] = useState(false)
  const eventSlug = session?.event.slug ?? ''

  async function handleLogout() {
    setIsLoggingOut(true)
    try {
      await logout()
    } catch {
      // Локально завершаем сессию даже при истёкшей cookie или потере сети.
    } finally {
      navigate(`/event/${encodeURIComponent(eventSlug)}/login`, { replace: true })
      setIsLoggingOut(false)
    }
  }

  return (
    <div className="min-h-screen bg-surface-2">
      <header className="sticky top-0 z-30 border-b border-border bg-surface/90 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-4">
          <div className="flex min-w-0 items-center gap-5">
            <div className="flex min-w-0 items-center gap-2.5">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-btn bg-brand text-[13px] font-bold text-white">
                SL
              </div>
              <div className="min-w-0">
                <p className="truncate text-[14px] font-semibold text-ink">
                  {session?.event.name ?? 'Мероприятие'}
                </p>
                <p className="text-[11px] text-muted">Кабинет участника</p>
              </div>
            </div>
            <nav className="hidden items-center gap-1 sm:flex">
              <NavLink
                to={`/event/${encodeURIComponent(eventSlug)}/me`}
                end
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2 rounded-btn px-3 py-2 text-[14px] font-medium transition-colors',
                    isActive ? 'bg-brand-subtle text-brand' : 'text-muted hover:text-ink',
                  )
                }
              >
                <LayoutDashboard className="h-4 w-4" />
                Главная
              </NavLink>
              <NavLink
                to={`/event/${encodeURIComponent(eventSlug)}/tasks`}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2 rounded-btn px-3 py-2 text-[14px] font-medium transition-colors',
                    isActive ? 'bg-brand-subtle text-brand' : 'text-muted hover:text-ink',
                  )
                }
              >
                <Target className="h-4 w-4" />
                Задания
              </NavLink>
              <NavLink
                to={`/event/${encodeURIComponent(eventSlug)}/me/qr`}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2 rounded-btn px-3 py-2 text-[14px] font-medium transition-colors',
                    isActive ? 'bg-brand-subtle text-brand' : 'text-muted hover:text-ink',
                  )
                }
              >
                <QrCode className="h-4 w-4" />
                Мой QR
              </NavLink>
              <NavLink
                to={`/event/${encodeURIComponent(eventSlug)}/shop`}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2 rounded-btn px-3 py-2 text-[14px] font-medium transition-colors',
                    isActive ? 'bg-brand-subtle text-brand' : 'text-muted hover:text-ink',
                  )
                }
              >
                <ShoppingBag className="h-4 w-4" />
                Магазин
              </NavLink>
              <NavLink
                to={`/event/${encodeURIComponent(eventSlug)}/orders`}
                className={({ isActive }) =>
                  cn(
                    'flex items-center gap-2 rounded-btn px-3 py-2 text-[14px] font-medium transition-colors',
                    isActive ? 'bg-brand-subtle text-brand' : 'text-muted hover:text-ink',
                  )
                }
              >
                <ClipboardList className="h-4 w-4" />
                Заказы
              </NavLink>
            </nav>
          </div>

          <div className="flex items-center gap-2 sm:gap-3">
            <div className="hidden text-right md:block">
              <p className="max-w-56 truncate text-[13px] font-medium text-ink">
                {session?.participant.full_name}
              </p>
              <p className="text-[11px] text-muted-2">Участник</p>
            </div>
            <div className="flex h-9 w-9 items-center justify-center rounded-full bg-brand-subtle text-[13px] font-semibold text-brand">
              {session ? initials(session.participant.full_name) : '—'}
            </div>
            <button
              type="button"
              aria-label="Выйти из кабинета участника"
              title="Выйти"
              disabled={isLoggingOut}
              onClick={handleLogout}
              className="rounded-btn p-2 text-muted-2 transition-colors hover:bg-danger/10 hover:text-danger disabled:opacity-50"
            >
              <LogOut className="h-5 w-5" />
            </button>
          </div>
        </div>
      </header>
      <nav className="sticky top-16 z-20 flex gap-1 overflow-x-auto border-b border-border bg-surface px-3 py-2 sm:hidden">
        {[
          { to: 'me', label: 'Главная', Icon: LayoutDashboard },
          { to: 'tasks', label: 'Задания', Icon: Target },
          { to: 'shop', label: 'Магазин', Icon: ShoppingBag },
          { to: 'orders', label: 'Заказы', Icon: ClipboardList },
          { to: 'me/qr', label: 'QR', Icon: QrCode },
        ].map(({ to, label, Icon }) => (
          <NavLink
            key={to}
            to={`/event/${encodeURIComponent(eventSlug)}/${to}`}
            end={to === 'me'}
            className={({ isActive }) =>
              cn(
                'flex shrink-0 items-center gap-1.5 rounded-btn px-2.5 py-1.5 text-[12px] font-medium',
                isActive ? 'bg-brand-subtle text-brand' : 'text-muted',
              )
            }
          >
            <Icon className="h-3.5 w-3.5" /> {label}
          </NavLink>
        ))}
      </nav>
      <main className="mx-auto max-w-5xl px-4 py-7 sm:py-9">
        <Outlet />
      </main>
    </div>
  )
}
