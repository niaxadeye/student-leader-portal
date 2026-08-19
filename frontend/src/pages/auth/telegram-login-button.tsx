import { useEffect, useRef } from 'react'
import type { TelegramWidgetUser } from '@/entities/event-participant/api'

const WIDGET_SRC = 'https://telegram.org/js/telegram-widget.js?22'
const CALLBACK_NAME = 'onTelegramAuth'

declare global {
  interface Window {
    [CALLBACK_NAME]?: (user: TelegramWidgetUser) => void
  }
}

/**
 * Официальный Telegram Login Widget. Работает в режиме колбэка: данные
 * приходят в JS, а не редиректом, поэтому фрагмент URL не задействован.
 */
export function TelegramLoginButton({
  botUsername,
  onAuth,
}: {
  botUsername?: string
  onAuth: (user: TelegramWidgetUser) => void
}) {
  const container = useRef<HTMLDivElement>(null)
  const onAuthRef = useRef(onAuth)
  onAuthRef.current = onAuth

  useEffect(() => {
    const node = container.current
    if (!botUsername || !node) return
    window[CALLBACK_NAME] = (user: TelegramWidgetUser) => onAuthRef.current(user)
    const script = document.createElement('script')
    script.async = true
    script.src = WIDGET_SRC
    script.setAttribute('data-telegram-login', botUsername)
    script.setAttribute('data-size', 'large')
    script.setAttribute('data-userpic', 'false')
    script.setAttribute('data-request-access', 'write')
    script.setAttribute('data-onauth', `${CALLBACK_NAME}(user)`)
    node.appendChild(script)
    return () => {
      node.replaceChildren()
      delete window[CALLBACK_NAME]
    }
  }, [botUsername])

  if (!botUsername) return null
  return <div ref={container} className="flex justify-center [&_iframe]:!w-full" />
}
