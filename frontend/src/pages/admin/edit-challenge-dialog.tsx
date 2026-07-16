import { useState } from 'react'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
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
  const [error, setError] = useState<string>()
  const update = useUpdateChallenge(challenge.id, challenge.contest_id)

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
        open_at: open,
        deadline_at: deadline,
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
