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

export interface LoginOptions {
  telegram: { enabled: boolean; bot_username?: string; mini_app_url?: string }
  vk: { enabled: boolean; app_id?: string; redirect_url?: string }
  events: Array<{ slug: string; name: string }>
}

export function fetchLoginOptions(): Promise<LoginOptions> {
  return participantApiRequest('/auth/login-options')
}

export function socialAuthStartURL(provider: 'vk', eventSlug?: string): string {
  const base = API_BASE_URL.replace(/\/$/, '')
  const slug = eventSlug?.trim()
  if (slug) {
    return `${base}/events/${encodeURIComponent(slug)}/participant-auth/${provider}/start`
  }
  return `${base}/participant-auth/${provider}/start`
}

function socialAuthPath(method: string): string {
  return `/participant-auth/${method}`
}

export interface PublicEvent {
  slug: string
  name: string
}

export type SocialLoginResult =
  | (ParticipantSession & { status?: 'authenticated' })
  | { status: 'choose_event'; events: PublicEvent[]; continue_token: string }

export function isAuthenticatedSession(
  result: SocialLoginResult,
): result is ParticipantSession & { status?: 'authenticated' } {
  return 'participant' in result && 'event' in result && Boolean(result.event?.slug)
}

export function loginParticipantByTelegramWebApp(
  initData: string,
  eventSlug?: string,
): Promise<SocialLoginResult> {
  return participantApiRequest(socialAuthPath('telegram/webapp'), {
    method: 'POST',
    body: { init_data: initData, event_slug: eventSlug || undefined },
  })
}

export function loginParticipantByVKToken(
  accessToken: string,
  eventSlug?: string,
): Promise<SocialLoginResult> {
  return participantApiRequest(socialAuthPath('vk'), {
    method: 'POST',
    body: { access_token: accessToken, event_slug: eventSlug || undefined },
  })
}

export function continueSocialLogin(
  continueToken: string,
  eventSlug?: string,
): Promise<SocialLoginResult> {
  return participantApiRequest(socialAuthPath('continue'), {
    method: 'POST',
    body: { continue_token: continueToken, event_slug: eventSlug || undefined },
  })
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
