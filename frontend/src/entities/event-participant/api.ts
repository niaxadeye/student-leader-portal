import { API_BASE_URL, ApiRequestError, type ApiError } from '@/shared/api/client'
import type { IdentifierLoginInput, NameLoginInput, ParticipantSession } from './types'

interface ParticipantRequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown
}

// Participant API намеренно не использует staff access-token и staff refresh flow.
// Единственный credential здесь — отдельная HttpOnly participant cookie.
export async function participantApiRequest<T>(
  path: string,
  options: ParticipantRequestOptions = {},
): Promise<T> {
  const { body, headers, ...rest } = options
  let response: Response
  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      ...rest,
      credentials: 'include',
      headers: {
        ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
        ...headers,
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiRequestError(0, {
      code: 'NETWORK_ERROR',
      message: 'Не удалось связаться с сервером',
    })
  }

  const json = await response.json().catch(() => null)
  if (!response.ok) {
    const error: ApiError = json?.error ?? {
      code: 'INTERNAL_ERROR',
      message: 'Не удалось выполнить запрос',
    }
    throw new ApiRequestError(response.status, error)
  }
  return json.data as T
}

function authPath(eventSlug: string, method: string): string {
  return `/events/${encodeURIComponent(eventSlug)}/participant-auth/${method}`
}

export function loginParticipantByName(
  eventSlug: string,
  input: NameLoginInput,
): Promise<ParticipantSession> {
  return participantApiRequest(authPath(eventSlug, 'fio'), { method: 'POST', body: input })
}

export function loginParticipantByUnionCard(
  eventSlug: string,
  input: IdentifierLoginInput,
): Promise<ParticipantSession> {
  return participantApiRequest(authPath(eventSlug, 'union-card'), { method: 'POST', body: input })
}

export function loginParticipantBySKS(
  eventSlug: string,
  input: IdentifierLoginInput,
): Promise<ParticipantSession> {
  return participantApiRequest(authPath(eventSlug, 'sks'), { method: 'POST', body: input })
}

export function fetchParticipantMe(): Promise<ParticipantSession> {
  return participantApiRequest('/participant/me')
}

export function logoutParticipant(): Promise<void> {
  return participantApiRequest('/participant/logout', { method: 'POST' })
}
