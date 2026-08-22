import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp, Pause, Play, RotateCcw, Shuffle, Square } from 'lucide-react'
import { Fragment, useEffect, useState } from 'react'
import {
  useAdminLive,
  useCompleteLiveContestant,
  useEndSpeechLive,
  useFinishLive,
  usePauseLive,
  useReorderLiveDraw,
  useRestartLive,
  useRestartLiveTimer,
  useSetLiveContestant,
  useSetLiveDurations,
  useSetLivePhase,
  useShuffleLiveDraw,
  useStartLive,
} from '@/entities/evaluation/admin-queries'
import {
  contestantLabel,
  formatLiveTimer,
  liveStateLabels,
  type LiveSnapshot,
  type LiveState,
} from '@/entities/evaluation/types'
import { ApiRequestError } from '@/shared/api/client'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { UserAvatar } from '@/shared/ui/avatar'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'
import { toast } from 'sonner'
import { LivesLiveConsole } from './evaluation-lives-live'

function contestantTitle(c: Parameters<typeof contestantLabel>[0]): string {
  return contestantLabel(c)
}

function minutesOf(snap: LiveSnapshot, state: LiveState, fallbackMin: number): string {
  const p = snap.phases.find((x) => x.maps_to_state === state)
  const sec = p?.duration_seconds ?? fallbackMin * 60
  return String(Math.max(1, Math.round(sec / 60)))
}

function useDisplayedRemaining(snap: LiveSnapshot | undefined) {
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
  return rem
}

function cmdError(err: unknown): string {
  if (err instanceof ApiRequestError && err.code === 'EVALUATION_REVISION_CONFLICT') {
    return 'Кто-то изменил сессию. Экран обновлён — повторите команду.'
  }
  return 'Не удалось выполнить команду'
}

export function EvaluationLiveSection({ challengeId, canEdit }: { challengeId: string; canEdit: boolean }) {
  const q = useAdminLive(challengeId)
  if (q.isLoading) return <Skeleton className="h-64 w-full" />
  if (q.isError) return <ErrorState onRetry={() => q.refetch()} />
  const snap = q.data
  if (!snap) return <EmptyState title="Нет live-состояния" description="Обновите страницу." />
  if (snap.scheme_type === 'NUMERIC_RESULT') {
    return (
      <EmptyState
        title="Live для этого типа не нужен"
        description="Числовой результат выставляет администратор на вкладке «Оценки». Жеребьёвки нет."
      />
    )
  }
  if (snap.scheme_type === 'REMOTE_CRITERIA') {
    return (
      <EmptyState
        title="Live для заочного оценивания не нужен"
        description="Заочное жюри ставит баллы по сданным работам в своём кабинете. Вкладка Live скрыта."
      />
    )
  }
  if (snap.scheme_type === 'ELIMINATION_LIVES') {
    return <LivesLiveConsole snap={snap} canEdit={canEdit} challengeId={challengeId} />
  }
  return <LiveConsole snap={snap} canEdit={canEdit} challengeId={challengeId} />
}

function LiveConsole({
  snap,
  canEdit,
  challengeId,
}: {
  snap: LiveSnapshot
  canEdit: boolean
  challengeId: string
}) {
  const remaining = useDisplayedRemaining(snap)
  const start = useStartLive(challengeId)
  const pause = usePauseLive(challengeId)
  const finish = useFinishLive(challengeId)
  const restart = useRestartLive(challengeId)
  const restartTimer = useRestartLiveTimer(challengeId)
  const completeContestant = useCompleteLiveContestant(challengeId)
  const endSpeech = useEndSpeechLive(challengeId)
  const setContestant = useSetLiveContestant(challengeId)
  const setPhase = useSetLivePhase(challengeId)
  const setDurations = useSetLiveDurations(challengeId)
  const shuffle = useShuffleLiveDraw(challengeId)
  const reorder = useReorderLiveDraw(challengeId)
  const pending =
    start.isPending ||
    pause.isPending ||
    finish.isPending ||
    restart.isPending ||
    restartTimer.isPending ||
    completeContestant.isPending ||
    endSpeech.isPending ||
    setContestant.isPending ||
    setPhase.isPending ||
    setDurations.isPending ||
    shuffle.isPending ||
    reorder.isPending
  const rev = snap.session_revision
  const idx = snap.contestants.findIndex((c) => c.user_id === snap.current_contestant_user_id)
  const finished = snap.state === 'FINISHED'
  const paused = snap.state === 'PAUSED'
  const hasRunningTimer = snap.phase_started_at != null
  const drawLocked = snap.draw_locked === true || (snap.state !== 'NOT_STARTED' && snap.state !== 'PREPARING')
  const undrawn = snap.contestants.some((c) => c.draw_number == null)
  const phases = snap.phases.filter((p) => p.maps_to_state !== 'SCORING' && p.maps_to_state !== 'POST_SCORING')
  const speechPhase = phases.find((p) => p.maps_to_state === 'LIVE')
  const canApplause =
    !!snap.current &&
    (snap.state === 'LIVE' || (snap.state === 'PAUSED' && snap.current_phase_id === speechPhase?.id))

  function run(fn: () => void) {
    if (!canEdit) return
    fn()
  }

  function onErr(err: unknown) {
    toast.error(cmdError(err))
  }

  function selectAt(i: number) {
    const c = snap.contestants[i]
    if (!c) return
    setContestant.mutate({ baseRevision: rev, extra: c.user_id }, { onError: onErr })
  }

  function onShuffle() {
    if (snap.drawn && !window.confirm('Переиграть жеребьёвку? Текущий порядок выступлений будет заменён.')) {
      return
    }
    shuffle.mutate(undefined, {
      onError: onErr,
      onSuccess: () => toast.success('Жеребьёвка проведена'),
    })
  }

  function onLockOrder() {
    reorder.mutate(
      snap.contestants.map((c) => c.user_id),
      {
        onError: onErr,
        onSuccess: () => toast.success('Порядок зафиксирован'),
      },
    )
  }

  function move(from: number, to: number) {
    if (to < 0 || to >= snap.contestants.length) return
    const ids = snap.contestants.map((c) => c.user_id)
    const [item] = ids.splice(from, 1)
    ids.splice(to, 0, item)
    reorder.mutate(ids, { onError: onErr })
  }

  const durationLabel =
    snap.phase_duration_seconds != null ? formatLiveTimer(snap.phase_duration_seconds) : 'без лимита'

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardBody className="flex flex-col gap-5 py-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-[13px] uppercase tracking-wide text-muted-2">Испытание</p>
              <h2 className="text-[22px] font-semibold text-ink">{snap.challenge_title}</h2>
            </div>
            <div className="flex items-center gap-2">
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
              <span className="text-[13px] text-muted">жюри online: {snap.jury_online}</span>
            </div>
          </div>

          <div className="rounded-card bg-surface-2 px-4 py-4">
            <p className="text-[13px] text-muted">Текущий конкурсант</p>
            <div className="mt-2 flex items-center gap-3">
              {snap.current ? (
                <UserAvatar src={snap.current.avatar_url} name={snap.current.full_name} size={56} />
              ) : null}
              <div className="min-w-0">
                <p className="text-[22px] font-semibold text-ink">
                  {snap.current ? contestantTitle(snap.current) : 'Не выбран'}
                </p>
                {snap.current && (
                  <p className="text-[13px] text-muted">
                    {snap.current.login}
                    {idx >= 0 ? ` · ${idx + 1} из ${snap.contestants.length}` : ''}
                    {snap.current.speech_duration_seconds != null
                      ? ` · выступил ${formatLiveTimer(snap.current.speech_duration_seconds)}`
                      : ''}
                  </p>
                )}
              </div>
            </div>
            {canEdit && !finished && (
              <div className="mt-3 flex gap-2">
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={pending || idx <= 0}
                  onClick={() => selectAt(idx - 1)}
                >
                  <ChevronLeft className="h-4 w-4" /> Предыдущий
                </Button>
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={pending || idx < 0 || idx >= snap.contestants.length - 1}
                  onClick={() => selectAt(idx + 1)}
                >
                  Следующий <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            )}
          </div>

          <div className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <p className="text-[13px] text-muted">Таймер фазы</p>
              <p
                className={
                  remaining != null && remaining < 0
                    ? 'font-mono text-[36px] font-semibold leading-none text-danger'
                    : 'font-mono text-[36px] font-semibold leading-none text-ink'
                }
              >
                {formatLiveTimer(remaining)}
              </p>
              <p className="mt-1 text-[12px] text-muted-2">лимит {durationLabel}</p>
            </div>
            {canEdit && (
              <div className="flex flex-wrap gap-2">
                {(snap.state === 'NOT_STARTED' || snap.state === 'PREPARING' || (paused && hasRunningTimer)) && (
                  <Button
                    size="sm"
                    disabled={pending}
                    onClick={() => run(() => start.mutate({ baseRevision: rev }, { onError: onErr }))}
                  >
                    <Play className="h-4 w-4" />
                    {snap.state === 'NOT_STARTED' || snap.state === 'PREPARING' ? 'Старт' : 'Продолжить'}
                  </Button>
                )}
                {paused && hasRunningTimer && (
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={pending}
                    onClick={() => run(() => restartTimer.mutate({ baseRevision: rev }, { onError: onErr }))}
                  >
                    <RotateCcw className="h-4 w-4" /> Начать отсчёт заново
                  </Button>
                )}
                {snap.state !== 'NOT_STARTED' &&
                  snap.state !== 'PAUSED' &&
                  snap.state !== 'FINISHED' &&
                  snap.state !== 'APPLAUSE' && (
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={pending}
                    onClick={() => run(() => pause.mutate({ baseRevision: rev }, { onError: onErr }))}
                  >
                    <Pause className="h-4 w-4" /> Остановить
                  </Button>
                )}
                {snap.state === 'FINISHED' && (
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={pending}
                    onClick={() => {
                      if (
                        !window.confirm(
                          'Провести испытание заново? Live-сессия сбросится, жеребьёвка сохранится.',
                        )
                      ) {
                        return
                      }
                      run(() => restart.mutate({ baseRevision: rev }, { onError: onErr }))
                    }}
                  >
                    <RotateCcw className="h-4 w-4" /> Провести заново
                  </Button>
                )}
                {snap.state !== 'FINISHED' && snap.state !== 'NOT_STARTED' && (
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={pending}
                    onClick={() => run(() => finish.mutate({ baseRevision: rev }, { onError: onErr }))}
                  >
                    <Square className="h-4 w-4" /> Завершить
                  </Button>
                )}
              </div>
            )}
          </div>

          <DurationFields
            snap={snap}
            canEdit={canEdit && !finished}
            pending={pending}
            onSave={(speechSeconds, questionsSeconds) =>
              run(() =>
                setDurations.mutate(
                  { baseRevision: rev, speechSeconds, questionsSeconds },
                  {
                    onError: onErr,
                    onSuccess: () => toast.success('Время фаз сохранено'),
                  },
                ),
              )
            }
          />

          {(phases.length > 0 || (canEdit && !finished && snap.state !== 'NOT_STARTED')) && (
            <div>
              <p className="mb-2 text-[13px] font-medium text-ink">Фаза</p>
              <div className="flex flex-wrap gap-2">
                {phases.map((p) => (
                  <Fragment key={p.id}>
                    <Button
                      size="sm"
                      variant={snap.current_phase_id === p.id ? 'primary' : 'secondary'}
                      disabled={!canEdit || pending || finished || snap.state === 'NOT_STARTED'}
                      onClick={() =>
                        run(() => setPhase.mutate({ baseRevision: rev, extra: p.id }, { onError: onErr }))
                      }
                    >
                      {p.title}
                      {p.duration_seconds != null ? ` ${formatLiveTimer(p.duration_seconds)}` : ''}
                    </Button>
                    {p.maps_to_state === 'LIVE' && canEdit && !finished && snap.state !== 'NOT_STARTED' && (
                      <Button
                        size="sm"
                        variant={snap.state === 'APPLAUSE' ? 'primary' : 'secondary'}
                        disabled={pending || !canApplause}
                        onClick={() =>
                          run(() =>
                            endSpeech.mutate(
                              { baseRevision: rev },
                              {
                                onError: onErr,
                                onSuccess: (data) => {
                                  const sec = data.current?.speech_duration_seconds
                                  toast.success(
                                    sec != null
                                      ? `Выступил за ${formatLiveTimer(sec)}. Таймер сброшен — вопросы ещё не начаты.`
                                      : 'Таймер сброшен. Можно начинать вопросы.',
                                  )
                                },
                              },
                            ),
                          )
                        }
                      >
                        Аплодисменты
                      </Button>
                    )}
                  </Fragment>
                ))}
                {canEdit && !finished && snap.state !== 'NOT_STARTED' && snap.current && (
                  <Button
                    size="sm"
                    disabled={pending}
                    className="bg-danger text-white hover:bg-danger/90"
                    onClick={() => {
                      const last = idx >= 0 && idx >= snap.contestants.length - 1
                      const ok = window.confirm(
                        last
                          ? 'Это последний в жеребьёвке. Завершить выступление и остановить таймер?'
                          : 'Завершить выступление текущего конкурсанта и перейти к следующему? Таймер остановится, отсчёт для следующего начнётся после кнопки «Выступление».',
                      )
                      if (!ok) return
                      run(() =>
                        completeContestant.mutate(
                          { baseRevision: rev },
                          {
                            onError: onErr,
                            onSuccess: () =>
                              toast.success(
                                last
                                  ? 'Выступление завершено, таймер остановлен'
                                  : 'Следующий конкурсант выбран. Нажмите «Выступление», чтобы начать отсчёт.',
                              ),
                          },
                        ),
                      )
                    }}
                  >
                    Конкурсант завершил выступление
                  </Button>
                )}
              </div>
            </div>
          )}
        </CardBody>
      </Card>

      <Card>
        <CardBody className="py-5">
          <div className="mb-3 flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 className="text-[16px] font-semibold text-ink">Жеребьёвка</h3>
              <p className="mt-1 text-[13px] text-muted">
                {drawLocked
                  ? 'Порядок зафиксирован на время испытания.'
                  : snap.drawn
                    ? undrawn
                      ? 'Есть конкурсанты без номера — добавьте их в порядок или переиграйте жеребьёвку.'
                      : `Порядок выступлений для этого испытания · ${snap.contestants.length}`
                    : 'Порядок пока по дате добавления. Проведите жеребьёвку или зафиксируйте текущий список.'}
              </p>
            </div>
            {canEdit && !drawLocked && snap.contestants.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {(!snap.drawn || undrawn) && (
                  <Button size="sm" variant="secondary" disabled={pending} onClick={onLockOrder}>
                    Зафиксировать порядок
                  </Button>
                )}
                <Button size="sm" disabled={pending} onClick={onShuffle}>
                  <Shuffle className="h-4 w-4" />
                  {snap.drawn ? 'Переиграть' : 'Провести жеребьёвку'}
                </Button>
              </div>
            )}
          </div>
          {snap.contestants.length === 0 ? (
            <EmptyState title="Нет конкурсантов" description="Добавьте их в карточке конкурса." />
          ) : (
            <ul className="flex flex-col gap-1">
              {snap.contestants.map((c, i) => {
                const active = c.user_id === snap.current_contestant_user_id
                return (
                  <li key={c.user_id} className="flex items-center gap-1">
                    {canEdit && !drawLocked && (
                      <div className="flex flex-col">
                        <button
                          type="button"
                          disabled={pending || i === 0}
                          onClick={() => move(i, i - 1)}
                          className="rounded p-0.5 text-muted hover:bg-muted/10 disabled:opacity-30"
                          aria-label="Выше"
                        >
                          <ChevronUp className="h-4 w-4" />
                        </button>
                        <button
                          type="button"
                          disabled={pending || i === snap.contestants.length - 1}
                          onClick={() => move(i, i + 1)}
                          className="rounded p-0.5 text-muted hover:bg-muted/10 disabled:opacity-30"
                          aria-label="Ниже"
                        >
                          <ChevronDown className="h-4 w-4" />
                        </button>
                      </div>
                    )}
                    <button
                      type="button"
                      disabled={!canEdit || pending || finished}
                      onClick={() => selectAt(i)}
                      className={
                        'flex min-w-0 flex-1 items-center justify-between rounded-[10px] px-3 py-2 text-left text-[14px] ' +
                        (active ? 'bg-brand-subtle text-brand' : 'hover:bg-muted/10')
                      }
                    >
                      <span className="flex min-w-0 items-center gap-2">
                        <UserAvatar src={c.avatar_url} name={c.full_name} size={28} />
                        <span className="truncate">{contestantLabel(c)}</span>
                      </span>
                      {c.speech_duration_seconds != null ? (
                        <span className="ml-2 shrink-0 font-mono text-[12px] text-muted">
                          {formatLiveTimer(c.speech_duration_seconds)}
                        </span>
                      ) : null}
                    </button>
                  </li>
                )
              })}
            </ul>
          )}
        </CardBody>
      </Card>
    </div>
  )
}

function DurationFields({
  snap,
  canEdit,
  pending,
  onSave,
}: {
  snap: LiveSnapshot
  canEdit: boolean
  pending: boolean
  onSave: (speechSeconds: number, questionsSeconds: number) => void
}) {
  const speechSrc = minutesOf(snap, 'LIVE', 8)
  const questionsSrc = minutesOf(snap, 'QUESTIONS', 5)
  const [speechMin, setSpeechMin] = useState(speechSrc)
  const [questionsMin, setQuestionsMin] = useState(questionsSrc)
  useEffect(() => {
    setSpeechMin(speechSrc)
    setQuestionsMin(questionsSrc)
  }, [speechSrc, questionsSrc])

  function save() {
    const speech = Math.round(Number(speechMin) * 60)
    const questions = Math.round(Number(questionsMin) * 60)
    if (!Number.isFinite(speech) || !Number.isFinite(questions) || speech < 30 || questions < 30) {
      toast.error('Укажите время от 1 до 120 минут')
      return
    }
    onSave(speech, questions)
  }

  return (
    <div>
      <p className="mb-2 text-[13px] font-medium text-ink">Время фаз</p>
      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-[13px] text-muted">
          Выступление, мин
          <Input
            type="number"
            min={1}
            max={120}
            className="h-9 w-24"
            value={speechMin}
            disabled={!canEdit || pending}
            onChange={(e) => setSpeechMin(e.target.value)}
          />
        </label>
        <label className="flex flex-col gap-1 text-[13px] text-muted">
          Вопросы, мин
          <Input
            type="number"
            min={1}
            max={120}
            className="h-9 w-24"
            value={questionsMin}
            disabled={!canEdit || pending}
            onChange={(e) => setQuestionsMin(e.target.value)}
          />
        </label>
        {canEdit && (
          <Button size="sm" variant="secondary" disabled={pending} onClick={save}>
            Сохранить время
          </Button>
        )}
      </div>
      <p className="mt-1 text-[12px] text-muted-2">Оценивание открыто всё время, отдельной фазы нет.</p>
    </div>
  )
}
