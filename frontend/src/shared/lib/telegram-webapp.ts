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
  ready: () => void
  expand: () => void
}

declare global {
  interface Window {
    Telegram?: { WebApp?: TelegramWebApp }
  }
}

const TELEGRAM_WEBAPP_SRC = 'https://telegram.org/js/telegram-web-app.js'

export function telegramWebApp(): TelegramWebApp | null {
  const app = window.Telegram?.WebApp
  if (!app?.initData) return null
  return app
}

export function maybeTelegramMiniApp(): boolean {
  if (window.Telegram?.WebApp) return true
  return /Telegram/i.test(navigator.userAgent)
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
