import { forwardRef } from 'react'
import * as CheckboxPrimitive from '@radix-ui/react-checkbox'
import * as RadioPrimitive from '@radix-ui/react-radio-group'
import { Check } from 'lucide-react'
import { cn } from '@/shared/lib/cn'

export const Checkbox = forwardRef<
  React.ElementRef<typeof CheckboxPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>
>(({ className, ...props }, ref) => (
  <CheckboxPrimitive.Root
    ref={ref}
    className={cn(
      'flex h-5 w-5 shrink-0 items-center justify-center rounded-[6px] border border-border bg-surface data-[state=checked]:border-brand data-[state=checked]:bg-brand',
      className,
    )}
    {...props}
  >
    <CheckboxPrimitive.Indicator>
      <Check className="h-3.5 w-3.5 text-white" strokeWidth={3} />
    </CheckboxPrimitive.Indicator>
  </CheckboxPrimitive.Root>
))
Checkbox.displayName = 'Checkbox'

export const RadioGroup = RadioPrimitive.Root

export const RadioItem = forwardRef<
  React.ElementRef<typeof RadioPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof RadioPrimitive.Item>
>(({ className, ...props }, ref) => (
  <RadioPrimitive.Item
    ref={ref}
    className={cn(
      'flex h-5 w-5 shrink-0 items-center justify-center rounded-full border border-border bg-surface data-[state=checked]:border-brand',
      className,
    )}
    {...props}
  >
    <RadioPrimitive.Indicator className="h-2.5 w-2.5 rounded-full bg-brand" />
  </RadioPrimitive.Item>
))
RadioItem.displayName = 'RadioItem'
