import { Toaster } from 'sonner'
import { CheckCircle2, AlertCircle, Info } from 'lucide-react'

// Уведомления на sonner. Вызовы — через `toast.success/error/info` из 'sonner'.
// Вид сохранён прежним (Kraken-токены): surface/border/rounded-card/shadow-subtle,
// иконки lucide по тону, позиция bottom-right, авто-скрытие ~4с.
const toneIcon = 'mt-0.5 h-5 w-5 shrink-0'

export function AppToaster() {
  return (
    <Toaster
      position="bottom-right"
      duration={4000}
      icons={{
        success: <CheckCircle2 className={`${toneIcon} text-success`} aria-hidden />,
        error: <AlertCircle className={`${toneIcon} text-danger`} aria-hidden />,
        info: <Info className={`${toneIcon} text-brand`} aria-hidden />,
      }}
      toastOptions={{
        unstyled: true,
        classNames: {
          toast:
            'flex w-full items-start gap-3 rounded-card border border-border bg-surface p-4 shadow-subtle',
          title: 'text-[14px] font-semibold text-ink',
          description: 'mt-0.5 text-[14px] text-muted',
          closeButton: 'text-muted-2 hover:text-ink',
        },
      }}
    />
  )
}
