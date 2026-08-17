import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react'
import { Outlet } from 'react-router-dom'
import { fetchParticipantMe, logoutParticipant } from './api'
import type { ParticipantSession } from './types'

type ParticipantAuthStatus = 'loading' | 'authenticated' | 'unauthenticated'

interface ParticipantAuthState {
  status: ParticipantAuthStatus
  session: ParticipantSession | null
  acceptSession: (session: ParticipantSession) => void
  refresh: () => Promise<void>
  logout: () => Promise<void>
}

const ParticipantAuthContext = createContext<ParticipantAuthState | null>(null)

export function ParticipantAuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<ParticipantAuthStatus>('loading')
  const [session, setSession] = useState<ParticipantSession | null>(null)

  const refresh = useCallback(async () => {
    try {
      const current = await fetchParticipantMe()
      setSession(current)
      setStatus('authenticated')
    } catch {
      setSession(null)
      setStatus('unauthenticated')
    }
  }, [])

  const acceptSession = useCallback((current: ParticipantSession) => {
    setSession(current)
    setStatus('authenticated')
  }, [])

  const logout = useCallback(async () => {
    try {
      await logoutParticipant()
    } finally {
      setSession(null)
      setStatus('unauthenticated')
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return (
    <ParticipantAuthContext.Provider value={{ status, session, acceptSession, refresh, logout }}>
      {children}
    </ParticipantAuthContext.Provider>
  )
}

export function ParticipantSessionLayout() {
  return (
    <ParticipantAuthProvider>
      <Outlet />
    </ParticipantAuthProvider>
  )
}

export function useParticipantAuth(): ParticipantAuthState {
  const context = useContext(ParticipantAuthContext)
  if (!context) {
    throw new Error('useParticipantAuth must be used within ParticipantAuthProvider')
  }
  return context
}
