import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '@/entities/auth/auth-context'
import { landingPath } from '@/entities/auth/roles'
import type { RoleCode } from '@/entities/auth/types'
import { Loader2 } from 'lucide-react'

function FullscreenLoader() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-surface-2">
      <Loader2 className="h-6 w-6 animate-spin text-brand" aria-label="Загрузка" />
    </div>
  )
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
