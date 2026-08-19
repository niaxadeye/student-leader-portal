import { useEffect, useRef, useState, type FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { ImagePlus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { deleteTaskImage, uploadTaskImage } from '@/entities/event-task/api'
import { useCreateTask, useUpdateTask } from '@/entities/event-task/queries'
import type { EventTask, EventTaskInput, TaskAssetType } from '@/entities/event-task/types'
import { isoToLocalInput, localInputToIso } from '@/shared/lib/format'
import { resizeImageToSquare } from '@/shared/lib/image'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/choice'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'

const SUBMISSION_TYPES: { value: TaskAssetType; label: string }[] = [
  { value: 'IMAGE', label: 'Изображения' },
  { value: 'LINK', label: 'Ссылки' },
]

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
  const fileRef = useRef<HTMLInputElement>(null)
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [points, setPoints] = useState('100')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [sortOrder, setSortOrder] = useState('0')
  const [types, setTypes] = useState<TaskAssetType[]>(['IMAGE', 'LINK'])
  const [iconFile, setIconFile] = useState<File | null>(null)
  const [iconPreview, setIconPreview] = useState<string | null>(null)
  const [iconRemoved, setIconRemoved] = useState(false)
  const [error, setError] = useState<string>()
  const [imageBusy, setImageBusy] = useState(false)

  useEffect(() => {
    if (!open) return
    setTitle(task?.title ?? '')
    setDescription(task?.description ?? '')
    setPoints(String(task?.points ?? 100))
    setStartsAt(isoToLocalInput(task?.starts_at ?? null))
    setEndsAt(isoToLocalInput(task?.ends_at ?? null))
    setSortOrder(String(task?.sort_order ?? 0))
    setTypes(task?.allowed_submission_types ?? ['IMAGE', 'LINK'])
    setIconFile(null)
    setIconRemoved(false)
    setIconPreview(task?.image_url ?? null)
    setError(undefined)
  }, [open, task])

  function toggleType(type: TaskAssetType, checked: boolean) {
    setTypes((current) => {
      if (checked) return current.includes(type) ? current : [...current, type]
      return current.filter((value) => value !== type)
    })
  }

  async function pickIcon(file: File | undefined) {
    if (!file) return
    setImageBusy(true)
    try {
      const icon = await resizeImageToSquare(file, 96)
      setIconFile(icon)
      setIconRemoved(false)
      setIconPreview((current) => {
        if (current && current.startsWith('blob:')) URL.revokeObjectURL(current)
        return URL.createObjectURL(icon)
      })
    } catch {
      toast.error('Не удалось обработать изображение. Выберите JPG, PNG, WEBP или GIF.')
    } finally {
      setImageBusy(false)
    }
  }

  function clearIcon() {
    setIconFile(null)
    setIconRemoved(true)
    setIconPreview((current) => {
      if (current && current.startsWith('blob:')) URL.revokeObjectURL(current)
      return null
    })
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError(undefined)
    const numericPoints = Number(points)
    const numericSort = Number(sortOrder)
    const starts = localInputToIso(startsAt)
    const ends = localInputToIso(endsAt)
    if (!title.trim() || !description.trim() || !Number.isInteger(numericPoints) || numericPoints <= 0) {
      setError('Укажите название, описание и положительное целое количество баллов.')
      return
    }
    if (types.length === 0) {
      setError('Выберите хотя бы один тип подтверждения.')
      return
    }
    if (starts && ends && new Date(starts) >= new Date(ends)) {
      setError('Время окончания должно быть позже начала.')
      return
    }
    const input: EventTaskInput = {
      title: title.trim(),
      description: description.trim(),
      icon: null,
      points: numericPoints,
      starts_at: starts,
      ends_at: ends,
      sort_order: Number.isFinite(numericSort) ? numericSort : 0,
      allowed_submission_types: types,
    }
    try {
      const saved = task ? await update.mutateAsync(input) : await create.mutateAsync(input)
      if (iconFile) await uploadTaskImage(contestId, saved.id, iconFile)
      else if (iconRemoved && task?.image_url) await deleteTaskImage(contestId, saved.id)
      await queryClient.invalidateQueries({ queryKey: ['admin', 'event-tasks', contestId] })
      toast.success(task ? 'Задание обновлено' : 'Задание создано')
      onOpenChange(false)
    } catch {
      setError('Не удалось сохранить задание. Проверьте поля и повторите.')
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-h-[90vh] max-w-2xl overflow-y-auto"
        title={task ? 'Редактирование задания' : 'Новое задание'}
        description="Награда, период доступности и то, чем участник подтвердит выполнение."
      >
        <form className="flex flex-col gap-4" onSubmit={submit}>
          <div className="flex items-start gap-4">
            <div>
              <p className="text-[14px] font-medium text-ink">Иконка</p>
              <p className="mt-1 max-w-[220px] text-[13px] text-muted">
                Квадрат 96×96. Любое фото обрежется по центру.
              </p>
              <div className="mt-2 flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => fileRef.current?.click()}
                  className="flex h-[96px] w-[96px] items-center justify-center overflow-hidden rounded-[12px] border border-dashed border-border bg-surface-2 transition-colors hover:border-brand hover:bg-brand-subtle"
                  aria-label="Загрузить иконку 96 на 96"
                >
                  {iconPreview ? (
                    <img src={iconPreview} alt="" className="h-full w-full object-cover" />
                  ) : (
                    <ImagePlus className="h-7 w-7 text-brand" />
                  )}
                </button>
                {iconPreview && (
                  <Button type="button" size="sm" variant="ghost" onClick={clearIcon}>
                    <Trash2 className="h-4 w-4 text-danger" />
                  </Button>
                )}
                <input
                  ref={fileRef}
                  type="file"
                  accept="image/jpeg,image/png,image/webp,image/gif"
                  className="sr-only"
                  onChange={(event) => {
                    void pickIcon(event.target.files?.[0])
                    event.target.value = ''
                  }}
                />
              </div>
            </div>
            <div className="min-w-0 flex-1">
              <Field label="Название" required error={error}>
                {(props) => (
                  <Input
                    {...props}
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    maxLength={300}
                    autoFocus
                  />
                )}
              </Field>
            </div>
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
            <Field label="Баллы за выполнение" required>
              {(props) => (
                <Input
                  {...props}
                  type="number"
                  min={1}
                  step={1}
                  value={points}
                  onChange={(e) => setPoints(e.target.value)}
                />
              )}
            </Field>
            <Field label="Порядок" description="Меньше число — выше в списке.">
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
          <div>
            <p className="text-[14px] font-medium text-ink">Типы подтверждения</p>
            <p className="mt-1 text-[13px] text-muted">
              Участник сможет приложить отмеченные виды доказательств.
            </p>
            <ul className="mt-2 space-y-2">
              {SUBMISSION_TYPES.map((item) => {
                const checked = types.includes(item.value)
                return (
                  <li key={item.value} className="flex items-center gap-2">
                    <Checkbox
                      checked={checked}
                      onCheckedChange={(value) => toggleType(item.value, value === true)}
                      id={`task-type-${item.value}`}
                    />
                    <label htmlFor={`task-type-${item.value}`} className="text-[14px] text-ink">
                      {item.label}
                    </label>
                  </li>
                )
              })}
            </ul>
          </div>
          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" loading={create.isPending || update.isPending || imageBusy}>
              Сохранить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
