import { useState } from 'react'
import { Copy, Check, KeyRound } from 'lucide-react'

/** Показ временного пароля один раз с копированием. Пароль больше нигде не хранится. */
export function TempPasswordNote({ login, password }: { login: string; password: string }) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    try {
      await navigator.clipboard.writeText(password)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* clipboard недоступен — пользователь скопирует вручную */
    }
  }

  return (
    <div className="rounded-card border border-brand/30 bg-brand-subtle/50 p-4">
      <div className="flex items-center gap-2 text-[14px] font-medium text-ink">
        <KeyRound className="h-4 w-4 text-brand" />
        Временный пароль
      </div>
      <p className="mt-1 text-[13px] text-muted">
        Передайте его пользователю <span className="font-medium text-ink">{login}</span>. Он показывается
        один раз — при первом входе пароль нужно сменить.
      </p>
      <div className="mt-3 flex items-center gap-2">
        <code className="flex-1 rounded-btn border border-border bg-surface px-3 py-2 font-mono text-[15px] text-ink">
          {password}
        </code>
        <button
          onClick={copy}
          className="inline-flex h-10 items-center gap-1.5 rounded-btn border border-border px-3 text-[14px] text-ink hover:bg-muted/10"
        >
          {copied ? <Check className="h-4 w-4 text-success" /> : <Copy className="h-4 w-4" />}
          {copied ? 'Скопировано' : 'Копировать'}
        </button>
      </div>
    </div>
  )
}
