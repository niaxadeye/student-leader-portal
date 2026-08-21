import { useEffect, useRef, useState } from 'react'
import type { TelegramWidgetUser } from '@/entities/event-participant/api'
import { Button } from '@/shared/ui/button'

const WIDGET_SRC = 'https://telegram.org/js/telegram-widget.js?22'
const CALLBACK_NAME = 'onTelegramAuth'
const LOAD_TIMEOUT_MS = 6000

declare global {
  interface Window {
    [CALLBACK_NAME]?: (user: TelegramWidgetUser) => void
  }
}

/**
 * Официальный виджет не грузим сами: telegram.org у части сетей таймаутится,
 * и браузер помечает всю вкладку как «не защищено» (origin Unknown/canceled).
 * Скрипт подключается только по клику.
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
  const [armed, setArmed] = useState(false)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    const node = container.current
    if (!armed || !botUsername || !node) return
    window[CALLBACK_NAME] = (user: TelegramWidgetUser) => onAuthRef.current(user)
    const script = document.createElement('script')
    script.async = true
    script.src = WIDGET_SRC
    script.setAttribute('data-telegram-login', botUsername)
    script.setAttribute('data-size', 'large')
    script.setAttribute('data-userpic', 'false')
    script.setAttribute('data-request-access', 'write')
    script.setAttribute('data-onauth', `${CALLBACK_NAME}(user)`)
    const fail = () => setFailed(true)
    script.onerror = fail
    const timer = window.setTimeout(fail, LOAD_TIMEOUT_MS)
    script.addEventListener('load', () => window.clearTimeout(timer))
    node.appendChild(script)
    return () => {
      window.clearTimeout(timer)
      node.replaceChildren()
      delete window[CALLBACK_NAME]
    }
  }, [armed, botUsername])

  if (!botUsername) return null
  if (failed) {
    return (
      <p className="text-center text-[13px] text-muted">
        Telegram в этой сети недоступен. Войдите через VK или резервный способ.
      </p>
    )
  }
  if (!armed) {
    return (
      <Button
        type="button"
        variant="outline"
        className="w-full border-[#229ED9] text-[#229ED9] hover:bg-[#229ED9]/10"
        onClick={() => setArmed(true)}
      >
        Войти через Telegram
      </Button>
    )
  }
  return <div ref={container} className="flex min-h-[40px] justify-center" />
}
