import { Navigate, Outlet, useLocation, useParams } from 'react-router-dom'
import { useAuth } from '@/entities/auth/auth-context'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import { landingPath } from '@/entities/auth/roles'
import type { RoleCode } from '@/entities/auth/types'
import { Loader2 } from 'lucide-react'

export function FullscreenLoader() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-surface-2">
      <Loader2 className="h-6 w-6 animate-spin text-brand" aria-label="Загрузка" />
    </div>
  )
}

function participantPath(eventSlug: string, page: 'login' | 'me'): string {
  return `/event/${encodeURIComponent(eventSlug)}/${page}`
}

/** Защищает кабинет отдельной participant cookie и проверяет scope мероприятия. */
export function RequireParticipantAuth() {
  const { eventSlug = '' } = useParams()
  const { status, session } = useParticipantAuth()

  if (status === 'loading') return <FullscreenLoader />
  if (status !== 'authenticated' || !session || session.event.slug !== eventSlug) {
    return <Navigate to={participantPath(eventSlug, 'login')} replace />
  }
  return <Outlet />
}

/** Уже вошедшего в это мероприятие участника уводит из login в его кабинет. */
export function RequireParticipantGuest() {
  const { eventSlug = '' } = useParams()
  const { status, session } = useParticipantAuth()

  if (status === 'loading') return <FullscreenLoader />
  if (status === 'authenticated' && session?.event.slug === eventSlug) {
    return <Navigate to={participantPath(eventSlug, 'me')} replace />
  }
  return <Outlet />
}

/** Требует авторизации. Форсит смену пароля, если она обязательна. */
export function RequireAuth() {
  const { status, user } = useAuth()
  const location = useLocation()

  if (status === 'loading') return <FullscreenLoader />
  if (status === 'unauthenticated' || !user) return <Navigate to="/login" replace />

  const onChangePwd = location.pathname === '/change-password'
  if (user.must_change_password && !onChangePwd) {
    return <Navigate to="/change-password" replace />
  }
  return <Outlet />
}

/** Требует хотя бы одну из ролей; иначе уводит на landing пользователя. */
export function RequireRole({ roles }: { roles: RoleCode[] }) {
  const { status, user } = useAuth()

  if (status === 'loading') return <FullscreenLoader />
  if (!user) return <Navigate to="/login" replace />
  const allowed = user.roles.some((r) => roles.includes(r))
  if (!allowed) return <Navigate to={landingPath(user.roles)} replace />
  return <Outlet />
}

/** Только для гостей: залогиненного уводит на его landing. */
export function RequireGuest() {
  const { status, user } = useAuth()

  if (status === 'loading') return <FullscreenLoader />
  if (status === 'authenticated' && user) {
    const to = user.must_change_password ? '/change-password' : landingPath(user.roles)
    return <Navigate to={to} replace />
  }
  return <Outlet />
}
