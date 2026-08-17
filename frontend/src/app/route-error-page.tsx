import { useEffect } from 'react'
import { isRouteErrorResponse, useRouteError } from 'react-router-dom'
import { isModuleLoadError, markModuleReloadAttempt } from '@/shared/lib/module-reload'

export function RouteErrorPage() {
  const error = useRouteError()
  const moduleLoadFailed = isModuleLoadError(error)

  useEffect(() => {
    if (!moduleLoadFailed || !markModuleReloadAttempt()) return
    const timeout = window.setTimeout(() => window.location.reload(), 100)
    return () => window.clearTimeout(timeout)
  }, [moduleLoadFailed])

  const status = isRouteErrorResponse(error) ? error.status : null

  return (
    <main className="flex min-h-screen items-center justify-center bg-surface-2 px-5 py-10">
      <section
        className="w-full max-w-md rounded-card border border-border bg-surface p-7 text-center shadow-sm"
        aria-live="polite"
      >
        <p className="text-[12px] font-semibold uppercase tracking-[0.12em] text-brand">
          {status ? `Ошибка ${status}` : 'Student Leader'}
        </p>
        <h1 className="mt-3 text-[24px] font-bold text-ink">
          {moduleLoadFailed ? 'Обновляем приложение' : 'Не удалось открыть страницу'}
        </h1>
        <p className="mt-3 text-[14px] leading-6 text-muted">
          {moduleLoadFailed
            ? 'Safari не смог загрузить файл интерфейса. Перезагрузите страницу, чтобы получить актуальную версию.'
            : 'Попробуйте перезагрузить страницу. Если ошибка повторится, вернитесь на главную.'}
        </p>
        <div className="mt-6 flex flex-col gap-2 sm:flex-row sm:justify-center">
          <button
            type="button"
            onClick={() => window.location.reload()}
            className="inline-flex h-11 items-center justify-center rounded-[10px] bg-brand px-5 text-[14px] font-semibold text-white hover:bg-brand-hover"
          >
            Перезагрузить
          </button>
          <a
            href="/"
            className="inline-flex h-11 items-center justify-center rounded-[10px] border border-border bg-surface px-5 text-[14px] font-semibold text-ink hover:bg-surface-2"
          >
            На главную
          </a>
        </div>
      </section>
    </main>
  )
}
