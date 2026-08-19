import { Target } from 'lucide-react'
import { cn } from '@/shared/lib/cn'

export function TaskIcon({
  url,
  className,
}: {
  url: string | null | undefined
  className?: string
}) {
  if (url) {
    return (
      <img
        src={url}
        alt=""
        className={cn('h-[96px] w-[96px] shrink-0 rounded-[12px] object-cover', className)}
      />
    )
  }
  return (
    <div
      className={cn(
        'flex h-[96px] w-[96px] shrink-0 items-center justify-center rounded-[12px] bg-brand-subtle',
        className,
      )}
    >
      <Target className="h-8 w-8 text-brand" />
    </div>
  )
}
