import { useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, ChevronRight, ClipboardList, FileText, Inbox, MapPin, Pencil } from 'lucide-react'
import { useAdminChallenge } from '@/entities/challenge/admin-queries'
import { useAdminContest } from '@/entities/contest/queries'
import { canEditContest } from '@/entities/contest/types'
import { formatDateTime } from '@/shared/lib/format'
import { useAppConfig } from '@/shared/config/use-app-config'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'
import { EditChallengeDialog } from './edit-challenge-dialog'
import { ChallengeDangerZone } from './challenge-danger-zone'
import { intakeBadge, liveConductedMeta, schemeTypeLabel } from './challenge-status'

/** Хаб испытания: два раздела — приём файлов/ТЗ и проведение. Доступы разъедутся позже. */
export function ChallengeHubPage() {
  const { challengeId } = useParams()
  const challengeQ = useAdminChallenge(challengeId)
  const { data: contest } = useAdminContest(challengeQ.data?.contest_id)
  const { data: appConfig } = useAppConfig()
  const [metaOpen, setMetaOpen] = useState(false)

  if (challengeQ.isLoading) return <Skeleton className="h-64 w-full" />
  if (challengeQ.isError) return <ErrorState onRetry={() => challengeQ.refetch()} />
  const challenge = challengeQ.data
  if (!challenge) {
    return (
      <EmptyState
        title="Испытание не найдено"
        description="Возможно, у вас нет доступа к этому испытанию."
      />
    )
  }

  const intake = intakeBadge(challenge)
  const conducted = liveConductedMeta(challenge.live_state)
  const canEdit = canEditContest(contest?.access_level)

  return (
    <div>
      <Link
        to={`/admin/contests/${challenge.contest_id}`}
        className="mb-4 inline-flex items-center gap-1 text-[14px] text-muted hover:text-ink"
      >
        <ArrowLeft className="h-4 w-4" /> К конкурсу
      </Link>

      <header className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-[28px] font-bold tracking-tight text-ink">{challenge.title}</h1>
            <Badge tone={conducted.tone}>{conducted.label}</Badge>
          </div>
          <p className="mt-1 text-[14px] text-muted">{schemeTypeLabel(challenge.scheme_type)}</p>
          {(challenge.held_at || challenge.venue) && (
            <p className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-[14px] text-muted">
              {challenge.held_at && <span>{formatDateTime(challenge.held_at)}</span>}
              {challenge.venue && (
                <span className="inline-flex items-center gap-1">
                  <MapPin className="h-3.5 w-3.5" />
                  {challenge.venue}
                </span>
              )}
            </p>
          )}
        </div>
        {canEdit && challenge.status !== 'ARCHIVED' && (
          <Button size="sm" variant="secondary" onClick={() => setMetaOpen(true)}>
            <Pencil className="h-4 w-4" /> Редактировать
          </Button>
        )}
      </header>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <HubCard
          to={`/admin/challenges/${challenge.id}/intake`}
          icon={Inbox}
          title="Приём файлов и ТЗ"
          description={
            challenge.accepts_submissions
              ? 'Конструктор формы, публикация приёма и ответы участников.'
              : 'Приём выключен. Форму и публикацию всё равно можно настроить здесь.'
          }
          extra={<Badge tone={intake.tone}>{intake.label}</Badge>}
        />
        <HubCard
          to={`/admin/challenges/${challenge.id}/briefing`}
          icon={FileText}
          title="Материалы для конкурсантов"
          description="Текст и файлы в личном кабинете, время публикации и персональная выдача."
        />
        <HubCard
          to={`/admin/challenges/${challenge.id}/run`}
          icon={ClipboardList}
          title="Проведение испытания"
          description="Схема оценивания, Live и оценки жюри."
          extra={<Badge tone={conducted.tone}>{conducted.label}</Badge>}
        />
      </div>

      <EditChallengeDialog challenge={challenge} open={metaOpen} onOpenChange={setMetaOpen} />

      {canEdit && appConfig?.features.jury === true && challengeId && (
        <div className="mt-6">
          <ChallengeDangerZone challengeId={challenge.id} />
        </div>
      )}
    </div>
  )
}

function HubCard({
  to,
  icon: Icon,
  title,
  description,
  extra,
}: {
  to: string
  icon: typeof Inbox
  title: string
  description: string
  extra?: ReactNode
}) {
  return (
    <Link to={to} className="block">
      <Card className="h-full transition hover:border-brand/40">
        <CardBody className="flex items-start gap-4 py-5">
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-btn bg-brand-subtle text-brand">
            <Icon className="h-5 w-5" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-[18px] font-semibold text-ink">{title}</h2>
              {extra}
            </div>
            <p className="mt-1 text-[14px] text-muted">{description}</p>
          </div>
          <ChevronRight className="mt-1 h-5 w-5 shrink-0 text-muted-2" />
        </CardBody>
      </Card>
    </Link>
  )
}
