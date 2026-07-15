import { Card } from '@/shared/ui/card'
import { cn } from '@/shared/lib/cn'

export function StatCard({
  label,
  value,
  icon: Icon,
  accent,
}: {
  label: string
  value: string | number
  icon: React.ComponentType<{ className?: string }>
  accent?: boolean
}) {
  return (
    <Card className="p-4">
      <div className="flex items-center gap-3">
        <div
          className={cn(
            'flex h-10 w-10 items-center justify-center rounded-btn',
            accent ? 'bg-brand text-white' : 'bg-brand-subtle text-brand',
          )}
        >
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <p className="text-[24px] font-bold leading-none text-ink">{value}</p>
          <p className="mt-1 text-[13px] text-muted">{label}</p>
        </div>
      </div>
    </Card>
  )
}
