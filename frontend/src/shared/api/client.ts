// Типизированный HTTP-клиент. Работает с envelope-ответами API (SITE.md §20).
// Access-токен хранится только в памяти; refresh-токен — в HttpOnly-cookie.
// На 401 один раз пытается обновить сессию через /auth/refresh и повторяет запрос.

export interface ApiError {
  code: string
  message: string
  details?: Record<string, unknown>
}

export class ApiRequestError extends Error {
  code: string
  status: number
  details?: Record<string, unknown>
  constructor(status: number, error: ApiError) {
    super(error.message)
    this.name = 'ApiRequestError'
    this.status = status
    this.code = error.code
    this.details = error.details
  }
}

const BASE_URL = import.meta.env.VITE_API_URL ?? '/api/v1'

// ── Хранилище access-токена (in-memory) ─────────────────────────────────────
let accessToken: string | null = null
export const setAccessToken = (t: string | null) => {
  accessToken = t
}
export const getAccessToken = () => accessToken

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown
  /** Не пытаться обновлять сессию на 401 (для самих auth-запросов). */
  skipAuthRefresh?: boolean
}

// Один общий in-flight refresh, чтобы параллельные 401 не плодили запросы.
let refreshInFlight: Promise<boolean> | null = null

async function tryRefresh(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      try {
        const res = await fetch(`${BASE_URL}/auth/refresh`, {
          method: 'POST',
          credentials: 'include',
        })
        if (!res.ok) return false
        const json = await res.json().catch(() => null)
        const token = json?.data?.access_token as string | undefined
        if (!token) return false
        setAccessToken(token)
        return true
      } catch {
        return false
      } finally {
        refreshInFlight = null
      }
    })()
  }
  return refreshInFlight
}

async function rawRequest(path: string, options: RequestOptions): Promise<Response> {
  const { body, headers, skipAuthRefresh: _s, ...rest } = options
  return fetch(`${BASE_URL}${path}`, {
    ...rest,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
}

/** Полный envelope: данные + meta (для серверной пагинации, SITE.md §43). */
export interface ApiEnvelope<T> {
  data: T
  meta?: Record<string, unknown>
}

/** Как apiRequest, но возвращает и data, и meta. */
export async function apiRequestFull<T>(
  path: string,
  options: RequestOptions = {},
): Promise<ApiEnvelope<T>> {
  let res = await rawRequest(path, options)

  // Истёк access-токен → однократный refresh + повтор исходного запроса.
  if (res.status === 401 && !options.skipAuthRefresh) {
    const refreshed = await tryRefresh()
    if (refreshed) res = await rawRequest(path, options)
  }

  const json = await res.json().catch(() => null)

  if (!res.ok) {
    const err: ApiError = json?.error ?? {
      code: 'INTERNAL_ERROR',
      message: 'Не удалось выполнить запрос',
    }
    throw new ApiRequestError(res.status, err)
  }

  return { data: json.data as T, meta: json.meta as Record<string, unknown> | undefined }
}

/** Выполняет запрос и разворачивает envelope. Бросает ApiRequestError на ошибке. */
export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { data } = await apiRequestFull<T>(path, options)
  return data
}

// ── Сырые тела (CSV import/export) ──────────────────────────────────────────

async function rawTextRequest(path: string, method: string, body?: string, contentType?: string) {
  let res = await fetch(`${BASE_URL}${path}`, {
    method,
    credentials: 'include',
    headers: {
      ...(contentType ? { 'Content-Type': contentType } : {}),
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
    },
    body,
  })
  if (res.status === 401) {
    const refreshed = await tryRefresh()
    if (refreshed) {
      res = await fetch(`${BASE_URL}${path}`, {
        method,
        credentials: 'include',
        headers: {
          ...(contentType ? { 'Content-Type': contentType } : {}),
          ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
        },
        body,
      })
    }
  }
  return res
}

/** POST CSV в теле, разворачивает JSON-envelope ответа (сводка импорта). */
export async function apiPostText<T>(path: string, text: string): Promise<T> {
  const res = await rawTextRequest(path, 'POST', text, 'text/csv')
  const json = await res.json().catch(() => null)
  if (!res.ok) {
    throw new ApiRequestError(res.status, json?.error ?? { code: 'INTERNAL_ERROR', message: 'Ошибка импорта' })
  }
  return json.data as T
}

/** GET, возвращает тело как текст (CSV-выгрузка). */
export async function apiGetText(path: string): Promise<string> {
  const res = await rawTextRequest(path, 'GET')
  if (!res.ok) {
    throw new ApiRequestError(res.status, { code: 'INTERNAL_ERROR', message: 'Ошибка экспорта' })
  }
  return res.text()
}

// ── Multipart-загрузка (файлы submission) ────────────────────────────────────

/** POST multipart/form-data. Content-Type ставит браузер (с boundary). Разворачивает envelope. */
export async function apiPostForm<T>(path: string, form: FormData): Promise<T> {
  const send = () =>
    fetch(`${BASE_URL}${path}`, {
      method: 'POST',
      credentials: 'include',
      headers: { ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}) },
      body: form,
    })
  let res = await send()
  if (res.status === 401 && (await tryRefresh())) res = await send()
  const json = await res.json().catch(() => null)
  if (!res.ok) {
    throw new ApiRequestError(
      res.status,
      json?.error ?? { code: 'INTERNAL_ERROR', message: 'Ошибка загрузки файла' },
    )
  }
  return json.data as T
}
