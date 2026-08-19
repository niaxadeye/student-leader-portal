import { ArrowLeft } from 'lucide-react'

export function BackButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center justify-center gap-1.5 text-[14px] text-muted hover:text-ink"
    >
      <ArrowLeft className="h-4 w-4" aria-hidden />
      Выбрать другой тип входа
    </button>
  )
}
