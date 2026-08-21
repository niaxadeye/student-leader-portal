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
    __TG_LAUNCH_HASH__?: string
  }
}

const SDK_SRC = 'https://telegram.org/js/telegram-web-app.js?63'

/** Telegram передаёт параметры запуска во фрагменте, оттуда же их читает SDK. */
export function telegramLaunchParams(): string {
  const hash = window.location.hash.replace(/^#/, '')
  return hash.includes('tgWebApp') ? hash : ''
}

function snapshotLaunchHash(): string {
  const injected = window.__TG_LAUNCH_HASH__
  if (typeof injected === 'string' && injected.includes('tgWebApp')) return injected
  return telegramLaunchParams()
}

// Снимок на момент загрузки: навигация роутера стирает фрагмент.
const launchParamsAtLoad = snapshotLaunchHash()

export function telegramLaunchParamsAtLoad(): string {
  return launchParamsAtLoad
}

export function telegramWebApp(): TelegramWebApp | null {
  const app = window.Telegram?.WebApp
  if (!app?.initData) return null
  return app
}

/** Скрипт SDK больше не висит на всех страницах. Mini App определяем по
 *  initData, platform, WebView-прокси или снимку hash при загрузке. */
export function maybeTelegramMiniApp(): boolean {
  const app = window.Telegram?.WebApp
  if (app?.initData) return true
  if (app?.platform && app.platform !== 'unknown') return true
  if (window.TelegramWebviewProxy) return true
  return Boolean(telegramLaunchParamsAtLoad())
}

function restoreLaunchHash(): void {
  if (!launchParamsAtLoad || window.location.hash.includes('tgWebApp')) return
  const url = `${window.location.pathname}${window.location.search}#${launchParamsAtLoad}`
  window.history.replaceState(window.history.state, '', url)
}

function ensureTelegramSdk(): void {
  if (!maybeTelegramMiniApp()) return
  if (document.querySelector('script[data-telegram-web-app]')) return
  restoreLaunchHash()
  const script = document.createElement('script')
  script.src = SDK_SRC
  script.async = true
  script.setAttribute('data-telegram-web-app', '1')
  document.head.appendChild(script)
}

ensureTelegramSdk()

/** Ссылку на Mini App собирают организаторы, поэтому префикс необязателен. */
export function miniAppEventSlug(startParam?: string): string {
  const value = startParam?.trim() ?? ''
  return value.startsWith('event_') ? value.slice('event_'.length) : value
}

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
