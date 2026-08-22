import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, ListChecks, ChevronRight, Copy } from 'lucide-react'
import { useAdminChallenges, useDuplicateChallenge, useUpdateChallenge } from '@/entities/challenge/admin-queries'
import { Card, CardBody } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Switch } from '@/shared/ui/switch'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'
import { formatDateTime } from '@/shared/lib/format'
import { toast } from 'sonner'
import { intakeStatusLine, liveConductedMeta, schemeTypeLabel } from './challenge-status'
import { CreateChallengeDialog } from './create-challenge-dialog'
import type { AdminChallenge, ChallengeInput } from '@/entities/challenge/admin-types'

function metaInput(ch: AdminChallenge, patch: Partial<ChallengeInput> = {}): ChallengeInput {
  return {
    title: ch.title,
    short_description: ch.short_description,
    full_description: ch.full_description,
    instructions: ch.instructions,
    open_at: ch.open_at,
    deadline_at: ch.deadline_at,
    close_at: ch.close_at,
    held_at: ch.held_at,
    venue: ch.venue,
    accepts_submissions: ch.accepts_submissions,
    ...patch,
  }
}

/** Список испытаний конкурса. Клик открывает хаб с разделами приёма и проведения. */
export function ChallengesSection({ contestId, canEdit }: { contestId: string; canEdit: boolean }) {
  const { data, isLoading, isError, refetch } = useAdminChallenges(contestId)
  const duplicate = useDuplicateChallenge(contestId)
  const [createOpen, setCreateOpen] = useState(false)
  const [copyingId, setCopyingId] = useState<string>()
  const [togglingId, setTogglingId] = useState<string>()

  function onDuplicate(ch: AdminChallenge) {
    setCopyingId(ch.id)
    duplicate.mutate(ch.id, {
      onSuccess: (copy) => toast.success(`Создана копия: ${copy.title}`),
      onError: () => toast.error('Не удалось дублировать испытание'),
      onSettled: () => setCopyingId(undefined),
    })
  }

  return (
    <section>
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-[20px] font-semibold text-ink">Испытания</h2>
        {canEdit && (
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" /> Новое испытание
          </Button>
        )}
      </div>

      {isLoading && <Skeleton className="h-32 w-full" />}
      {isError && <ErrorState onRetry={() => refetch()} />}
      {data && data.length === 0 && (
        <EmptyState
          title="Пока нет испытаний"
          description="Создайте первое испытание и соберите форму в конструкторе."
        />
      )}

      {data && data.length > 0 && (
        <div className="flex flex-col gap-2">
          {data.map((ch) => (
            <ChallengeListCard
              key={ch.id}
              ch={ch}
              contestId={contestId}
              canEdit={canEdit}
              copying={duplicate.isPending && copyingId === ch.id}
              toggling={togglingId === ch.id}
              onDuplicate={() => onDuplicate(ch)}
              onToggleStart={() => setTogglingId(ch.id)}
              onToggleEnd={() => setTogglingId(undefined)}
            />
          ))}
        </div>
      )}

      <CreateChallengeDialog contestId={contestId} open={createOpen} onOpenChange={setCreateOpen} />
    </section>
  )
}

function ChallengeListCard({
  ch,
  contestId,
  canEdit,
  copying,
  toggling,
  onDuplicate,
  onToggleStart,
  onToggleEnd,
}: {
  ch: AdminChallenge
  contestId: string
  canEdit: boolean
  copying: boolean
  toggling: boolean
  onDuplicate: () => void
  onToggleStart: () => void
  onToggleEnd: () => void
}) {
  const update = useUpdateChallenge(ch.id, contestId)
  const conducted = liveConductedMeta(ch.live_state)

  function onToggle(next: boolean) {
    onToggleStart()
    update.mutate(metaInput(ch, { accepts_submissions: next }), {
      onSuccess: () =>
        toast.success(next ? 'Приём файлов и ТЗ включён' : 'Приём файлов и ТЗ выключен'),
      onError: () => toast.error('Не удалось изменить приём файлов'),
      onSettled: onToggleEnd,
    })
  }

  return (
    <Card className="transition hover:border-brand/40">
      <CardBody className="flex items-center gap-2 py-3.5 pr-2">
        <Link
          to={`/admin/challenges/${ch.id}`}
          className="flex min-w-0 flex-1 items-center gap-4"
        >
          <ListChecks className="h-5 w-5 shrink-0 text-brand" />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <p className="truncate text-[15px] font-medium text-ink">{ch.title}</p>
              <Badge tone={conducted.tone}>{conducted.label}</Badge>
            </div>
            <p className="mt-0.5 truncate text-[13px] text-muted">{schemeTypeLabel(ch.scheme_type)}</p>
            <p className="mt-0.5 truncate text-[13px] text-muted">
              {intakeStatusLine(ch)}
              {ch.held_at ? ` · ${formatDateTime(ch.held_at)}` : ''}
              {ch.venue ? ` · ${ch.venue}` : ''}
            </p>
          </div>
          <ChevronRight className="h-4 w-4 shrink-0 text-muted-2" />
        </Link>
        {canEdit && (
          <div className="flex shrink-0 flex-col items-center gap-1 px-1">
            <span className="text-[11px] leading-none text-muted">Файлы и ТЗ</span>
            <Switch
              checked={ch.accepts_submissions !== false}
              disabled={update.isPending && toggling}
              onCheckedChange={onToggle}
              label="Получаем файлы и ТЗ"
            />
          </div>
        )}
        {canEdit && (
          <button
            type="button"
            title="Дублировать"
            aria-label={`Дублировать «${ch.title}»`}
            disabled={copying}
            onClick={onDuplicate}
            className="shrink-0 rounded-md p-2 text-muted transition hover:bg-muted/10 hover:text-ink disabled:opacity-40"
          >
            <Copy className="h-4 w-4" />
          </button>
        )}
      </CardBody>
    </Card>
  )
}
