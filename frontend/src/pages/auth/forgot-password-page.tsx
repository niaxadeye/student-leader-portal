import { Link } from 'react-router-dom'
import { LifeBuoy } from 'lucide-react'

// Self-service сброса нет: пароль сбрасывает администратор конкурса (SITE.md §7).
export function ForgotPasswordPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-surface-2 px-4">
      <div className="w-full max-w-[400px] text-center">
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-brand-subtle">
          <LifeBuoy className="h-6 w-6 text-brand" />
        </div>
        <h1 className="text-[24px] font-bold text-ink">Забыли пароль?</h1>
        <p className="mt-2 text-[15px] leading-relaxed text-muted">
          Восстановление пароля выполняет администратор конкурса. Обратитесь к дирекции,
          чтобы сбросить пароль — вам выдадут временный, который потребуется сменить при входе.
        </p>
        <Link
          to="/login"
          className="mt-6 inline-block text-[14px] font-medium text-brand hover:text-brand-dark"
        >
          ← Вернуться ко входу
        </Link>
      </div>
    </div>
  )
}
