import { useMemo, useRef, useState } from 'react'
import {
  Archive,
  Ban,
  ChevronLeft,
  ChevronRight,
  Coins,
  Download,
  FileSpreadsheet,
  Pencil,
  Plus,
  RotateCcw,
  Search,
  Tags,
  Upload,
  X,
} from 'lucide-react'
import { toast } from 'sonner'
import { exportAdminParticipants } from '@/entities/event-participant/admin-api'
import {
  useAdminEventParticipants,
  useEventDirections,
  useEventParticipantStatus,
  useImportEventParticipants,
} from '@/entities/event-participant/admin-queries'
import type {
  AdminParticipantFilters,
  ParticipantExportFormat,
  ParticipantImportResult,
  ParticipantStatusAction,
} from '@/entities/event-participant/admin-types'
import type { EventParticipant, EventParticipantStatus } from '@/entities/event-participant/types'
import { ApiRequestError } from '@/shared/api/client'
import { formatDateOnly } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card } from '@/shared/ui/card'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'
import { Input } from '@/shared/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/shared/ui/select'
import { EventDirectionsDialog } from './event-directions-dialog'
import { EventParticipantDialog } from './event-participant-dialog'
import { EventParticipantImportResultDialog } from './event-participant-import-result-dialog'
import { EventParticipantPointsDialog } from './event-participant-points-dialog'

const pageSize = 25
type StatusFilter = 'ALL' | EventParticipantStatus

const statusMeta: Record<
  EventParticipantStatus,
  { label: string; tone: 'success' | 'danger' | 'neutral' }
> = {
  ACTIVE: { label: 'Активен', tone: 'success' },
  BLOCKED: { label: 'Заблокирован', tone: 'danger' },
  ARCHIVED: { label: 'В архиве', tone: 'neutral' },
}

function importErrorMessage(error: unknown): string {
  if (error instanceof ApiRequestError) {
    if (error.status === 413) return 'Файл превышает допустимый размер 16 МиБ.'
    if (error.code === 'VALIDATION_ERROR') {
      return 'Не удалось прочитать файл. Проверьте формат и обязательные колонки.'
    }
  }
  return 'Не удалось импортировать участников.'
}

export function EventParticipantsSection({ contestId }: { contestId: string }) {
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('ALL')
  const [directionFilter, setDirectionFilter] = useState('ALL')
  const [offset, setOffset] = useState(0)
  const [formOpen, setFormOpen] = useState(false)
  const [directionsOpen, setDirectionsOpen] = useState(false)
  const [editing, setEditing] = useState<EventParticipant | null>(null)
  const [pointsParticipant, setPointsParticipant] = useState<EventParticipant | null>(null)
  const [importResult, setImportResult] = useState<ParticipantImportResult | null>(null)
  const [exporting, setExporting] = useState<ParticipantExportFormat | null>(null)
  const fileInput = useRef<HTMLInputElement>(null)

  const filters = useMemo<AdminParticipantFilters>(
    () => ({
      search,
      status: statusFilter === 'ALL' ? '' : statusFilter,
      directionId: directionFilter === 'ALL' ? '' : directionFilter,
      limit: pageSize,
      offset,
    }),
    [directionFilter, offset, search, statusFilter],
  )
  const participants = useAdminEventParticipants(contestId, filters)
  const directions = useEventDirections(contestId)
  const statusMutation = useEventParticipantStatus(contestId)
  const importer = useImportEventParticipants(contestId)
  const data = participants.data

  function submitSearch(event: React.FormEvent) {
    event.preventDefault()
    setSearch(searchInput.trim())
    setOffset(0)
  }

  function clearSearch() {
    setSearchInput('')
    setSearch('')
    setOffset(0)
  }

  function changeDirectionFilter(value: string) {
    setDirectionFilter(value)
    setOffset(0)
  }

  function changeStatusFilter(value: StatusFilter) {
    setStatusFilter(value)
    setOffset(0)
  }

  function openCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function openEdit(participant: EventParticipant) {
    setEditing(participant)
    setFormOpen(true)
  }

  function changeStatus(participant: EventParticipant, action: ParticipantStatusAction) {
    if (action === 'block') {
      if (
        !window.confirm(
          `Заблокировать участника «${participant.full_name}» и завершить его активные сессии?`,
        )
      ) {
        return
      }
    }
    if (action === 'archive') {
      if (
        !window.confirm(
          `Архивировать участника «${participant.full_name}»? Вернуть его из архива через интерфейс нельзя.`,
        )
      ) {
        return
      }
    }
    statusMutation.mutate(
      { participantId: participant.id, action },
      {
        onSuccess: () => {
          const message =
            action === 'block'
              ? 'Участник заблокирован'
              : action === 'unblock'
                ? 'Участник разблокирован'
                : 'Участник архивирован'
          toast.success(message)

          const nextStatus =
            action === 'unblock' ? 'ACTIVE' : action === 'block' ? 'BLOCKED' : 'ARCHIVED'
          if (
            statusFilter !== 'ALL' &&
            statusFilter !== nextStatus &&
            data?.participants.length === 1 &&
            offset > 0
          ) {
            setOffset(Math.max(0, offset - pageSize))
          }
        },
        onError: () => toast.error('Не удалось изменить статус участника'),
      },
    )
  }

  function importFile(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    if (file.size > 16 * 1024 * 1024) {
      toast.error('Файл превышает допустимый размер 16 МиБ.')
      return
    }
    const extension = file.name.toLowerCase().split('.').pop()
    if (extension !== 'csv' && extension !== 'xlsx') {
      toast.error('Поддерживаются только файлы CSV и XLSX.')
      return
    }
    importer.mutate(file, {
      onSuccess: (result) => {
        setImportResult(result)
        if (result.errors || result.duplicates) {
          toast.info('Импорт завершён — проверьте построчный отчёт.')
        } else {
          toast.success('Импорт участников завершён.')
        }
      },
      onError: (error) => toast.error(importErrorMessage(error)),
    })
  }

  async function exportFile(format: ParticipantExportFormat) {
    setExporting(format)
    try {
      const blob = await exportAdminParticipants(contestId, format)
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = `event-participants.${format}`
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(url)
    } catch {
      toast.error('Не удалось экспортировать участников.')
    } finally {
      setExporting(null)
    }
  }

  const total = data?.total ?? 0
  const from = total === 0 ? 0 : offset + 1
  const to = Math.min(offset + (data?.participants.length ?? 0), total)

  return (
    <section>
      <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-[20px] font-semibold text-ink">Участники мероприятия</h2>
          <p className="mt-1 text-[13px] text-muted">
            Отдельные профили EventParticipant для входа в платформу мероприятия.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" variant="secondary" onClick={() => setDirectionsOpen(true)}>
            <Tags className="h-4 w-4" /> Направления
          </Button>
          <Button size="sm" onClick={openCreate}>
            <Plus className="h-4 w-4" /> Добавить участника
          </Button>
        </div>
      </div>

      <Card className="overflow-hidden">
        <div className="flex flex-col gap-3 border-b border-border p-4 lg:flex-row lg:items-center lg:justify-between">
          <form onSubmit={submitSearch} className="flex min-w-0 flex-1 gap-2">
            <div className="relative min-w-0 flex-1 lg:max-w-md">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-2" />
              <Input
                value={searchInput}
                onChange={(event) => setSearchInput(event.target.value)}
                placeholder="ФИО, профбилет или barcode"
                aria-label="Поиск участников"
                className="pl-9 pr-9"
              />
              {(searchInput || search) && (
                <button
                  type="button"
                  aria-label="Очистить поиск"
                  onClick={clearSearch}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded p-1 text-muted-2 hover:text-ink"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
            <Button type="submit" size="sm" variant="secondary" className="h-11">
              Найти
            </Button>
          </form>

          <div className="flex flex-wrap gap-2">
            <div className="w-52">
              <Select value={directionFilter} onValueChange={changeDirectionFilter}>
                <SelectTrigger className="h-9 text-[14px]" aria-label="Фильтр по направлению">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ALL">Все направления</SelectItem>
                  {(directions.data ?? []).map((direction) => (
                    <SelectItem key={direction.id} value={direction.id}>
                      {direction.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="w-44">
              <Select
                value={statusFilter}
                onValueChange={(value) => changeStatusFilter(value as StatusFilter)}
              >
                <SelectTrigger className="h-9 text-[14px]" aria-label="Фильтр по статусу">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ALL">Все статусы</SelectItem>
                  <SelectItem value="ACTIVE">Активные</SelectItem>
                  <SelectItem value="BLOCKED">Заблокированные</SelectItem>
                  <SelectItem value="ARCHIVED">Архивные</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <input
              ref={fileInput}
              type="file"
              accept=".csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
              hidden
              onChange={importFile}
            />
            <Button
              size="sm"
              variant="outline"
              loading={importer.isPending}
              onClick={() => fileInput.current?.click()}
            >
              <Upload className="h-4 w-4" /> Импорт
            </Button>
            <Button
              size="sm"
              variant="outline"
              loading={exporting === 'csv'}
              disabled={exporting !== null}
              onClick={() => exportFile('csv')}
              title="CSV также можно использовать как шаблон импорта"
            >
              <Download className="h-4 w-4" /> CSV
            </Button>
            <Button
              size="sm"
              variant="outline"
              loading={exporting === 'xlsx'}
              disabled={exporting !== null}
              onClick={() => exportFile('xlsx')}
            >
              <FileSpreadsheet className="h-4 w-4" /> XLSX
            </Button>
          </div>
        </div>

        {participants.isLoading ? (
          <div className="p-4">
            <Skeleton className="h-48 w-full" />
          </div>
        ) : participants.isError ? (
          <div className="p-4">
            <ErrorState onRetry={() => participants.refetch()} />
          </div>
        ) : !data?.participants.length ? (
          <div className="p-4">
            <EmptyState
              icon={search || statusFilter !== 'ALL' || directionFilter !== 'ALL' ? Search : Plus}
              title={
                search || statusFilter !== 'ALL' || directionFilter !== 'ALL'
                  ? 'Ничего не найдено'
                  : 'Участников пока нет'
              }
              description={
                search || statusFilter !== 'ALL' || directionFilter !== 'ALL'
                  ? 'Измените поисковый запрос или фильтры.'
                  : 'Добавьте участника вручную или загрузите список CSV/XLSX с колонкой direction/направление.'
              }
              action={
                search || statusFilter !== 'ALL' || directionFilter !== 'ALL' ? (
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() => {
                      clearSearch()
                      changeStatusFilter('ALL')
                      changeDirectionFilter('ALL')
                    }}
                  >
                    Сбросить фильтры
                  </Button>
                ) : (
                  <Button size="sm" onClick={openCreate}>
                    Добавить участника
                  </Button>
                )
              }
            />
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[900px] text-left text-[14px]">
              <thead className="text-[11px] uppercase tracking-wide text-muted-2">
                <tr className="border-b border-border">
                  <th className="px-4 py-2 font-medium">Участник</th>
                  <th className="px-4 py-2 font-medium">Направление</th>
                  <th className="px-4 py-2 font-medium">Идентификаторы</th>
                  <th className="px-4 py-2 font-medium">Статус</th>
                  <th className="px-4 py-2 text-right font-medium">Действия</th>
                </tr>
              </thead>
              <tbody>
                {data.participants.map((participant) => (
                  <ParticipantRow
                    key={participant.id}
                    participant={participant}
                    disabled={statusMutation.isPending}
                    onEdit={() => openEdit(participant)}
                    onPoints={() => setPointsParticipant(participant)}
                    onStatus={(action) => changeStatus(participant, action)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}

        <div className="flex items-center justify-between gap-3 border-t border-border px-4 py-3">
          <span className="text-[12px] text-muted">
            {total ? `${from}–${to} из ${total}` : '0 участников'}
          </span>
          <div className="flex gap-1">
            <IconButton
              title="Предыдущая страница"
              disabled={offset === 0 || participants.isFetching}
              onClick={() => setOffset(Math.max(0, offset - pageSize))}
            >
              <ChevronLeft className="h-4 w-4" />
            </IconButton>
            <IconButton
              title="Следующая страница"
              disabled={offset + pageSize >= total || participants.isFetching}
              onClick={() => setOffset(offset + pageSize)}
            >
              <ChevronRight className="h-4 w-4" />
            </IconButton>
          </div>
        </div>
      </Card>

      <EventParticipantDialog
        contestId={contestId}
        participant={editing}
        open={formOpen}
        onOpenChange={setFormOpen}
      />
      <EventDirectionsDialog
        contestId={contestId}
        open={directionsOpen}
        onOpenChange={setDirectionsOpen}
      />
      <EventParticipantImportResultDialog
        result={importResult}
        onOpenChange={(open) => {
          if (!open) setImportResult(null)
        }}
      />
      <EventParticipantPointsDialog
        contestId={contestId}
        participant={pointsParticipant}
        onOpenChange={(open) => {
          if (!open) setPointsParticipant(null)
        }}
      />
    </section>
  )
}

function ParticipantRow({
  participant,
  disabled,
  onEdit,
  onPoints,
  onStatus,
}: {
  participant: EventParticipant
  disabled: boolean
  onEdit: () => void
  onPoints: () => void
  onStatus: (action: ParticipantStatusAction) => void
}) {
  const status = statusMeta[participant.status]
  return (
    <tr className="border-b border-border last:border-0 hover:bg-muted/5">
      <td className="px-4 py-3">
        <p className="font-medium text-ink">{participant.full_name}</p>
        <p className="mt-0.5 text-[12px] text-muted-2">
          Дата рождения: {formatDateOnly(participant.birth_date)}
        </p>
      </td>
      <td className="px-4 py-3">
        {participant.direction_name ? (
          <Badge>{participant.direction_name}</Badge>
        ) : (
          <span className="text-[12px] text-muted-2">Не указано</span>
        )}
      </td>
      <td className="px-4 py-3 text-[12px]">
        <p className="text-muted">
          Профбилет:{' '}
          <span className="font-medium text-ink">{participant.union_card_number || '—'}</span>
        </p>
        <p className="mt-1 text-muted">
          СКС: <span className="font-medium text-ink">{participant.sks_barcode || '—'}</span>
        </p>
      </td>
      <td className="px-4 py-3">
        <Badge tone={status.tone}>{status.label}</Badge>
      </td>
      <td className="px-4 py-3">
        <div className="flex justify-end gap-1">
          <IconButton title="Баллы и корректировка" disabled={disabled} onClick={onPoints}>
            <Coins className="h-4 w-4" />
          </IconButton>
          {participant.status !== 'ARCHIVED' && (
            <IconButton title="Редактировать" disabled={disabled} onClick={onEdit}>
              <Pencil className="h-4 w-4" />
            </IconButton>
          )}
          {participant.status === 'ACTIVE' && (
            <IconButton
              title="Заблокировать"
              disabled={disabled}
              danger
              onClick={() => onStatus('block')}
            >
              <Ban className="h-4 w-4" />
            </IconButton>
          )}
          {participant.status === 'BLOCKED' && (
            <IconButton
              title="Разблокировать"
              disabled={disabled}
              onClick={() => onStatus('unblock')}
            >
              <RotateCcw className="h-4 w-4" />
            </IconButton>
          )}
          {participant.status !== 'ARCHIVED' && (
            <IconButton
              title="Архивировать"
              disabled={disabled}
              danger
              onClick={() => onStatus('archive')}
            >
              <Archive className="h-4 w-4" />
            </IconButton>
          )}
        </div>
      </td>
    </tr>
  )
}

function IconButton({
  title,
  disabled,
  danger,
  onClick,
  children,
}: {
  title: string
  disabled?: boolean
  danger?: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      title={title}
      aria-label={title}
      disabled={disabled}
      onClick={onClick}
      className={`rounded-btn p-2 text-muted-2 transition-colors hover:bg-muted/10 disabled:opacity-40 ${danger ? 'hover:text-danger' : 'hover:text-brand'}`}
    >
      {children}
    </button>
  )
}
