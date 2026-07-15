import { forwardRef } from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { Loader2 } from 'lucide-react'
import { cn } from '@/shared/lib/cn'

// Варианты кнопок строго по DESIGN.md §4. Радиус 12px, без pill.
const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 rounded-btn font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none whitespace-nowrap',
  {
    variants: {
      variant: {
        primary: 'bg-brand text-white hover:bg-brand-dark',
        outline: 'bg-surface text-brand-dark border border-brand-dark hover:bg-brand-subtle',
        subtle: 'bg-brand-subtle text-brand hover:bg-brand/20',
        secondary: 'bg-muted/10 text-ink hover:bg-muted/20',
        ghost: 'text-ink hover:bg-muted/10',
      },
      size: {
        md: 'h-11 px-4 text-[16px]',
        sm: 'h-9 px-3 text-[14px]',
        icon: 'h-11 w-11',
      },
    },
    defaultVariants: { variant: 'primary', size: 'md' },
  },
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
  loading?: boolean
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild, loading, disabled, children, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        ref={ref}
        className={cn(buttonVariants({ variant, size }), className)}
        disabled={disabled || loading}
        {...props}
      >
        {loading && <Loader2 className="h-4 w-4 animate-spin" aria-hidden />}
        {children}
      </Comp>
    )
  },
)
Button.displayName = 'Button'
