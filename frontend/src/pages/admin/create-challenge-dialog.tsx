import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import { Switch } from '@/shared/ui/switch'
import { toast } from 'sonner'
import { useCreateChallenge } from '@/entities/challenge/admin-queries'
import { localInputToIso } from '@/shared/lib/format'
import { ApiRequestError } from '@/shared/api/client'

export function CreateChallengeDialog({
  contestId,
  open,
  onOpenChange,
}: {
  contestId: string
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const [title, setTitle] = useState('')
  const [shortDescription, setShortDescription] = useState('')
  const [deadlineAt, setDeadlineAt] = useState('')
  const [heldAt, setHeldAt] = useState('')
  const [venue, setVenue] = useState('')
  const [acceptsSubmissions, setAcceptsSubmissions] = useState(true)
  const [error, setError] = useState<string>()
  const create = useCreateChallenge(contestId)
  const navigate = useNavigate()

  function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(undefined)
    if (!title.trim()) {
      setError('Укажите название испытания.')
      return
    }
    create.mutate(
      {
        title: title.trim(),
        short_description: shortDescription.trim() || null,
        deadline_at: localInputToIso(deadlineAt),
        held_at: localInputToIso(heldAt),
        venue: venue.trim() || null,
        accepts_submissions: acceptsSubmissions,
      },
      {
        onSuccess: (c) => {
          toast.success('Испытание создано')
          onOpenChange(false)
          setTitle('')
          setShortDescription('')
          setDeadlineAt('')
          setHeldAt('')
          setVenue('')
          setAcceptsSubmissions(true)
          navigate(`/admin/challenges/${c.id}`)
        },
        onError: (err) => {
          if (err instanceof ApiRequestError && err.code === 'SLUG_TAKEN') {
            setError('Испытание с таким слагом уже есть в конкурсе.')
          } else {
            setError('Не удалось создать испытание. Попробуйте ещё раз.')
          }
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        title="Новое испытание"
        description="Приём файлов и ТЗ создаётся как черновик. Опубликуйте его в разделе «Приём файлов и ТЗ»."
      >
        <form onSubmit={submit} className="flex flex-col gap-4">
          <Field label="Название" required error={error}>
            {(p) => (
              <Input
                {...p}
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="Презентация проекта"
                autoFocus
              />
            )}
          </Field>
          <Field label="Краткое описание">
            {(p) => (
              <Textarea
                {...p}
                value={shortDescription}
                onChange={(e) => setShortDescription(e.target.value)}
                placeholder="Одно-два предложения для карточки испытания"
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
          <Field label="Дедлайн сдачи" helpText="Нужен, если принимаете файлы и ТЗ. Можно задать позже.">
            {(p) => (
              <Input
                {...p}
                type="datetime-local"
                value={deadlineAt}
                onChange={(e) => setDeadlineAt(e.target.value)}
              />
            )}
          </Field>
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
            <Button type="submit" loading={create.isPending}>
              Создать
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
