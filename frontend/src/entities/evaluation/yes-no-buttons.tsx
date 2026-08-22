import { cn } from '@/shared/lib/cn'

export type YesNoAnswer = 'YES' | 'NO'

export function answerLabel(value?: string | null): string {
  if (value === 'YES') return 'Да'
  if (value === 'NO') return 'Нет'
  return '—'
}

export function YesNoButtons({
  value,
  disabled,
  onSelect,
}: {
  value?: YesNoAnswer | string | null
  disabled?: boolean
  onSelect: (answer: YesNoAnswer) => void
}) {
  return (
    <div className="flex gap-2">
      <button
        type="button"
        disabled={disabled}
        onClick={() => onSelect('YES')}
        className={cn(
          'inline-flex h-9 items-center justify-center rounded-btn px-3 text-[14px] font-medium transition-colors disabled:pointer-events-none disabled:opacity-50',
          value === 'YES'
            ? 'bg-success text-white hover:bg-success-dark'
            : 'border border-success bg-surface text-success hover:bg-success/10',
        )}
      >
        Да
      </button>
      <button
        type="button"
        disabled={disabled}
        onClick={() => onSelect('NO')}
        className={cn(
          'inline-flex h-9 items-center justify-center rounded-btn px-3 text-[14px] font-medium transition-colors disabled:pointer-events-none disabled:opacity-50',
          value === 'NO'
            ? 'bg-danger text-white hover:bg-danger/90'
            : 'border border-danger bg-surface text-danger hover:bg-danger/10',
        )}
      >
        Нет
      </button>
    </div>
  )
}
