import { ChevronDown, ChevronLeft, ChevronRight, ChevronUp, ListOrdered, Play, RotateCcw, Shuffle, Square } from 'lucide-react'
import { useState } from 'react'
import { useFinishLive, useReorderLiveDraw, useRestartLive, useSetLiveQuestionPlan, useShuffleLiveDraw, useStartLive, useStepLiveQuestion } from '@/entities/evaluation/admin-queries'
import { contestantLabel, liveStateLabels, type LiveSnapshot, type LiveState, type LivesQuestionLog } from '@/entities/evaluation/types'
import { answerLabel } from '@/entities/evaluation/yes-no-buttons'
import { QuestionPlanDialog } from '@/pages/admin/question-plan-dialog'
import { ApiRequestError } from '@/shared/api/client'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { UserAvatar } from '@/shared/ui/avatar'
import { EmptyState } from '@/shared/ui/states'
import { toast } from 'sonner'
import { LivesHearts, eliminatedLabel } from '@/entities/evaluation/lives-hearts'

function cmdError(err: unknown): string {
  if (err instanceof ApiRequestError && err.code === 'EVALUATION_REVISION_CONFLICT') {
    return 'Кто-то изменил сессию. Экран обновлён — повторите команду.'
  }
  return 'Не удалось выполнить команду'
}

export function LivesLiveConsole({
  snap,
  canEdit,
  challengeId,
}: {
  snap: LiveSnapshot
  canEdit: boolean
  challengeId: string
}) {
  const start = useStartLive(challengeId)
  const finish = useFinishLive(challengeId)
  const restart = useRestartLive(challengeId)
  const step = useStepLiveQuestion(challengeId)
  const plan = useSetLiveQuestionPlan(challengeId)
  const shuffle = useShuffleLiveDraw(challengeId)
  const reorder = useReorderLiveDraw(challengeId)
  const [planOpen, setPlanOpen] = useState(false)
  const pending = start.isPending || finish.isPending || restart.isPending || step.isPending || plan.isPending || shuffle.isPending || reorder.isPending
  const rev = snap.session_revision
  const finished = snap.state === 'FINISHED'
  const drawLocked = snap.draw_locked === true || (snap.state !== 'NOT_STARTED' && snap.state !== 'PREPARING')
  const undrawn = snap.contestants.some((c) => c.draw_number == null)
  const q = snap.current_question_number ?? snap.lives?.current_question ?? 1
  const board = snap.lives
  const starting = snap.starting_lives ?? board?.starting_lives ?? 3
  const correct = snap.correct_answer ?? board?.correct_answer ?? null
  const total = snap.question_count && snap.question_count > 0 ? snap.question_count : 0
  const atLast = total > 0 && q >= total

  function run(fn: () => void) {
    if (!canEdit) return
    fn()
  }

  function onErr(err: unknown) {
    toast.error(cmdError(err))
  }

  function onShuffle() {
    if (snap.drawn && !window.confirm('Переиграть жеребьёвку? Текущий порядок будет заменён.')) {
      return
    }
    shuffle.mutate(undefined, {
      onError: onErr,
      onSuccess: () => toast.success('Жеребьёвка проведена'),
    })
  }

  function move(from: number, to: number) {
    if (to < 0 || to >= snap.contestants.length) return
    const ids = snap.contestants.map((c) => c.user_id)
    const [item] = ids.splice(from, 1)
    ids.splice(to, 0, item)
    reorder.mutate(ids, { onError: onErr })
  }

  const questions = board?.questions ?? []

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardBody className="flex flex-col gap-5 py-5">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-[13px] uppercase tracking-wide text-muted-2">2 к 1</p>
              <h2 className="text-[22px] font-semibold text-ink">{snap.challenge_title}</h2>
              <p className="mt-1 text-[13px] text-muted">
                Лог администратора — события ответственного жюри.
                {board && !board.operator_user_id
                  ? ' Назначьте ответственное жюри на вкладке «Оценивание».'
                  : board?.official
                    ? ' Показан официальный лог.'
                    : ''}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Badge
                tone={
                  snap.state === 'LIVE' ? 'success' : snap.state === 'PAUSED' ? 'warning' : 'neutral'
                }
              >
                {liveStateLabels[snap.state as LiveState] ?? snap.state}
              </Badge>
              <span className="text-[13px] text-muted">жюри online: {snap.jury_online}</span>
            </div>
          </div>

          <div className="rounded-card bg-surface-2 px-4 py-5">
            <p className="text-[13px] text-muted">Текущий вопрос</p>
            <p className="mt-1 font-mono text-[36px] font-semibold leading-none text-ink">
              №{q}
              {total > 0 ? <span className="ml-2 font-sans text-[16px] font-medium text-muted">из {total}</span> : null}
            </p>
            <p className="mt-2 text-[14px] text-ink">
              Правильный ответ: <span className="font-medium">{answerLabel(correct)}</span>
            </p>
            {canEdit && !finished && (
              <div className="mt-4 flex flex-col gap-3">
                <div className="flex flex-wrap gap-2">
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={pending || q <= 1}
                    onClick={() => run(() => step.mutate({ baseRevision: rev, extra: '-1' }, { onError: onErr }))}
                  >
                    <ChevronLeft className="h-4 w-4" /> Предыдущий вопрос
                  </Button>
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={pending || atLast}
                    onClick={() => run(() => step.mutate({ baseRevision: rev, extra: '1' }, { onError: onErr }))}
                  >
                    Следующий вопрос <ChevronRight className="h-4 w-4" />
                  </Button>
                  <Button size="sm" disabled={pending} onClick={() => setPlanOpen(true)}>
                    <ListOrdered className="h-4 w-4" /> Задать вопросы
                  </Button>
                </div>
              </div>
            )}
          </div>

          {canEdit && (
            <div className="flex flex-wrap gap-2">
              {(snap.state === 'NOT_STARTED' || snap.state === 'PREPARING' || snap.state === 'PAUSED') && (
                <Button size="sm" disabled={pending} onClick={() => run(() => start.mutate({ baseRevision: rev }, { onError: onErr }))}>
                  <Play className="h-4 w-4" />
                  {snap.state === 'PAUSED' ? 'Продолжить' : 'Старт'}
                </Button>
              )}
              {snap.state === 'FINISHED' && (
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={pending}
                  onClick={() => {
                    if (!window.confirm('Провести испытание заново? Журнал жизней и ответы конкурсантов будут очищены. Ключ вопросов сохранится, жеребьёвка тоже.')) {
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
        </CardBody>
      </Card>

      <Card>
        <CardBody className="py-5">
          <h3 className="text-[16px] font-semibold text-ink">Лог вопросов</h3>
          <p className="mt-1 text-[13px] text-muted">Правильный ответ и кто потерял жизнь по логу ответственного жюри.</p>
          {questions.length === 0 ? (
            <p className="mt-3 text-[14px] text-muted">Вопросов пока нет.</p>
          ) : (
            <ol className="mt-3 flex flex-col gap-2">
              {questions.map((item) => (
                <QuestionLogRow key={item.question_number} item={item} />
              ))}
            </ol>
          )}
        </CardBody>
      </Card>

      <Card>
        <CardBody className="py-5">
          <div className="mb-3 flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 className="text-[16px] font-semibold text-ink">Конкурсанты</h3>
              <p className="mt-1 text-[13px] text-muted">Жизни по логу ответственного жюри. Старт: {starting}.</p>
            </div>
            {canEdit && !drawLocked && snap.contestants.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {(!snap.drawn || undrawn) && (
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={pending}
                    onClick={() =>
                      reorder.mutate(
                        snap.contestants.map((c) => c.user_id),
                        { onError: onErr, onSuccess: () => toast.success('Порядок зафиксирован') },
                      )
                    }
                  >
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
              {snap.contestants.map((c, i) => (
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
                  <div className="flex min-w-0 flex-1 items-center justify-between rounded-[10px] px-3 py-2">
                    <span className="flex min-w-0 items-center gap-2">
                      <UserAvatar src={c.avatar_url} name={c.full_name} size={28} />
                      <span className="min-w-0 truncate text-[14px] text-ink">{contestantLabel(c)}</span>
                    </span>
                    <span className="flex shrink-0 items-center gap-2">
                      {c.answer ? (
                        <Badge tone={c.answer === 'YES' ? 'success' : 'danger'}>{answerLabel(c.answer)}</Badge>
                      ) : null}
                      <LivesHearts lives={c.eliminated ? 0 : (c.lives ?? starting)} starting={starting} />
                      {c.eliminated ? (
                        <Badge tone="danger">{eliminatedLabel(c.eliminated_question)}</Badge>
                      ) : null}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </CardBody>
      </Card>

      <QuestionPlanDialog
        open={planOpen}
        snap={snap}
        pending={plan.isPending}
        onClose={() => setPlanOpen(false)}
        onSave={(questionCount, answers) =>
          plan.mutate(
            { questionCount, answers },
            {
              onError: onErr,
              onSuccess: () => {
                toast.success(`Сохранены ответы на ${questionCount} ${questionWord(questionCount)}`)
                setPlanOpen(false)
              },
            },
          )
        }
      />
    </div>
  )
}

function questionWord(n: number): string {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return 'вопрос'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return 'вопроса'
  return 'вопросов'
}

function QuestionLogRow({ item }: { item: LivesQuestionLog }) {
  const answers = item.answers ?? []
  return (
    <li
      className={
        'flex flex-col gap-1 rounded-[10px] border px-3 py-2 ' +
        (item.current ? 'border-brand bg-brand-subtle' : 'border-border')
      }
    >
      <p className="text-[14px] font-medium text-ink">
        Вопрос {item.question_number}
        {item.current ? <span className="ml-2 text-[12px] font-normal text-brand">текущий</span> : null}
        <span className="ml-2 text-[13px] font-normal text-muted">
          правильный: {answerLabel(item.correct_answer)}
        </span>
      </p>
      {answers.length > 0 ? (
        <p className="text-[13px] text-ink">
          {answers
            .map((a) => `${a.full_name} — ${answerLabel(a.answer)}${a.mismatch ? ' (ошибка)' : ''}`)
            .join(', ')}
        </p>
      ) : (
        <p className="text-[13px] text-muted">Ответы ещё не зафиксированы</p>
      )}
      {item.losses.length === 0 ? (
        <p className="text-[13px] text-muted">Никто не потерял жизнь</p>
      ) : (
        <p className="text-[13px] text-ink">Потеряли жизнь: {item.losses.map((l) => l.full_name).join(', ')}</p>
      )}
    </li>
  )
}
