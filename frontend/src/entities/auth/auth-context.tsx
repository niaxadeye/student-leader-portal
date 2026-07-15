import { createContext, useContext, useEffect, useState, useCallback } from 'react'
import { fetchMe, logout as apiLogout } from './api'
import type { CurrentUser } from './types'

type Status = 'loading' | 'authenticated' | 'unauthenticated'

interface AuthState {
  status: Status
  user: CurrentUser | null
  /** Перечитать /me (после логина/смены пароля). */
  refresh: () => Promise<void>
  /** Локально выставить пользователя без запроса (сразу после login). */
  setUser: (u: CurrentUser | null) => void
  logout: () => Promise<void>
}

const AuthCtx = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<Status>('loading')
  const [user, setUserState] = useState<CurrentUser | null>(null)

  const refresh = useCallback(async () => {
    try {
      const me = await fetchMe()
      setUserState(me)
      setStatus('authenticated')
    } catch {
      setUserState(null)
      setStatus('unauthenticated')
    }
  }, [])

  const setUser = useCallback((u: CurrentUser | null) => {
    setUserState(u)
    setStatus(u ? 'authenticated' : 'unauthenticated')
  }, [])

  const logout = useCallback(async () => {
    try {
      await apiLogout()
    } finally {
      setUserState(null)
      setStatus('unauthenticated')
    }
  }, [])

  // Восстановление сессии на старте: /me → интерцептор при 401 сделает refresh.
  useEffect(() => {
    void refresh()
  }, [refresh])

  return (
    <AuthCtx.Provider value={{ status, user, refresh, setUser, logout }}>
      {children}
    </AuthCtx.Provider>
  )
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthCtx)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
