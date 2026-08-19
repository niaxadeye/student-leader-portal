import { Navigate, useParams } from 'react-router-dom'

/** Старые ссылки /event/:slug/login ведут на единую страницу входа. */
export function ParticipantLoginPage() {
  const { eventSlug = '' } = useParams()
  const to = eventSlug
    ? `/login?as=participant&event=${encodeURIComponent(eventSlug)}`
    : '/login?as=participant'
  return <Navigate to={to} replace />
}
