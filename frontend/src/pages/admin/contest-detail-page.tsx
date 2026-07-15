import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Users, Calendar, Rocket, Flag, Archive, Pencil, PencilRuler } from 'lucide-react'
import { useAdminContest, useTransitionContest } from '@/entities/contest/queries'
import { Card, CardBody } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'
import { useToast } from '@/shared/ui/toast'
import { formatDate } from '@/shared/lib/format'
import { contestStatusMeta } from './contest-status'
import { ContestantsTable } from './contestants-table'
import { ChallengesSection } from './challenges-section'
import { EditContestDialog } from './edit-contest-dialog'
import {
  canEditContest,
  canManageParticipants,
  type ContestStatus,
} from '@/entities/contest/types'

/** Доступные переходы по статусу (зеркалит матрицу бэкенда). */
const actionsByStatus: Record<ContestStatus, Array<'publish' | 'finish' | 'archive'>> = {
  DRAFT: ['publish'],
  ACTIVE: ['finish'],
  FINISHED: ['archive'],
  ARCHIVED: [],
}

const actionMeta = {
  publish: { label: 'Опубликовать', icon: Rocket },
  finish: { label: 'Завершить', icon: Flag },
  archive: { label: 'В архив', icon: Archive },
}

export function AdminContestDetailPage() {
  const { contestId } = useParams()
  const { data: contest, isLoading, isError, refetch } = useAdminContest(contestId)
  const transition = useTransitionContest(contestId!)
  const toast = useToast()
  const [editOpen, setEditOpen] = useState(false)

  if (isLoading) return <Skeleton className="h-64 w-full" />
  if (isError) return <ErrorState onRetry={() => refetch()} />
  if (!contest)
    return (
      <EmptyState
        title="Конкурс не найден"
        description="Возможно, у вас нет доступа к этому конкурсу."
        action={
          <Link to="/admin/contests" className="text-[14px] font-medium text-brand">
            ← К списку конкурсов
          </Link>
        }
      />
    )

  const status = contestStatusMeta[contest.status]
  const actions = actionsByStatus[contest.status]
  const canEdit = canEditContest(contest.access_level)
  const canManage = canManageParticipants(contest.access_level)

  function runTransition(action: 'publish' | 'finish' | 'archive') {
    transition.mutate(action, {
      onSuccess: () => toast({ title: `${actionMeta[action].label}: готово`, tone: 'success' }),
      onError: () => toast({ title: 'Не удалось изменить статус', tone: 'error' }),
    })
  }

  return (
    <div>
      <Link
        to="/admin/contests"
        className="mb-4 inline-flex items-center gap-1 text-[14px] text-muted hover:text-ink"
      >
        <ArrowLeft className="h-4 w-4" /> Конкурсы
      </Link>

      <header className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-[28px] font-bold tracking-tight text-ink">{contest.name}</h1>
            <Badge tone={status.tone}>{status.label}</Badge>
          </div>
          <p className="mt-1 flex items-center gap-1.5 text-[14px] text-muted">
            <Calendar className="h-4 w-4" />
            {contest.start_at ? formatDate(contest.start_at) : 'Даты не заданы'}
            {contest.end_at ? ` — ${formatDate(contest.end_at)}` : ''}
          </p>
        </div>
        <div className="flex gap-2">
          {canEdit && contest.status !== 'ARCHIVED' && (
            <Button size="sm" variant="secondary" onClick={() => setEditOpen(true)}>
              <Pencil className="h-4 w-4" /> Редактировать
            </Button>
          )}
          {canEdit &&
            actions.map((a) => {
              const Icon = actionMeta[a].icon
              return (
                <Button
                  key={a}
                  size="sm"
                  variant={a === 'archive' ? 'secondary' : 'primary'}
                  loading={transition.isPending}
                  onClick={() => runTransition(a)}
                >
                  <Icon className="h-4 w-4" /> {actionMeta[a].label}
                </Button>
              )
            })}
        </div>
      </header>

      <div className="mb-8 grid grid-cols-2 gap-4 sm:max-w-md">
        <Card>
          <CardBody className="flex items-center gap-3 py-4">
            <Users className="h-5 w-5 text-brand" />
            <div>
              <p className="text-[22px] font-bold leading-none text-ink">
                {contest.participants_count}
              </p>
              <p className="mt-1 text-[13px] text-muted">Конкурсантов</p>
            </div>
          </CardBody>
        </Card>
        <Card>
          <CardBody className="flex items-center gap-3 py-4">
            <PencilRuler className="h-5 w-5 text-brand" />
            <div>
              <p className="text-[22px] font-bold leading-none text-ink">
                {contest.challenges_count}
              </p>
              <p className="mt-1 text-[13px] text-muted">Испытаний</p>
            </div>
          </CardBody>
        </Card>
      </div>

      <div className="mb-8">
        <ChallengesSection contestId={contest.id} canEdit={canEdit} />
      </div>

      <h2 className="mb-3 text-[20px] font-semibold text-ink">Конкурсанты</h2>
      <ContestantsTable contestId={contest.id} canManage={canManage} />

      <EditContestDialog contest={contest} open={editOpen} onOpenChange={setEditOpen} />
    </div>
  )
}
