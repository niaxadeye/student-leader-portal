import { useState } from 'react'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import {
  useCreateEventDirection,
  useDeleteEventDirection,
  useEventDirections,
  useUpdateEventDirection,
} from '@/entities/event-participant/admin-queries'
import type { EventDirection } from '@/entities/event-participant/types'
import { ApiRequestError } from '@/shared/api/client'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Input } from '@/shared/ui/input'
import { ErrorState, Skeleton } from '@/shared/ui/states'

function directionError(error: unknown): string {
  if (error instanceof ApiRequestError) {
    if (error.code === 'DIRECTION_NAME_TAKEN') return 'Направление с таким названием уже есть.'
    if (error.code === 'DIRECTION_IN_USE') {
      return 'Нельзя удалить: направление назначено участникам или лекциям.'
    }
    if (error.code === 'FORBIDDEN') return 'Недостаточно прав, чтобы менять каталог направлений.'
    if (error.code === 'VALIDATION_ERROR') return 'Укажите название до 80 символов.'
  }
  return 'Не удалось сохранить направление.'
}

export function EventDirectionsDialog({
  contestId,
  open,
  onOpenChange,
  canEdit = true,
}: {
  contestId: string
  open: boolean
  onOpenChange: (open: boolean) => void
  canEdit?: boolean
}) {
  const directions = useEventDirections(open ? contestId : undefined)
  const create = useCreateEventDirection(contestId)
  const update = useUpdateEventDirection(contestId)
  const remove = useDeleteEventDirection(contestId)
  const [name, setName] = useState('')
  const [editing, setEditing] = useState<EventDirection | null>(null)
  const [editName, setEditName] = useState('')

  const pending = create.isPending || update.isPending || remove.isPending

  function submitCreate(event: React.FormEvent) {
    event.preventDefault()
    const value = name.trim()
    if (!value) return
    create.mutate(value, {
      onSuccess: () => {
        setName('')
        toast.success('Направление добавлено')
      },
      onError: (error) => toast.error(directionError(error)),
    })
  }

  function submitRename(event: React.FormEvent) {
    event.preventDefault()
    if (!editing) return
    const value = editName.trim()
    if (!value) return
    update.mutate(
      { directionId: editing.id, name: value },
      {
        onSuccess: () => {
          setEditing(null)
          toast.success('Направление переименовано')
        },
        onError: (error) => toast.error(directionError(error)),
      },
    )
  }

  function deleteOne(direction: EventDirection) {
    if (!window.confirm(`Удалить направление «${direction.name}»?`)) return
    remove.mutate(direction.id, {
      onSuccess: () => toast.success('Направление удалено'),
      onError: (error) => toast.error(directionError(error)),
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-h-[90vh] max-w-lg overflow-y-auto"
        title="Направления"
        description="Участник относится к одному направлению. Лекцию можно назначить нескольким — или оставить для всех."
      >
        {canEdit && (
          <form onSubmit={submitCreate} className="flex items-end gap-2">
            <label className="min-w-0 flex-1 text-[14px] font-medium text-ink">
              Новое направление
              <Input
                className="mt-1.5"
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="IT, Гуманитарное, Медиа…"
                autoComplete="off"
              />
            </label>
            <Button type="submit" loading={create.isPending} disabled={!name.trim()}>
              <Plus className="h-4 w-4" /> Добавить
            </Button>
          </form>
        )}

        <div className="mt-5">
          {directions.isLoading && <Skeleton className="h-24 w-full" />}
          {directions.isError && <ErrorState onRetry={() => directions.refetch()} />}
          {directions.data?.length === 0 && (
            <p className="rounded-[12px] bg-surface-2 p-4 text-[13px] text-muted">
              {canEdit
                ? 'Каталог пуст. Пока лекции видят все участники.'
                : 'Каталог пуст. Добавить направления может организатор, который управляет участниками.'}
            </p>
          )}
          {!!directions.data?.length && (
            <ul className="divide-y divide-border rounded-[12px] border border-border">
              {directions.data.map((direction) => (
                <li key={direction.id} className="flex items-center gap-2 px-3 py-2.5">
                  {editing?.id === direction.id ? (
                    <form onSubmit={submitRename} className="flex min-w-0 flex-1 gap-2">
                      <Input
                        value={editName}
                        onChange={(event) => setEditName(event.target.value)}
                        autoFocus
                        autoComplete="off"
                      />
                      <Button type="submit" size="sm" loading={update.isPending}>
                        ОК
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={() => setEditing(null)}
                      >
                        Отмена
                      </Button>
                    </form>
                  ) : (
                    <>
                      <span className="min-w-0 flex-1 truncate font-medium text-ink">
                        {direction.name}
                      </span>
                      {canEdit ? (
                        <>
                          <Button
                            size="sm"
                            variant="ghost"
                            disabled={pending}
                            onClick={() => {
                              setEditing(direction)
                              setEditName(direction.name)
                            }}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            disabled={pending}
                            onClick={() => deleteOne(direction)}
                          >
                            <Trash2 className="h-4 w-4 text-danger" />
                          </Button>
                        </>
                      ) : null}
                    </>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
