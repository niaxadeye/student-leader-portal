import { useEffect, useState } from 'react'
import { formatDateTime } from '@/shared/lib/format'
import type { ScoreCorrection } from '@/entities/evaluation/types'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Textarea } from '@/shared/ui/input'
import { toast } from 'sonner'

export type PendingScoreOverride = {
  kind: 'CRITERION' | 'NUMERIC'
  contestantUserId: string
  contestantName: string
  juryUserId?: string
  juryName?: string
  criterionId?: string
  criterionTitle?: string
  min?: number
  max?: number
  oldScore: number | null
  newScore: number | null
}

function fmtScore(n: number | null | undefined): string {
  if (n == null) return 'пусто'
  return Number.isInteger(n) ? String(n) : n.toFixed(1)
}

export function ScoreOverrideHint({ visible }: { visible: boolean }) {
  if (!visible) return null
  return (
    <p className="text-[13px] text-muted">
      Мегаадмин может править любые баллы. Правка применится только после указания причины и попадёт в журнал ниже.
      Для критериев раскройте строку конкурсанта.
    </p>
  )
}

export function ScoreCorrectionsLog({ items }: { items: ScoreCorrection[] }) {
  return (
    <Card className="overflow-hidden">
      <CardBody className="flex flex-col gap-3 py-5">
        <div>
          <h2 className="text-[18px] font-semibold text-ink">Журнал правок баллов</h2>
          <p className="mt-1 text-[13px] text-muted">
            Только правки мегаадмина с указанием причины. Обычные оценки жюри сюда не пишутся.
          </p>
        </div>
        {items.length === 0 ? (
          <p className="text-[14px] text-muted">Правок пока нет.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[720px] text-left text-[13px]">
              <thead className="text-[12px] uppercase tracking-wide text-muted-2">
                <tr className="border-b border-border">
                  <th className="px-3 py-2 font-medium">Когда</th>
                  <th className="px-3 py-2 font-medium">Кто</th>
                  <th className="px-3 py-2 font-medium">Конкурсант</th>
                  <th className="px-3 py-2 font-medium">Что</th>
                  <th className="px-3 py-2 font-medium">Было → стало</th>
                  <th className="px-3 py-2 font-medium">Причина</th>
                </tr>
              </thead>
              <tbody>
                {items.map((c) => (
                  <tr key={c.id} className="border-b border-border last:border-0 align-top">
                    <td className="whitespace-nowrap px-3 py-2 text-muted">{formatDateTime(c.created_at)}</td>
                    <td className="px-3 py-2">{c.actor_name}</td>
                    <td className="px-3 py-2">{c.contestant_name}</td>
                    <td className="px-3 py-2">
                      {c.kind === 'NUMERIC'
                        ? 'Числовой результат'
                        : [c.criterion_title, c.jury_name].filter(Boolean).join(' · ')}
                    </td>
                    <td className="whitespace-nowrap px-3 py-2 tabular-nums">
                      {fmtScore(c.old_score)} → {fmtScore(c.new_score)}
                    </td>
                    <td className="px-3 py-2 text-ink">{c.reason}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardBody>
    </Card>
  )
}

export function ScoreOverrideDialog({
  pending,
  loading,
  onClose,
  onConfirm,
}: {
  pending: PendingScoreOverride | null
  loading: boolean
  onClose: () => void
  onConfirm: (reason: string) => void
}) {
  const [reason, setReason] = useState('')
  const open = pending != null
  const trimmed = reason.trim()
  const tooShort = trimmed.length < 5

  useEffect(() => {
    if (pending) setReason('')
  }, [pending])

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          setReason('')
          onClose()
        }
      }}
    >
      <DialogContent
        title="Правка балла"
        description="Укажите причину — без неё изменение не сохранится и попадёт в журнал на этой вкладке."
      >
        {pending ? (
          <div className="flex flex-col gap-4">
            <p className="text-[14px] text-ink">
              {pending.contestantName}
              {pending.kind === 'CRITERION' ? (
                <>
                  {' · '}
                  {pending.criterionTitle}
                  {pending.juryName ? ` · ${pending.juryName}` : ''}
                </>
              ) : (
                ' · числовой результат'
              )}
            </p>
            <p className="text-[14px] tabular-nums text-muted">
              {fmtScore(pending.oldScore)} → {fmtScore(pending.newScore)}
            </p>
            <Field label="Причина" required helpText="Не меньше 5 символов. Её увидят все, у кого есть вкладка «Оценки».">
              {(p) => (
                <Textarea
                  {...p}
                  rows={3}
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="Почему меняете балл"
                />
              )}
            </Field>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="secondary" onClick={onClose} disabled={loading}>
                Отмена
              </Button>
              <Button
                type="button"
                loading={loading}
                disabled={tooShort}
                onClick={() => onConfirm(trimmed)}
              >
                Применить
              </Button>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  )
}

export function OverrideScoreInput({
  value,
  min,
  max,
  disabled,
  ariaLabel,
  onCommit,
}: {
  value: number | null
  min: number
  max: number
  disabled?: boolean
  ariaLabel: string
  onCommit: (next: number | null) => void
}) {
  const [text, setText] = useState(value == null ? '' : String(value))
  const [focused, setFocused] = useState(false)

  useEffect(() => {
    if (!focused) setText(value == null ? '' : String(value))
  }, [value, focused])

  function commit() {
    const trimmed = text.trim().replace(',', '.')
    if (trimmed === '') {
      if (value != null) onCommit(null)
      return
    }
    const n = Number(trimmed)
    if (!Number.isFinite(n) || n < min || n > max) {
      toast.error(`Балл — число от ${min} до ${max}`)
      setText(value == null ? '' : String(value))
      return
    }
    if (value != null && Math.abs(value - n) < 1e-9) return
    onCommit(n)
  }

  return (
    <input
      className="h-8 w-[4.5rem] rounded-[8px] border border-border bg-surface px-2 text-[13px] tabular-nums text-ink focus-visible:border-brand focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand/20"
      type="number"
      min={min}
      max={max}
      step="any"
      disabled={disabled}
      value={text}
      aria-label={ariaLabel}
      onChange={(e) => setText(e.target.value)}
      onFocus={() => setFocused(true)}
      onBlur={() => {
        commit()
        setFocused(false)
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter') {
          e.preventDefault()
          ;(e.target as HTMLInputElement).blur()
        }
      }}
    />
  )
}
