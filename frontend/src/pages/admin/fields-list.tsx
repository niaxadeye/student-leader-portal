import { ChevronUp, ChevronDown, Pencil, Trash2, GripVertical } from 'lucide-react'
import { Card, CardBody } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { useReorderFields, useDeleteField } from '@/entities/challenge/admin-queries'
import { toast } from 'sonner'
import type { AdminField } from '@/entities/challenge/admin-types'
import { fieldTypeLabels } from './challenge-status'

interface Props {
  challengeId: string
  fields: AdminField[]
  onEdit: (field: AdminField) => void
  /** VIEW-режим: скрывает перестановку/редактирование/удаление. */
  canEdit: boolean
}

/** Список полей с перестановкой (↑↓), редактированием и удалением. */
export function FieldsList({ challengeId, fields, onEdit, canEdit }: Props) {
  const reorder = useReorderFields(challengeId)
  const del = useDeleteField(challengeId)

  function move(index: number, dir: -1 | 1) {
    const target = index + dir
    if (target < 0 || target >= fields.length) return
    const ids = fields.map((f) => f.id)
    ;[ids[index], ids[target]] = [ids[target], ids[index]]
    reorder.mutate(ids, {
      onError: () => toast.error('Не удалось изменить порядок'),
    })
  }

  function remove(field: AdminField) {
    if (!window.confirm(`Удалить поле «${field.label}»?`)) return
    del.mutate(field.id, {
      onSuccess: () => toast.success('Поле удалено'),
      onError: () => toast.error('Не удалось удалить поле'),
    })
  }

  return (
    <div className="flex flex-col gap-2">
      {fields.map((f, i) => (
        <Card key={f.id}>
          <CardBody className="flex items-center gap-3 py-3">
            <GripVertical className="h-4 w-4 shrink-0 text-muted-2" />
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-[15px] font-medium text-ink">{f.label}</span>
                <Badge tone="neutral">{fieldTypeLabels[f.type] ?? f.type}</Badge>
                {f.required && <Badge tone="warning">обязательное</Badge>}
              </div>
              <p className="mt-0.5 font-mono text-[12px] text-muted">{f.key}</p>
            </div>
            {canEdit && (
              <div className="flex shrink-0 items-center gap-0.5">
                <IconBtn label="Выше" disabled={i === 0} onClick={() => move(i, -1)}>
                  <ChevronUp className="h-4 w-4" />
                </IconBtn>
                <IconBtn label="Ниже" disabled={i === fields.length - 1} onClick={() => move(i, 1)}>
                  <ChevronDown className="h-4 w-4" />
                </IconBtn>
                <IconBtn label="Редактировать" onClick={() => onEdit(f)}>
                  <Pencil className="h-4 w-4" />
                </IconBtn>
                <IconBtn label="Удалить" danger onClick={() => remove(f)}>
                  <Trash2 className="h-4 w-4" />
                </IconBtn>
              </div>
            )}
          </CardBody>
        </Card>
      ))}
    </div>
  )
}

function IconBtn({
  children,
  label,
  onClick,
  disabled,
  danger,
}: {
  children: React.ReactNode
  label: string
  onClick: () => void
  disabled?: boolean
  danger?: boolean
}) {
  return (
    <button
      type="button"
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      className={
        'rounded-md p-2 text-muted transition disabled:opacity-30 disabled:hover:bg-transparent ' +
        (danger ? 'hover:bg-danger/10 hover:text-danger' : 'hover:bg-muted/10 hover:text-ink')
      }
    >
      {children}
    </button>
  )
}
