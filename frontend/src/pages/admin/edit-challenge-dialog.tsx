import { useEffect, useState } from 'react'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import { Switch } from '@/shared/ui/switch'
import { toast } from 'sonner'
import { useUpdateChallenge } from '@/entities/challenge/admin-queries'
import { isoToLocalInput, localInputToIso } from '@/shared/lib/format'
import type { AdminChallenge } from '@/entities/challenge/admin-types'

export function EditChallengeDialog({
  challenge,
  open,
  onOpenChange,
}: {
  challenge: AdminChallenge
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const [title, setTitle] = useState(challenge.title)
  const [shortDescription, setShortDescription] = useState(challenge.short_description ?? '')
  const [openAt, setOpenAt] = useState(isoToLocalInput(challenge.open_at))
  const [deadlineAt, setDeadlineAt] = useState(isoToLocalInput(challenge.deadline_at))
  const [heldAt, setHeldAt] = useState(isoToLocalInput(challenge.held_at))
  const [venue, setVenue] = useState(challenge.venue ?? '')
  const [acceptsSubmissions, setAcceptsSubmissions] = useState(challenge.accepts_submissions)
  const [error, setError] = useState<string>()
  const update = useUpdateChallenge(challenge.id, challenge.contest_id)

  useEffect(() => {
    if (!open) return
    setTitle(challenge.title)
    setShortDescription(challenge.short_description ?? '')
    setOpenAt(isoToLocalInput(challenge.open_at))
    setDeadlineAt(isoToLocalInput(challenge.deadline_at))
    setHeldAt(isoToLocalInput(challenge.held_at))
    setVenue(challenge.venue ?? '')
    setAcceptsSubmissions(challenge.accepts_submissions)
    setError(undefined)
  }, [open, challenge])

  function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(undefined)
    if (!title.trim()) {
      setError('Укажите название испытания.')
      return
    }
    const open = localInputToIso(openAt)
    const deadline = localInputToIso(deadlineAt)
    if (open && deadline && new Date(deadline) < new Date(open)) {
      setError('Дедлайн раньше открытия.')
      return
    }
    update.mutate(
      {
        title: title.trim(),
        short_description: shortDescription.trim() || null,
        full_description: challenge.full_description,
        instructions: challenge.instructions,
        open_at: open,
        deadline_at: deadline,
        close_at: challenge.close_at,
        held_at: localInputToIso(heldAt),
        venue: venue.trim() || null,
        accepts_submissions: acceptsSubmissions,
      },
      {
        onSuccess: () => {
          toast.success('Испытание обновлено')
          onOpenChange(false)
        },
        onError: () => setError('Не удалось сохранить изменения.'),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title="Редактирование испытания" description="Слаг и статус здесь не меняются.">
        <form onSubmit={submit} className="flex flex-col gap-4">
          <Field label="Название" required error={error}>
            {(p) => <Input {...p} value={title} onChange={(e) => setTitle(e.target.value)} autoFocus />}
          </Field>
          <Field label="Краткое описание">
            {(p) => (
              <Textarea
                {...p}
                value={shortDescription}
                onChange={(e) => setShortDescription(e.target.value)}
              />
            )}
          </Field>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Дата и время проведения">
              {(p) => (
                <Input {...p} type="datetime-local" value={heldAt} onChange={(e) => setHeldAt(e.target.value)} />
              )}
            </Field>
            <Field label="Место проведения">
              {(p) => (
                <Input
                  {...p}
                  value={venue}
                  onChange={(e) => setVenue(e.target.value)}
                  placeholder="Аудитория, сцена…"
                />
              )}
            </Field>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Открытие приёма">
              {(p) => (
                <Input {...p} type="datetime-local" value={openAt} onChange={(e) => setOpenAt(e.target.value)} />
              )}
            </Field>
            <Field label="Дедлайн сдачи">
              {(p) => (
                <Input {...p} type="datetime-local" value={deadlineAt} onChange={(e) => setDeadlineAt(e.target.value)} />
              )}
            </Field>
          </div>
          <div className="flex items-center justify-between gap-3 rounded-[12px] border border-border px-3.5 py-3">
            <div>
              <p className="text-[14px] font-medium text-ink">Получаем файлы и ТЗ</p>
              <p className="mt-0.5 text-[13px] text-muted">Если выключено, участники не видят форму сдачи.</p>
            </div>
            <Switch
              checked={acceptsSubmissions}
              onCheckedChange={setAcceptsSubmissions}
              label="Получаем файлы и ТЗ"
            />
          </div>
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
