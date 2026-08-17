import { useEffect, useState, type FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { ImagePlus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { deleteTaskImage, uploadTaskImage } from '@/entities/event-task/api'
import { useCreateTask, useUpdateTask } from '@/entities/event-task/queries'
import type { EventTask, EventTaskInput, TaskAssetType } from '@/entities/event-task/types'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'

function toLocal(value: string | null | undefined): string {
  if (!value) return ''
  const date = new Date(value)
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

function toISO(value: string): string | null {
  return value ? new Date(value).toISOString() : null
}

export function EventTaskDialog({
  contestId,
  task,
  open,
  onOpenChange,
}: {
  contestId: string
  task: EventTask | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const create = useCreateTask(contestId)
  const update = useUpdateTask(contestId, task?.id ?? '')
  const queryClient = useQueryClient()
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [icon, setIcon] = useState('')
  const [points, setPoints] = useState('100')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [sortOrder, setSortOrder] = useState('0')
  const [types, setTypes] = useState<TaskAssetType[]>(['IMAGE', 'LINK'])
  const [image, setImage] = useState<File | null>(null)
  const [imageBusy, setImageBusy] = useState(false)

  useEffect(() => {
    if (!open) return
    setTitle(task?.title ?? '')
    setDescription(task?.description ?? '')
    setIcon(task?.icon ?? '')
    setPoints(String(task?.points ?? 100))
    setStartsAt(toLocal(task?.starts_at))
    setEndsAt(toLocal(task?.ends_at))
    setSortOrder(String(task?.sort_order ?? 0))
    setTypes(task?.allowed_submission_types ?? ['IMAGE', 'LINK'])
    setImage(null)
  }, [open, task])

  function toggleType(type: TaskAssetType) {
    setTypes((current) =>
      current.includes(type) ? current.filter((value) => value !== type) : [...current, type],
    )
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    const numericPoints = Number(points)
    const numericSort = Number(sortOrder)
    if (!title.trim() || !description.trim() || numericPoints <= 0 || types.length === 0) {
      toast.error('Заполните описание, награду и хотя бы один тип подтверждения')
      return
    }
    if (startsAt && endsAt && new Date(startsAt) >= new Date(endsAt)) {
      toast.error('Дата окончания должна быть позже даты начала')
      return
    }
    const input: EventTaskInput = {
      title: title.trim(),
      description: description.trim(),
      icon: icon.trim() || null,
      points: numericPoints,
      starts_at: toISO(startsAt),
      ends_at: toISO(endsAt),
      sort_order: Number.isFinite(numericSort) ? numericSort : 0,
      allowed_submission_types: types,
    }
    try {
      const saved = task ? await update.mutateAsync(input) : await create.mutateAsync(input)
      if (image) await uploadTaskImage(contestId, saved.id, image)
      await queryClient.invalidateQueries({ queryKey: ['admin', 'event-tasks', contestId] })
      toast.success(task ? 'Задание обновлено' : 'Задание создано')
      onOpenChange(false)
    } catch {
      toast.error('Не удалось сохранить задание')
    }
  }

  async function removeImage() {
    if (!task?.image_url) return
    setImageBusy(true)
    try {
      await deleteTaskImage(contestId, task.id)
      await queryClient.invalidateQueries({ queryKey: ['admin', 'event-tasks', contestId] })
      toast.success('Обложка удалена')
      onOpenChange(false)
    } catch {
      toast.error('Не удалось удалить обложку')
    } finally {
      setImageBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-h-[90vh] max-w-2xl overflow-y-auto"
        title={task ? 'Редактировать задание' : 'Новое задание'}
        description="Укажите награду, период доступности и допустимые подтверждения."
      >
        <form className="flex flex-col gap-4" onSubmit={submit}>
          <div className="grid gap-4 sm:grid-cols-[100px_1fr]">
            <Field label="Иконка" helpText="Короткий emoji или символ до 64 знаков.">
              {(props) => (
                <Input
                  {...props}
                  value={icon}
                  onChange={(e) => setIcon(e.target.value)}
                  placeholder="🎯"
                />
              )}
            </Field>
            <Field label="Название" required>
              {(props) => (
                <Input
                  {...props}
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  maxLength={300}
                />
              )}
            </Field>
          </div>
          <Field label="Описание и условия" required>
            {(props) => (
              <Textarea
                {...props}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={6}
                maxLength={20_000}
              />
            )}
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Баллы" required>
              {(props) => (
                <Input
                  {...props}
                  type="number"
                  min={1}
                  max={1_000_000}
                  value={points}
                  onChange={(e) => setPoints(e.target.value)}
                />
              )}
            </Field>
            <Field label="Порядок">
              {(props) => (
                <Input
                  {...props}
                  type="number"
                  value={sortOrder}
                  onChange={(e) => setSortOrder(e.target.value)}
                />
              )}
            </Field>
            <Field label="Доступно с">
              {(props) => (
                <Input
                  {...props}
                  type="datetime-local"
                  value={startsAt}
                  onChange={(e) => setStartsAt(e.target.value)}
                />
              )}
            </Field>
            <Field label="Доступно до">
              {(props) => (
                <Input
                  {...props}
                  type="datetime-local"
                  value={endsAt}
                  onChange={(e) => setEndsAt(e.target.value)}
                />
              )}
            </Field>
          </div>
          <fieldset>
            <legend className="text-[14px] font-medium text-ink">Типы подтверждения</legend>
            <div className="mt-2 flex flex-wrap gap-4">
              {(['IMAGE', 'LINK'] as const).map((type) => (
                <label key={type} className="flex items-center gap-2 text-[14px] text-ink">
                  <input
                    type="checkbox"
                    checked={types.includes(type)}
                    onChange={() => toggleType(type)}
                  />
                  {type === 'IMAGE' ? 'Изображения' : 'Ссылки'}
                </label>
              ))}
            </div>
          </fieldset>
          <Field label="Обложка" description="JPG, PNG, WEBP или GIF, до 20 МБ.">
            {(props) => (
              <Input
                {...props}
                type="file"
                accept="image/jpeg,image/png,image/webp,image/gif"
                onChange={(e) => setImage(e.target.files?.[0] ?? null)}
              />
            )}
          </Field>
          {task?.image_url && (
            <div className="flex items-center gap-3 rounded-[12px] bg-surface-2 p-3">
              <img
                src={task.image_url}
                alt="Обложка задания"
                className="h-14 w-20 rounded-lg object-cover"
              />
              <Button
                type="button"
                size="sm"
                variant="ghost"
                loading={imageBusy}
                onClick={removeImage}
              >
                <Trash2 className="h-4 w-4 text-danger" /> Удалить обложку
              </Button>
            </div>
          )}
          <div className="mt-2 flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" loading={create.isPending || update.isPending || imageBusy}>
              <ImagePlus className="h-4 w-4" /> Сохранить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
