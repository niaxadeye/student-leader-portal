import { useEffect, useState } from 'react'
import { Coins, History } from 'lucide-react'
import { toast } from 'sonner'
import type { EventParticipant } from '@/entities/event-participant/types'
import { formatPoints, signedPoints } from '@/entities/points/format'
import {
  useAdjustAdminParticipantPoints,
  useAdminParticipantPoints,
} from '@/entities/points/queries'
import { ApiRequestError } from '@/shared/api/client'
import { formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { Skeleton } from '@/shared/ui/states'

function newIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return `admin-adjustment:${crypto.randomUUID()}`
  }
  return `admin-adjustment:${Date.now()}:${Math.random().toString(36).slice(2)}`
}

function adjustmentError(error: unknown): string {
  if (error instanceof TypeError) {
    return 'Ответ сервера не получен. Повторите отправку — двойного начисления не произойдёт.'
  }
  if (error instanceof ApiRequestError) {
    if (error.code === 'IDEMPOTENCY_CONFLICT') {
      return 'Защитный ключ уже использован с другими данными. Закройте окно и создайте новую корректировку.'
    }
    if (error.code === 'VALIDATION_ERROR') {
      return 'Укажите ненулевое целое количество баллов и обязательную причину.'
    }
    if (error.code === 'NETWORK_ERROR' || error.status === 0) {
      return 'Ответ сервера не получен. Повторите отправку — двойного начисления не произойдёт.'
    }
  }
  return 'Не удалось выполнить корректировку.'
}

export function EventParticipantPointsDialog({
  contestId,
  participant,
  onOpenChange,
}: {
  contestId: string
  participant: EventParticipant | null
  onOpenChange: (open: boolean) => void
}) {
  const participantId = participant?.id ?? ''
  const overview = useAdminParticipantPoints(contestId, participantId, participant !== null)
  const adjustment = useAdjustAdminParticipantPoints(contestId, participantId)
  const [amount, setAmount] = useState('')
  const [reason, setReason] = useState('')
  const [idempotencyKey, setIdempotencyKey] = useState(newIdempotencyKey)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!participant) return
    setAmount('')
    setReason('')
    setError('')
    setIdempotencyKey(newIdempotencyKey())
  }, [participant])

  function close() {
    if (!adjustment.isPending) onOpenChange(false)
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()
    setError('')
    const parsedAmount = Number(amount)
    const cleanReason = reason.trim()
    if (!Number.isSafeInteger(parsedAmount) || parsedAmount === 0 || cleanReason.length < 3) {
      setError('Укажите ненулевое целое количество баллов и причину не короче 3 символов.')
      return
    }
    adjustment.mutate(
      { amount: parsedAmount, reason: cleanReason, idempotency_key: idempotencyKey },
      {
        onSuccess: (result) => {
          toast.success(
            result.replayed ? 'Корректировка уже была применена' : 'Баллы скорректированы',
            {
              description: `Доступно: ${formatPoints(result.balance.available_points)}`,
            },
          )
          onOpenChange(false)
        },
        onError: (requestError) => setError(adjustmentError(requestError)),
      },
    )
  }

  return (
    <Dialog open={participant !== null} onOpenChange={(open) => (open ? undefined : close())}>
      <DialogContent
        className="max-w-2xl"
        title="Баллы участника"
        description={participant?.full_name ?? ''}
      >
        {overview.isLoading ? (
          <Skeleton className="h-64 w-full" />
        ) : overview.isError || !overview.data ? (
          <div className="rounded-[12px] bg-danger/10 p-4 text-[14px] text-danger">
            Не удалось загрузить баланс. Закройте окно и попробуйте снова.
          </div>
        ) : (
          <div className="flex flex-col gap-5">
            <div className="grid grid-cols-3 gap-2">
              <BalanceCell label="Баланс" value={overview.data.balance.ledger_balance} />
              <BalanceCell label="В резерве" value={overview.data.balance.reserved_points} />
              <BalanceCell label="Доступно" value={overview.data.balance.available_points} accent />
            </div>

            <form onSubmit={submit} className="rounded-[12px] border border-border p-4" noValidate>
              <div className="mb-4 flex items-center gap-2">
                <Coins className="h-5 w-5 text-brand" />
                <h3 className="text-[16px] font-semibold text-ink">Ручная корректировка</h3>
              </div>
              <div className="grid gap-4 sm:grid-cols-[160px_1fr]">
                <Field label="Сумма" description="Например: 100 или -50" required>
                  {(props) => (
                    <Input
                      {...props}
                      type="number"
                      step="1"
                      value={amount}
                      onChange={(event) => setAmount(event.target.value)}
                      placeholder="+100"
                    />
                  )}
                </Field>
                <Field label="Причина" description="Сохранится в ledger и audit" required>
                  {(props) => (
                    <Textarea
                      {...props}
                      value={reason}
                      onChange={(event) => setReason(event.target.value)}
                      maxLength={1000}
                      placeholder="Причина начисления или списания"
                      className="min-h-20"
                    />
                  )}
                </Field>
              </div>
              {error && (
                <div
                  role="alert"
                  className="mt-3 rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[13px] text-danger"
                >
                  {error}
                </div>
              )}
              <div className="mt-4 flex justify-end gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  disabled={adjustment.isPending}
                  onClick={close}
                >
                  Отмена
                </Button>
                <Button type="submit" loading={adjustment.isPending}>
                  Применить
                </Button>
              </div>
            </form>

            <div>
              <div className="mb-2 flex items-center gap-2">
                <History className="h-4 w-4 text-muted" />
                <h3 className="text-[14px] font-semibold text-ink">Последние операции</h3>
              </div>
              {!overview.data.entries.length ? (
                <p className="rounded-[12px] bg-surface-2 px-4 py-5 text-center text-[13px] text-muted">
                  Операций пока нет.
                </p>
              ) : (
                <div className="max-h-52 overflow-auto rounded-[12px] border border-border">
                  {overview.data.entries.map((entry) => (
                    <div
                      key={entry.id}
                      className="flex items-start justify-between gap-4 border-b border-border px-3.5 py-3 last:border-0"
                    >
                      <div>
                        <p className="text-[13px] font-medium text-ink">{entry.description}</p>
                        <p className="mt-0.5 text-[11px] text-muted">
                          {formatDateTime(entry.created_at)}
                        </p>
                      </div>
                      <Badge tone={entry.amount > 0 ? 'success' : 'danger'}>
                        {signedPoints(entry.amount)}
                      </Badge>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function BalanceCell({ label, value, accent }: { label: string; value: number; accent?: boolean }) {
  return (
    <div className={`rounded-[12px] p-3 ${accent ? 'bg-brand-subtle' : 'bg-surface-2'}`}>
      <p className="text-[11px] text-muted">{label}</p>
      <p className={`mt-1 text-[20px] font-bold ${accent ? 'text-brand' : 'text-ink'}`}>
        {formatPoints(value)}
      </p>
    </div>
  )
}
