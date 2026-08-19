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

export function telegramWebApp(): TelegramWebApp | null {
  const app = window.Telegram?.WebApp
  if (!app?.initData) return null
  return app
}
