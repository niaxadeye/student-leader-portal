import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/shared/lib/cn'

// Бейджи по DESIGN.md §4.2 + маппинг статусов формы из SITE.md §8.
const badgeVariants = cva(
  'inline-flex items-center gap-1 rounded-badge px-2 py-0.5 text-[12px] font-medium',
  {
    variants: {
      tone: {
        neutral: 'bg-muted/12 text-[#484b5e]',
        success: 'bg-success/[0.16] text-success-dark',
        brand: 'bg-brand-subtle text-brand',
        warning: 'bg-amber-100 text-amber-700',
        danger: 'bg-danger/10 text-danger',
      },
    },
    defaultVariants: { tone: 'neutral' },
  },
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, tone, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />
}
