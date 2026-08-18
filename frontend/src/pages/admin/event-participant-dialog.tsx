import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  useCreateEventParticipant,
  useEventDirections,
  useUpdateEventParticipant,
} from '@/entities/event-participant/admin-queries'
import type { AdminParticipantInput } from '@/entities/event-participant/admin-types'
import type { EventParticipant } from '@/entities/event-participant/types'
import { ApiRequestError } from '@/shared/api/client'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/select'

function localToday(): string {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
}

function mutationError(error: unknown): string {
  if (error instanceof ApiRequestError) {
    if (error.code === 'PARTICIPANT_IDENTIFIER_TAKEN') {
      return 'Профбилет или barcode уже назначен другому участнику этого мероприятия.'
    }
    if (error.code === 'VALIDATION_ERROR') return 'Проверьте ФИО и дату рождения.'
  }
  return 'Не удалось сохранить участника. Попробуйте ещё раз.'
}

export function EventParticipantDialog({
  contestId,
  participant,
  open,
  onOpenChange,
}: {
  contestId: string
  participant: EventParticipant | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const create = useCreateEventParticipant(contestId)
  const update = useUpdateEventParticipant(contestId)
  const [fullName, setFullName] = useState('')
  const [birthDate, setBirthDate] = useState('')
  const [unionCard, setUnionCard] = useState('')
  const [sksBarcode, setSKSBarcode] = useState('')
  const [directionId, setDirectionId] = useState('none')
  const [error, setError] = useState('')
  const directions = useEventDirections(open ? contestId : undefined)
  const editing = participant !== null
  const isPending = create.isPending || update.isPending

  useEffect(() => {
    if (!open) return
    setFullName(participant?.full_name ?? '')
    setBirthDate(participant?.birth_date.slice(0, 10) ?? '')
    setUnionCard(participant?.union_card_number ?? '')
    setSKSBarcode(participant?.sks_barcode ?? '')
    setDirectionId(participant?.direction_id ?? 'none')
    setError('')
  }, [open, participant])

  function close() {
    if (!isPending) onOpenChange(false)
  }

  function submit(event: React.FormEvent) {
    event.preventDefault()
    setError('')
    const name = fullName.trim().replace(/\s+/g, ' ')
    if (!name || !birthDate) {
      setError('ФИО и дата рождения обязательны.')
      return
    }
    if (birthDate > localToday()) {
      setError('Дата рождения не может быть в будущем.')
      return
    }

    const input: AdminParticipantInput = {
      full_name: name,
      birth_date: birthDate,
      union_card_number: unionCard.trim() || undefined,
      sks_barcode: sksBarcode.trim() || undefined,
      direction_id: directionId === 'none' ? null : directionId,
    }
    const options = {
      onSuccess: () => {
        toast.success(editing ? 'Участник обновлён' : 'Участник добавлен')
        onOpenChange(false)
      },
      onError: (requestError: unknown) => setError(mutationError(requestError)),
    }
    if (participant) {
      update.mutate({ participantId: participant.id, input }, options)
    } else {
      create.mutate(input, options)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? onOpenChange(true) : close())}>
      <DialogContent
        title={editing ? 'Редактировать участника' : 'Новый участник мероприятия'}
        description="Это отдельный участник мероприятия — учётная запись User не создаётся."
      >
        <form onSubmit={submit} className="flex flex-col gap-4" noValidate>
          <Field label="ФИО" required>
            {(props) => (
              <Input
                {...props}
                autoFocus
                autoComplete="off"
                value={fullName}
                onChange={(event) => setFullName(event.target.value)}
                placeholder="Иванов Иван Иванович"
              />
            )}
          </Field>
          <Field label="Дата рождения" required>
            {(props) => (
              <Input
                {...props}
                type="date"
                max={localToday()}
                value={birthDate}
                onChange={(event) => setBirthDate(event.target.value)}
              />
            )}
          </Field>
          <Field label="Номер профсоюзного билета" description="Необязательное поле">
            {(props) => (
              <Input
                {...props}
                autoComplete="off"
                value={unionCard}
                onChange={(event) => setUnionCard(event.target.value)}
              />
            )}
          </Field>
          <Field label="Barcode СКС РФ" description="Необязательное поле">
            {(props) => (
              <Input
                {...props}
                autoComplete="off"
                value={sksBarcode}
                onChange={(event) => setSKSBarcode(event.target.value)}
              />
            )}
          </Field>
          <Field
            label="Направление"
            description="Лекции можно ограничить направлением. Без направления участник видит только общие лекции."
          >
            {(props) => (
              <Select value={directionId} onValueChange={setDirectionId}>
                <SelectTrigger id={props.id} aria-invalid={props['aria-invalid']}>
                  <SelectValue placeholder="Не указано" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Не указано</SelectItem>
                  {(directions.data ?? []).map((direction) => (
                    <SelectItem key={direction.id} value={direction.id}>
                      {direction.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </Field>

          {error && (
            <div
              role="alert"
              className="rounded-[10px] bg-danger/10 px-3.5 py-2.5 text-[13px] text-danger"
            >
              {error}
            </div>
          )}

          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" variant="ghost" disabled={isPending} onClick={close}>
              Отмена
            </Button>
            <Button type="submit" loading={isPending}>
              {editing ? 'Сохранить' : 'Добавить'}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
