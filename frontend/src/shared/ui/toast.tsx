import { createContext, useCallback, useContext, useState } from 'react'
import * as ToastPrimitive from '@radix-ui/react-toast'
import { CheckCircle2, AlertCircle, Info, X } from 'lucide-react'
import { cn } from '@/shared/lib/cn'

type ToastTone = 'success' | 'error' | 'info'
type ToastItem = { id: number; title: string; description?: string; tone: ToastTone }

const ToastCtx = createContext<(t: Omit<ToastItem, 'id'>) => void>(() => {})
export const useToast = () => useContext(ToastCtx)

const icons = { success: CheckCircle2, error: AlertCircle, info: Info }
const toneCls = {
  success: 'text-success',
  error: 'text-danger',
  info: 'text-brand',
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([])
  const push = useCallback((t: Omit<ToastItem, 'id'>) => {
    setItems((prev) => [...prev, { ...t, id: Date.now() + Math.random() }])
  }, [])

  return (
    <ToastCtx.Provider value={push}>
      <ToastPrimitive.Provider swipeDirection="right" duration={4000}>
        {children}
        {items.map(({ id, title, description, tone }) => {
          const Icon = icons[tone]
          return (
            <ToastPrimitive.Root
              key={id}
              onOpenChange={(o) => !o && setItems((p) => p.filter((i) => i.id !== id))}
              className="flex items-start gap-3 rounded-card border border-border bg-surface p-4 shadow-subtle data-[state=open]:animate-in data-[state=open]:slide-in-from-right"
            >
              <Icon className={cn('mt-0.5 h-5 w-5 shrink-0', toneCls[tone])} aria-hidden />
              <div className="flex-1">
                <ToastPrimitive.Title className="text-[14px] font-semibold text-ink">
                  {title}
                </ToastPrimitive.Title>
                {description && (
                  <ToastPrimitive.Description className="mt-0.5 text-[14px] text-muted">
                    {description}
                  </ToastPrimitive.Description>
                )}
              </div>
              <ToastPrimitive.Close aria-label="Закрыть" className="text-muted-2 hover:text-ink">
                <X className="h-4 w-4" />
              </ToastPrimitive.Close>
            </ToastPrimitive.Root>
          )
        })}
        <ToastPrimitive.Viewport className="fixed bottom-0 right-0 z-50 flex w-full max-w-sm flex-col gap-2 p-4" />
      </ToastPrimitive.Provider>
    </ToastCtx.Provider>
  )
}
