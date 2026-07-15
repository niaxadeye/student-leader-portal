import { useId } from 'react'
import { HelpCircle } from 'lucide-react'
import { cn } from '@/shared/lib/cn'

interface FieldProps {
  label: string
  /** Справка к полю (SITE.md §11: подсказка у каждого поля). */
  helpText?: string
  description?: string
  required?: boolean
  error?: string
  className?: string
  children: (props: { id: string; 'aria-invalid': boolean }) => React.ReactNode
}

/** Обёртка поля: label + описание + help_text + ошибка. Единый a11y-контракт. */
export function Field({
  label,
  helpText,
  description,
  required,
  error,
  className,
  children,
}: FieldProps) {
  const id = useId()
  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <div className="flex items-center gap-1.5">
        <label htmlFor={id} className="text-[14px] font-medium text-ink">
          {label}
          {required && <span className="ml-0.5 text-danger">*</span>}
        </label>
        {helpText && (
          <span className="group relative inline-flex">
            <HelpCircle className="h-4 w-4 text-muted-2" aria-label={helpText} tabIndex={0} />
            <span className="pointer-events-none absolute bottom-full left-1/2 z-10 mb-1.5 hidden w-56 -translate-x-1/2 rounded-md bg-ink px-2.5 py-1.5 text-[12px] leading-snug text-white group-hover:block group-focus-within:block">
              {helpText}
            </span>
          </span>
        )}
      </div>
      {description && <p className="text-[13px] text-muted">{description}</p>}
      {children({ id, 'aria-invalid': !!error })}
      {error && <p className="text-[13px] text-danger">{error}</p>}
    </div>
  )
}
