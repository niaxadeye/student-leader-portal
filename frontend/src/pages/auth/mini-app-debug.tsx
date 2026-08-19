import {
  maybeTelegramMiniApp,
  telegramLaunchParams,
  telegramLaunchParamsAtLoad,
} from '@/shared/lib/telegram-webapp'

/**
 * Временная панель для разбора запуска Mini App: показывает, что именно
 * пришло от Telegram. Видна только внутри Telegram или по ?debug=1.
 */
export function MiniAppDebug({ probe }: { probe: string }) {
  const forced = new URLSearchParams(window.location.search).get('debug') === '1'
  if (!forced && !maybeTelegramMiniApp()) return null

  const app = window.Telegram?.WebApp
  const user = app?.initDataUnsafe?.user
  const fullName = [user?.first_name, user?.last_name].filter(Boolean).join(' ')
  const rows: Array<[string, string]> = [
    ['SDK window.Telegram.WebApp', app ? 'есть' : 'нет'],
    ['TelegramWebviewProxy', window.TelegramWebviewProxy ? 'есть' : 'нет'],
    ['#tgWebApp при загрузке', telegramLaunchParamsAtLoad() ? 'есть' : 'нет'],
    ['#tgWebApp сейчас', telegramLaunchParams() ? 'есть' : 'нет'],
    ['initData', app?.initData ? `${app.initData.length} символов` : 'пусто'],
    ['user.id', user?.id ? String(user.id) : '—'],
    ['username', user?.username ? `@${user.username}` : '—'],
    ['имя', fullName || '—'],
    ['start_param', app?.initDataUnsafe?.start_param || '—'],
    ['platform / version', [app?.platform, app?.version].filter(Boolean).join(' / ') || '—'],
    ['определение Mini App', probe],
    ['user-agent', navigator.userAgent],
  ]

  return (
    <div className="mt-4 rounded-card border border-dashed border-border bg-surface p-4">
      <p className="mb-2 text-[12px] font-semibold uppercase tracking-wide text-muted">debug</p>
      <dl className="flex flex-col gap-1">
        {rows.map(([label, value]) => (
          <div key={label} className="flex gap-2 text-[12px]">
            <dt className="shrink-0 text-muted">{label}</dt>
            <dd className="ml-auto break-all text-right font-mono text-ink">{value}</dd>
          </div>
        ))}
      </dl>
      {app?.initData && (
        <details className="mt-2">
          <summary className="cursor-pointer text-[12px] text-muted">initData целиком</summary>
          <pre className="mt-1 max-h-40 overflow-auto whitespace-pre-wrap break-all rounded-[8px] bg-surface-2 p-2 text-[11px] text-ink">
            {app.initData}
          </pre>
        </details>
      )}
    </div>
  )
}
