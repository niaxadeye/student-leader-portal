import { useState } from 'react'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import { useToast } from '@/shared/ui/toast'
import { useUpdateContest } from '@/entities/contest/queries'
import { isoToLocalInput, localInputToIso } from '@/shared/lib/format'
import type { AdminContest } from '@/entities/contest/types'

export function EditContestDialog({
  contest,
  open,
  onOpenChange,
}: {
  contest: AdminContest
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const [name, setName] = useState(contest.name)
  const [description, setDescription] = useState(contest.description ?? '')
  const [startAt, setStartAt] = useState(isoToLocalInput(contest.start_at))
  const [endAt, setEndAt] = useState(isoToLocalInput(contest.end_at))
  const [timezone, setTimezone] = useState(contest.timezone)
  const [error, setError] = useState<string>()
  const update = useUpdateContest(contest.id)
  const toast = useToast()

  function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(undefined)
    if (!name.trim()) {
      setError('Укажите название конкурса.')
      return
    }
    const start = localInputToIso(startAt)
    const end = localInputToIso(endAt)
    if (start && end && new Date(end) < new Date(start)) {
      setError('Дата окончания раньше начала.')
      return
    }
    update.mutate(
      {
        name: name.trim(),
        description: description.trim() || undefined,
        start_at: start,
        end_at: end,
        timezone: timezone.trim() || undefined,
      },
      {
        onSuccess: () => {
          toast({ title: 'Конкурс обновлён', tone: 'success' })
          onOpenChange(false)
        },
        onError: () => setError('Не удалось сохранить изменения.'),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title="Редактирование конкурса" description="Слаг и статус здесь не меняются.">
        <form onSubmit={submit} className="flex flex-col gap-4">
          <Field label="Название" required error={error}>
            {(p) => <Input {...p} value={name} onChange={(e) => setName(e.target.value)} autoFocus />}
          </Field>
          <Field label="Описание">
            {(p) => (
              <Textarea {...p} value={description} onChange={(e) => setDescription(e.target.value)} />
            )}
          </Field>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Начало">
              {(p) => (
                <Input {...p} type="datetime-local" value={startAt} onChange={(e) => setStartAt(e.target.value)} />
              )}
            </Field>
            <Field label="Окончание">
              {(p) => (
                <Input {...p} type="datetime-local" value={endAt} onChange={(e) => setEndAt(e.target.value)} />
              )}
            </Field>
          </div>
          <Field label="Часовой пояс" helpText="IANA-имя, например Europe/Moscow.">
            {(p) => <Input {...p} value={timezone} onChange={(e) => setTimezone(e.target.value)} />}
          </Field>
          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" loading={update.isPending}>
              Сохранить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}