import { useEffect, useState } from 'react'
import { RefreshCw, ShieldCheck } from 'lucide-react'
import { useParticipantQRCode } from '@/entities/lecture/queries'
import { useParticipantAuth } from '@/entities/event-participant/auth-context'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { ErrorState, Skeleton } from '@/shared/ui/states'

export function ParticipantQRPage() {
  const { session } = useParticipantAuth()
  const code = useParticipantQRCode(session?.event.id, session?.participant.id)
  const [image, setImage] = useState<string | null>(null)
  const [secondsLeft, setSecondsLeft] = useState(0)
  const token = code.data?.token
  const expiresAt = code.data?.expires_at

  useEffect(() => {
    let cancelled = false
    if (!token) {
      setImage(null)
      return
    }
    void import('qrcode').then(async ({ toDataURL }) => {
      const dataURL = await toDataURL(token, {
        width: 360,
        margin: 2,
        errorCorrectionLevel: 'M',
        color: { dark: '#151827', light: '#ffffff' },
      })
      if (!cancelled) setImage(dataURL)
    })
    return () => {
      cancelled = true
    }
  }, [token])

  useEffect(() => {
    if (!expiresAt) return
    const update = () => {
      setSecondsLeft(Math.max(0, Math.ceil((new Date(expiresAt).getTime() - Date.now()) / 1000)))
    }
    update()
    const timer = window.setInterval(update, 1000)
    return () => window.clearInterval(timer)
  }, [expiresAt])

  return (
    <div className="mx-auto max-w-lg">
      <div className="mb-5 text-center">
        <p className="text-[13px] font-medium text-brand">Посещаемость</p>
        <h1 className="mt-1 text-[28px] font-bold text-ink">Мой QR-код</h1>
        <p className="mt-2 text-[14px] text-muted">
          Покажите код организатору на входе в лекционный зал.
        </p>
      </div>

      <Card className="overflow-hidden">
        <CardBody className="flex flex-col items-center p-6 sm:p-8">
          {code.isLoading && <Skeleton className="h-80 w-80 max-w-full" />}
          {code.isError && <ErrorState onRetry={() => code.refetch()} />}
          {code.data && (
            <>
              <div className="flex w-full items-center justify-between gap-3">
                <Badge tone="success">
                  <ShieldCheck className="h-3 w-3" /> Подписан сервером
                </Badge>
                <span
                  className={`text-[13px] font-medium ${secondsLeft <= 8 ? 'text-danger' : 'text-muted'}`}
                >
                  {secondsLeft > 0 ? `Обновление через ${secondsLeft} сек.` : 'Обновляется…'}
                </span>
              </div>
              <div className="my-5 flex aspect-square w-full max-w-[360px] items-center justify-center rounded-[20px] bg-white p-2 shadow-subtle">
                {image ? (
                  <img src={image} alt="Одноразовый QR-код участника" className="h-full w-full" />
                ) : (
                  <Skeleton className="h-full w-full" />
                )}
              </div>
              <Button variant="secondary" loading={code.isFetching} onClick={() => code.refetch()}>
                <RefreshCw className="h-4 w-4" /> Обновить код
              </Button>
              <p className="mt-4 text-center text-[12px] leading-relaxed text-muted">
                Код одноразовый и действует меньше минуты. В нём нет ФИО, даты рождения или
                открытого идентификатора участника.
              </p>
            </>
          )}
        </CardBody>
      </Card>
    </div>
  )
}
