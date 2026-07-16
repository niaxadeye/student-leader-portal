import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import { toast } from 'sonner'
import { useCreateContest } from '@/entities/contest/queries'
import { ApiRequestError } from '@/shared/api/client'

export function CreateContestDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string>()
  const create = useCreateContest()
  const navigate = useNavigate()

  function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(undefined)
    if (!name.trim()) {
      setError('Укажите название конкурса.')
      return
    }
    create.mutate(
      { name: name.trim(), slug: slug.trim() || undefined, description: description.trim() || undefined },
      {
        onSuccess: (c) => {
          toast.success('Конкурс создан')
          onOpenChange(false)
          setName('')
          setSlug('')
          setDescription('')
          navigate(`/admin/contests/${c.id}`)
        },
        onError: (err) => {
          if (err instanceof ApiRequestError && err.code === 'SLUG_TAKEN') {
            setError('Такой слаг уже занят. Укажите другой.')
          } else {
            setError('Не удалось создать конкурс. Попробуйте ещё раз.')
          }
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title="Новый конкурс" description="Конкурс создаётся в статусе «Черновик».">
        <form onSubmit={submit} className="flex flex-col gap-4">
          <Field label="Название" required error={error}>
            {(p) => (
              <Input {...p} value={name} onChange={(e) => setName(e.target.value)} placeholder="Студенческий лидер 2026" autoFocus />
            )}
          </Field>
          <Field label="Слаг" helpText="Идентификатор в URL. Если пусто — сгенерируется из названия.">
            {(p) => (
              <Input {...p} value={slug} onChange={(e) => setSlug(e.target.value)} placeholder="student-leader-2026" />
            )}
          </Field>
          <Field label="Описание">
            {(p) => (
              <Textarea {...p} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="Краткое описание конкурса" />
            )}
          </Field>
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
