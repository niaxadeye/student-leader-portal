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

const TELEGRAM_WEBAPP_SRC = 'https://telegram.org/js/telegram-web-app.js'

export function telegramWebApp(): TelegramWebApp | null {
  const app = window.Telegram?.WebApp
  if (!app?.initData) return null
  return app
}

/** Повод подождать initData. В вебвью Telegram user-agent обычно обычный,
 *  поэтому опираемся на инжектированный мост и параметры запуска. */
export function maybeTelegramMiniApp(): boolean {
  if (window.Telegram?.WebApp) return true
  if (window.TelegramWebviewProxy) return true
  if (telegramLaunchParams()) return true
  return /Telegram/i.test(navigator.userAgent)
}

/** Ссылку на Mini App собирают организаторы, поэтому префикс необязателен. */
export function miniAppEventSlug(startParam?: string): string {
  const value = startParam?.trim() ?? ''
  return value.startsWith('event_') ? value.slice('event_'.length) : value
}

export function ensureTelegramWebAppScript(): void {
  if (document.querySelector('script[data-telegram-web-app]')) return
  const script = document.createElement('script')
  script.src = TELEGRAM_WEBAPP_SRC
  script.async = true
  script.dataset.telegramWebApp = '1'
  document.head.appendChild(script)
}

export function waitForTelegramWebApp(timeoutMs = 2500): Promise<TelegramWebApp | null> {
  const existing = telegramWebApp()
  if (existing) return Promise.resolve(existing)
  if (!maybeTelegramMiniApp()) return Promise.resolve(null)
  ensureTelegramWebAppScript()
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
