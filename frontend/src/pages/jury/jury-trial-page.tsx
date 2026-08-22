import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, ChevronLeft, ChevronRight } from 'lucide-react'
import { useJuryLive } from '@/entities/evaluation/jury-queries'
import {
  contestantLabel,
  formatLiveTimer,
  liveStateLabels,
  type LiveContestant,
  type LiveSnapshot,
  type LiveState,
} from '@/entities/evaluation/types'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { UserAvatar } from '@/shared/ui/avatar'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'
import { useEffect, useState } from 'react'
import { JuryScorecardSection } from './jury-scorecard'
import { JuryLivesBoard } from './jury-lives-board'
import { JuryRemoteTrial } from './jury-remote-page'

export function JuryTrialPage() {
  const { challengeId } = useParams()
  const q = useJuryLive(challengeId)
  const snap = q.data
  const remaining = snap?.timer_remaining_seconds ?? null
  const paused = snap?.state === 'PAUSED' || !!snap?.paused_at
  const rev = snap?.session_revision
  const [rem, setRem] = useState<number | null>(null)

  useEffect(() => {
    if (remaining == null) {
      setRem(null)
      return
    }
    if (paused) {
      setRem(remaining)
      return
    }
    const origin = Date.now()
    setRem(remaining)
    const id = window.setInterval(() => setRem(remaining - (Date.now() - origin) / 1000), 250)
    return () => window.clearInterval(id)
  }, [rev, remaining, paused])

  if (q.isLoading) return <Skeleton className="h-48 w-full" />
  if (q.isError) return <ErrorState onRetry={() => q.refetch()} />
  if (!snap) return <EmptyState title="Нет live-сессии" description="Администратор ещё не открыл испытание." />

  if (snap.scheme_type === 'NUMERIC_RESULT') {
    return (
      <EmptyState
        title="Жюри в этом испытании не участвует"
        description="Числовой результат выставляет администратор испытания."
      />
    )
  }

  if (snap.scheme_type === 'REMOTE_CRITERIA') {
    return (
      <JuryRemoteTrial
        challengeId={challengeId!}
        title={snap.challenge_title}
        contestants={snap.contestants}
      />
    )
  }

  if (snap.scheme_type === 'ELIMINATION_LIVES') {
    return <JuryLivesTrial snap={snap} />
  }

  return <JuryTrialBody snap={snap} remaining={rem} />
}

function JuryLivesTrial({ snap }: { snap: LiveSnapshot }) {
  return (
    <div className="flex flex-col gap-4">
      <Link to="/jury" className="inline-flex items-center gap-1 text-[14px] text-muted hover:text-ink">
        <ArrowLeft className="h-4 w-4" /> К испытаниям
      </Link>
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-[28px] font-bold tracking-tight text-ink">{snap.challenge_title}</h1>
          <p className="mt-1 text-[15px] text-muted">2 к 1. Отметьте ошибку на текущем вопросе или восстановите жизнь.</p>
        </div>
        <Badge tone={snap.state === 'LIVE' ? 'success' : snap.state === 'FINISHED' ? 'neutral' : 'warning'}>
          {liveStateLabels[snap.state as LiveState] ?? snap.state}
        </Badge>
      </header>
      <JuryLivesBoard snap={snap} />
    </div>
  )
}

function JuryTrialBody({ snap, remaining }: { snap: LiveSnapshot; remaining: number | null }) {
  const liveId = snap.current_contestant_user_id
  const [followLive, setFollowLive] = useState(true)
  const [manualId, setManualId] = useState<string | null>(null)
  const viewedId = followLive ? liveId : (manualId ?? liveId)
  const viewingOther = !!viewedId && !!liveId && viewedId !== liveId
  const viewed = snap.contestants.find((c) => c.user_id === viewedId) ?? null
  const live = snap.current
  const idx = snap.contestants.findIndex((c) => c.user_id === viewedId)

  function select(id: string) {
    if (id === liveId) {
      setFollowLive(true)
      setManualId(null)
      return
    }
    setFollowLive(false)
    setManualId(id)
  }

  function selectAt(i: number) {
    const c = snap.contestants[i]
    if (c) select(c.user_id)
  }

  return (
    <div className="flex flex-col gap-4">
      <Link to="/jury" className="inline-flex items-center gap-1 text-[14px] text-muted hover:text-ink">
        <ArrowLeft className="h-4 w-4" /> К испытаниям
      </Link>
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-[28px] font-bold tracking-tight text-ink">{snap.challenge_title}</h1>
          <p className="mt-1 text-[15px] text-muted">
            Карточка следует за сценой. Можно открыть другого конкурсанта и вернуться к текущему.
          </p>
        </div>
        <Badge
          tone={
            snap.state === 'LIVE'
              ? 'success'
              : snap.state === 'PAUSED' || snap.state === 'APPLAUSE'
                ? 'warning'
                : 'neutral'
          }
        >
          {liveStateLabels[snap.state as LiveState] ?? snap.state}
        </Badge>
      </header>

      <Card>
        <CardBody className="flex flex-col gap-4 py-6">
          <div>
            <p className="text-[13px] text-muted">Сейчас выступает</p>
            <div className="mt-2 flex items-center gap-3">
              {live ? <UserAvatar src={live.avatar_url} name={live.full_name} size={56} /> : null}
              <div className="min-w-0">
                <p className="text-[22px] font-semibold text-ink">
                  {live ? contestantLabel(live) : 'Конкурсант не выбран'}
                </p>
                {live?.speech_duration_seconds != null ? (
                  <p className="text-[13px] text-muted">
                    Выступил за {formatLiveTimer(live.speech_duration_seconds)}
                  </p>
                ) : null}
              </div>
            </div>
          </div>
          <div>
            <p className="text-[13px] text-muted">Таймер</p>
            <p
              className={
                remaining != null && remaining < 0
                  ? 'font-mono text-[36px] font-semibold text-danger'
                  : 'font-mono text-[36px] font-semibold text-ink'
              }
            >
              {formatLiveTimer(remaining)}
            </p>
          </div>
        </CardBody>
      </Card>

      {viewingOther && live && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-card border border-brand/30 bg-brand-subtle px-4 py-3">
          <p className="text-[14px] text-ink">
            Вы просматриваете другого конкурсанта. Сейчас выступает: {contestantLabel(live)}
          </p>
          <Button size="sm" onClick={() => select(live.user_id)}>
            К текущему
          </Button>
        </div>
      )}

      <Card>
        <CardBody className="flex flex-col gap-3 py-5">
          <div>
            <p className="text-[13px] text-muted">{viewingOther ? 'Вы оцениваете' : 'Карточка оценок'}</p>
            <div className="mt-1 flex items-center gap-3">
              {viewed ? <UserAvatar src={viewed.avatar_url} name={viewed.full_name} size={40} /> : null}
              <div className="min-w-0">
                <p className="text-[18px] font-semibold text-ink">
                  {viewed ? contestantLabel(viewed) : 'Выберите конкурсанта'}
                </p>
                {viewed && idx >= 0 ? (
                  <p className="text-[13px] text-muted">
                    {idx + 1} из {snap.contestants.length}
                    {followLive ? ' · следим за сценой' : ''}
                  </p>
                ) : null}
              </div>
            </div>
          </div>
          {snap.contestants.length > 0 && (
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="secondary"
                disabled={idx <= 0}
                onClick={() => selectAt(idx < 0 ? 0 : idx - 1)}
              >
                <ChevronLeft className="h-4 w-4" /> Предыдущий
              </Button>
              <Button
                size="sm"
                variant="secondary"
                disabled={idx >= snap.contestants.length - 1 || snap.contestants.length === 0}
                onClick={() => selectAt(idx < 0 ? 0 : idx + 1)}
              >
                Следующий <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </CardBody>
      </Card>

      <JuryScorecardSection
        challengeId={snap.challenge_id}
        contestantUserId={viewedId}
        sessionState={snap.state}
      />

      <ContestantList
        contestants={snap.contestants}
        viewedId={viewedId}
        liveId={liveId}
        onSelect={select}
      />
    </div>
  )
}

function ContestantList({
  contestants,
  viewedId,
  liveId,
  onSelect,
}: {
  contestants: LiveContestant[]
  viewedId: string | null
  liveId: string | null
  onSelect: (id: string) => void
}) {
  return (
    <Card>
      <CardBody className="py-5">
        <h3 className="text-[16px] font-semibold text-ink">Все конкурсанты</h3>
        <p className="mt-1 text-[13px] text-muted">
          Нажмите, чтобы открыть карточку оценок. Порядок как в жеребьёвке.
        </p>
        {contestants.length === 0 ? (
          <EmptyState title="Нет конкурсантов" description="Организатор ещё не добавил участников." />
        ) : (
          <ul className="mt-3 flex flex-col gap-1">
            {contestants.map((c) => {
              const viewing = c.user_id === viewedId
              const onStage = c.user_id === liveId
              return (
                <li key={c.user_id}>
                  <button
                    type="button"
                    onClick={() => onSelect(c.user_id)}
                    className={
                      'flex w-full items-center justify-between gap-2 rounded-[10px] px-3 py-2 text-left text-[14px] ' +
                      (viewing ? 'bg-brand-subtle text-brand' : 'hover:bg-muted/10')
                    }
                  >
                    <span className="flex min-w-0 items-center gap-2">
                      <UserAvatar src={c.avatar_url} name={c.full_name} size={28} />
                      <span className="truncate">{contestantLabel(c)}</span>
                    </span>
                    {onStage ? (
                      <Badge tone={viewing ? 'brand' : 'success'}>на сцене</Badge>
                    ) : null}
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </CardBody>
    </Card>
  )
}
