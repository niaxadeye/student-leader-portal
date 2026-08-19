export interface TelegramWebAppUser {
  id: number
  first_name?: string
  last_name?: string
  username?: string
}

export interface TelegramWebApp {
  initData: string
  initDataUnsafe?: {
    user?: TelegramWebAppUser
    start_param?: string
  }
  platform?: string
  version?: string
  ready: () => void
  expand: () => void
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp }
    TelegramWebviewProxy?: unknown
  }
}

/** Telegram передаёт параметры запуска во фрагменте, оттуда же их читает SDK. */
export function telegramLaunchParams(): string {
  const hash = window.location.hash.replace(/^#/, '')
  return hash.includes('tgWebApp') ? hash : ''
}

// Снимок на момент загрузки бандла: навигация роутера стирает фрагмент.
const launchParamsAtLoad = telegramLaunchParams()

export function telegramLaunchParamsAtLoad(): string {
  return launchParamsAtLoad
}

export function telegramWebApp(): TelegramWebApp | null {
  const app = window.Telegram?.WebApp
  if (!app?.initData) return null
  return app
}

/** Скрипт SDK подключён на всех страницах, поэтому сам по себе объект
 *  window.Telegram.WebApp признаком запуска не является: без параметров
 *  запуска он поднимается с platform "unknown". */
export function maybeTelegramMiniApp(): boolean {
  const app = window.Telegram?.WebApp
  if (app?.initData) return true
  if (app?.platform && app.platform !== 'unknown') return true
  if (window.TelegramWebviewProxy) return true
  return Boolean(telegramLaunchParamsAtLoad())
}

/** Ссылку на Mini App собирают организаторы, поэтому префикс необязателен. */
export function miniAppEventSlug(startParam?: string): string {
  const value = startParam?.trim() ?? ''
  return value.startsWith('event_') ? value.slice('event_'.length) : value
}

// Скрипт подключён в index.html, здесь только ждём, пока он отработает.
export function waitForTelegramWebApp(timeoutMs = 2500): Promise<TelegramWebApp | null> {
  const existing = telegramWebApp()
  if (existing) return Promise.resolve(existing)
  if (!maybeTelegramMiniApp()) return Promise.resolve(null)
  const deadline = Date.now() + timeoutMs
  return new Promise((resolve) => {
    const tick = () => {
      const app = telegramWebApp()
      if (app) {
        resolve(app)
        return
      }
      if (Date.now() >= deadline) {
        resolve(null)
        return
      }
      window.setTimeout(tick, 50)
    }
    tick()
  })
}
