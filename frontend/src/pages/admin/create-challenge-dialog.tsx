import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import { useToast } from '@/shared/ui/toast'
import { useCreateChallenge } from '@/entities/challenge/admin-queries'
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
  const [error, setError] = useState<string>()
  const create = useCreateChallenge(contestId)
  const toast = useToast()
  const navigate = useNavigate()

  function submit(e: React.FormEvent) {
    e.preventDefault()
    setError(undefined)
    if (!title.trim()) {
      setError('Укажите название испытания.')
      return
    }
    create.mutate(
      { title: title.trim(), short_description: shortDescription.trim() || null },
      {
        onSuccess: (c) => {
          toast({ title: 'Испытание создано', tone: 'success' })
          onOpenChange(false)
          setTitle('')
          setShortDescription('')
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
      <DialogContent title="Новое испытание" description="Испытание создаётся в статусе «Черновик».">
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
