import { useEffect, useState } from 'react'
import { Plus, X } from 'lucide-react'
import { toast } from 'sonner'
import { useEventDirections } from '@/entities/event-participant/admin-queries'
import type { Lecture, LectureInput } from '@/entities/lecture/types'
import { useCreateLecture, useUpdateLecture } from '@/entities/lecture/queries'
import { isoToLocalInput, localInputToIso } from '@/shared/lib/format'
import { Button } from '@/shared/ui/button'
import { Checkbox } from '@/shared/ui/choice'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input, Textarea } from '@/shared/ui/input'

const MAX_PEOPLE_PER_ROLE = 20

function namesOrBlank(names: string[] | undefined): string[] {
  return names && names.length > 0 ? names : ['']
}

function cleanedNames(names: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const raw of names) {
    const name = raw.trim().replace(/\s+/g, ' ')
    if (!name) continue
    const key = name.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    out.push(name)
  }
  return out
}

export function LectureDialog({
  contestId,
  lecture,
  open,
  onOpenChange,
}: {
  contestId: string
  lecture: Lecture | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [points, setPoints] = useState('100')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [attendanceStartsAt, setAttendanceStartsAt] = useState('')
  const [attendanceEndsAt, setAttendanceEndsAt] = useState('')
  const [directionIds, setDirectionIds] = useState<string[]>([])
  const [speakers, setSpeakers] = useState<string[]>([''])
  const [moderators, setModerators] = useState<string[]>([''])
  const [location, setLocation] = useState('')
  const [error, setError] = useState<string>()
  const directions = useEventDirections(open ? contestId : undefined)
  const create = useCreateLecture(contestId)
  const update = useUpdateLecture(contestId, lecture?.id ?? '')

  useEffect(() => {
    if (!open) return
    setTitle(lecture?.title ?? '')
    setDescription(lecture?.description ?? '')
    setPoints(String(lecture?.points ?? 100))
    setStartsAt(isoToLocalInput(lecture?.starts_at ?? null))
    setEndsAt(isoToLocalInput(lecture?.ends_at ?? null))
    setAttendanceStartsAt(isoToLocalInput(lecture?.attendance_starts_at ?? null))
    setAttendanceEndsAt(isoToLocalInput(lecture?.attendance_ends_at ?? null))
    setDirectionIds(lecture?.direction_ids ?? [])
    setSpeakers(namesOrBlank(lecture?.speakers))
    setModerators(namesOrBlank(lecture?.moderators))
    setLocation(lecture?.location ?? '')
    setError(undefined)
  }, [lecture, open])

  function submit(event: React.FormEvent) {
    event.preventDefault()
    setError(undefined)
    const numericPoints = Number(points)
    const starts = localInputToIso(startsAt)
    const ends = localInputToIso(endsAt)
    const attendanceStarts = localInputToIso(attendanceStartsAt)
    const attendanceEnds = localInputToIso(attendanceEndsAt)
    if (!title.trim() || !Number.isInteger(numericPoints) || numericPoints <= 0) {
      setError('Укажите название и положительное целое количество баллов.')
      return
    }
    if (
      (starts && ends && new Date(starts) >= new Date(ends)) ||
      (attendanceStarts && attendanceEnds && new Date(attendanceStarts) >= new Date(attendanceEnds))
    ) {
      setError('Время окончания должно быть позже начала.')
      return
    }
    const input: LectureInput = {
      title: title.trim(),
      description: description.trim() || null,
      points: numericPoints,
      starts_at: starts,
      ends_at: ends,
      attendance_starts_at: attendanceStarts,
      attendance_ends_at: attendanceEnds,
      direction_ids: directionIds,
      speakers: cleanedNames(speakers),
      moderators: cleanedNames(moderators),
      location: location.trim() || null,
    }
    const mutation = lecture ? update : create
    mutation.mutate(input, {
      onSuccess: () => {
        toast.success(lecture ? 'Лекция обновлена' : 'Лекция создана')
        onOpenChange(false)
      },
      onError: () => setError('Не удалось сохранить лекцию. Проверьте поля и повторите.'),
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-h-[90vh] max-w-2xl overflow-y-auto"
        title={lecture ? 'Редактирование лекции' : 'Новая лекция'}
        description="Окно регистрации можно сделать уже основного времени лекции."
      >
        <form onSubmit={submit} className="flex flex-col gap-4">
          <Field label="Название" required error={error}>
            {(props) => (
              <Input
                {...props}
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                autoFocus
              />
            )}
          </Field>
          <Field label="Описание">
            {(props) => (
              <Textarea
                {...props}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            )}
          </Field>
          <Field label="Место проведения" description="Аудитория, зал или площадка.">
            {(props) => (
              <Input
                {...props}
                value={location}
                onChange={(e) => setLocation(e.target.value)}
                maxLength={300}
                placeholder="Например, зал А"
              />
            )}
          </Field>
          <Field label="Баллы за посещение" required>
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
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Начало лекции">
              {(props) => (
                <Input
                  {...props}
                  type="datetime-local"
                  value={startsAt}
                  onChange={(e) => setStartsAt(e.target.value)}
                />
              )}
            </Field>
            <Field label="Окончание лекции">
              {(props) => (
                <Input
                  {...props}
                  type="datetime-local"
                  value={endsAt}
                  onChange={(e) => setEndsAt(e.target.value)}
                />
              )}
            </Field>
            <Field label="Начало регистрации">
              {(props) => (
                <Input
                  {...props}
                  type="datetime-local"
                  value={attendanceStartsAt}
                  onChange={(e) => setAttendanceStartsAt(e.target.value)}
                />
              )}
            </Field>
            <Field label="Окончание регистрации">
              {(props) => (
                <Input
                  {...props}
                  type="datetime-local"
                  value={attendanceEndsAt}
                  onChange={(e) => setAttendanceEndsAt(e.target.value)}
                />
              )}
            </Field>
          </div>
          <PeopleNamesEditor
            label="Спикеры"
            description="Можно указать одного или нескольких. Поле можно оставить пустым."
            names={speakers}
            onChange={setSpeakers}
            placeholder="ФИО спикера"
          />
          <PeopleNamesEditor
            label="Модераторы"
            description="Можно указать одного или нескольких. Поле можно оставить пустым."
            names={moderators}
            onChange={setModerators}
            placeholder="ФИО модератора"
          />
          <div>
            <p className="text-[14px] font-medium text-ink">Направления</p>
            <p className="mt-1 text-[13px] text-muted">
              Если ничего не выбрать, лекцию увидят все участники. Иначе — только выбранные
              направления.
            </p>
            {(directions.data?.length ?? 0) === 0 ? (
              <p className="mt-2 rounded-[10px] bg-surface-2 p-3 text-[13px] text-muted">
                Каталог направлений пуст. Создайте их в блоке участников.
              </p>
            ) : (
              <ul className="mt-2 space-y-2">
                {(directions.data ?? []).map((direction) => {
                  const checked = directionIds.includes(direction.id)
                  return (
                    <li key={direction.id} className="flex items-center gap-2">
                      <Checkbox
                        checked={checked}
                        onCheckedChange={(value) => {
                          setDirectionIds((current) =>
                            value
                              ? [...current, direction.id]
                              : current.filter((id) => id !== direction.id),
                          )
                        }}
                        id={`lecture-direction-${direction.id}`}
                      />
                      <label
                        htmlFor={`lecture-direction-${direction.id}`}
                        className="text-[14px] text-ink"
                      >
                        {direction.name}
                      </label>
                    </li>
                  )
                })}
              </ul>
            )}
          </div>
          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              Отмена
            </Button>
            <Button type="submit" loading={create.isPending || update.isPending}>
              Сохранить
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function PeopleNamesEditor({
  label,
  description,
  names,
  onChange,
  placeholder,
}: {
  label: string
  description: string
  names: string[]
  onChange: (next: string[]) => void
  placeholder: string
}) {
  function update(index: number, value: string) {
    const next = names.slice()
    next[index] = value
    onChange(next)
  }
  function add() {
    if (names.length >= MAX_PEOPLE_PER_ROLE) return
    onChange([...names, ''])
  }
  function remove(index: number) {
    const next = names.filter((_, i) => i !== index)
    onChange(next.length ? next : [''])
  }

  return (
    <div>
      <p className="text-[14px] font-medium text-ink">{label}</p>
      <p className="mt-1 text-[13px] text-muted">{description}</p>
      <div className="mt-2 flex flex-col gap-2">
        {names.map((name, index) => (
          <div key={index} className="flex items-center gap-2">
            <Input
              value={name}
              onChange={(event) => update(index, event.target.value)}
              placeholder={placeholder}
              maxLength={120}
            />
            <button
              type="button"
              onClick={() => remove(index)}
              className="shrink-0 rounded-md p-2 text-muted hover:bg-danger/10 hover:text-danger"
              aria-label={`Удалить: ${label}`}
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        ))}
        {names.length < MAX_PEOPLE_PER_ROLE && (
          <Button type="button" variant="ghost" size="sm" onClick={add} className="self-start">
            <Plus className="h-4 w-4" /> Добавить
          </Button>
        )}
      </div>
    </div>
  )
}
