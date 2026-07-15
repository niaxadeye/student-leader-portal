import { apiRequest, setAccessToken } from '@/shared/api/client'
import type {
  ChangePasswordInput,
  CurrentUser,
  LoginInput,
  LoginResult,
} from './types'

/** Логин. Auth-запросы не проходят через refresh-интерцептор (skipAuthRefresh). */
export async function login(input: LoginInput): Promise<LoginResult> {
  const res = await apiRequest<LoginResult>('/auth/login', {
    method: 'POST',
    body: input,
    skipAuthRefresh: true,
  })
  setAccessToken(res.access_token)
  return res
}

/** Текущий пользователь. Требует access-токен (refresh-интерцептор подхватит 401). */
export function fetchMe(): Promise<CurrentUser> {
  return apiRequest<CurrentUser>('/auth/me')
}

export async function logout(): Promise<void> {
  await apiRequest('/auth/logout', { method: 'POST' })
  setAccessToken(null)
}

export async function logoutAll(): Promise<void> {
  await apiRequest('/auth/logout-all', { method: 'POST' })
  setAccessToken(null)
}

/** Смена пароля. Бэкенд отзывает сессии → токен более невалиден. */
export async function changePassword(input: ChangePasswordInput): Promise<void> {
  await apiRequest('/auth/change-password', { method: 'POST', body: input })
  setAccessToken(null)
}
