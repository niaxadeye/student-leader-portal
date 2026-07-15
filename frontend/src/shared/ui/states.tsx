import { cn } from '@/shared/lib/cn'
import { Inbox, AlertTriangle } from 'lucide-react'

export function Skeleton({ className }: { className?: string }) {
  return <div className={cn('animate-pulse rounded-md bg-muted/15', className)} />
}

export function EmptyState({
  title,
  description,
  icon: Icon = Inbox,
  action,
}: {
  title: string
  description?: string
  icon?: React.ComponentType<{ className?: string }>
  action?: React.ReactNode
}) {
  return (
    <div className="flex flex-col items-center justify-center rounded-card border border-dashed border-border bg-surface px-6 py-12 text-center">
      <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-brand-subtle">
        <Icon className="h-6 w-6 text-brand" />
      </div>
      <p className="text-[16px] font-medium text-ink">{title}</p>
      {description && <p className="mt-1 max-w-sm text-[14px] text-muted">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  )
}

export function ErrorState({
  title = 'Не удалось загрузить данные',
  description = 'Попробуйте обновить. Если ошибка повторяется — обратитесь к администратору.',
  onRetry,
}: {
  title?: string
  description?: string
  onRetry?: () => void
}) {
  return (
    <div className="flex flex-col items-center justify-center rounded-card border border-dashed border-danger/30 bg-surface px-6 py-12 text-center">
      <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-danger/10">
        <AlertTriangle className="h-6 w-6 text-danger" />
      </div>
      <p className="text-[16px] font-medium text-ink">{title}</p>
      <p className="mt-1 max-w-sm text-[14px] text-muted">{description}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-4 rounded-btn border border-border px-3.5 py-2 text-[14px] font-medium text-ink hover:bg-muted/10"
        >
          Обновить
        </button>
      )}
    </div>
  )
}
