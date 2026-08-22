import { useState } from 'react'
import { useJuryRestoreLife, useJurySetAnswer } from '@/entities/evaluation/jury-queries'
import { contestantLabel, type LiveContestant, type LiveSnapshot } from '@/entities/evaluation/types'
import { YesNoButtons, type YesNoAnswer } from '@/entities/evaluation/yes-no-buttons'
import { ApiRequestError } from '@/shared/api/client'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { UserAvatar } from '@/shared/ui/avatar'
import { EmptyState } from '@/shared/ui/states'
import { toast } from 'sonner'
import { LivesHearts } from '@/entities/evaluation/lives-hearts'

function lifeError(err: unknown): string {
  if (err instanceof ApiRequestError) {
    if (err.code === 'EVALUATION_SCORING_CLOSED') return 'Испытание завершено'
    if (err.code === 'EVALUATION_LIFE_ELIMINATED') return 'Конкурсант уже выбыл'
  }
  return 'Не удалось сохранить'
}

export function JuryLivesBoard({ snap }: { snap: LiveSnapshot }) {
  const mark = useJurySetAnswer(snap.challenge_id)
  const restore = useJuryRestoreLife(snap.challenge_id)
  const [restoreFor, setRestoreFor] = useState<LiveContestant | null>(null)
  const q = snap.current_question_number ?? snap.lives?.current_question ?? 1
  const total = snap.question_count && snap.question_count > 0 ? snap.question_count : 0
  const starting = snap.starting_lives ?? snap.lives?.starting_lives ?? 3
  const finished = snap.state === 'FINISHED'
  const official = snap.lives?.official === true
  const pending = mark.isPending || restore.isPending

  function onAnswer(c: LiveContestant, answer: YesNoAnswer) {
    mark.mutate(
      { contestantUserId: c.user_id, answer },
      { onError: (err) => toast.error(lifeError(err)) },
    )
  }

  function onRestore(questionNumber: number) {
    if (!restoreFor) return
    restore.mutate(
      { contestantUserId: restoreFor.user_id, questionNumber },
      {
        onError: (err) => toast.error(lifeError(err)),
        onSuccess: () => {
          toast.success(`Жизнь восстановлена за вопрос ${questionNumber}`)
          setRestoreFor(null)
        },
      },
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardBody className="flex flex-wrap items-end justify-between gap-3 py-5">
          <div>
            <p className="text-[13px] text-muted">Текущий вопрос</p>
            <p className="font-mono text-[36px] font-semibold leading-none text-ink">
              №{q}
              {total > 0 ? <span className="ml-2 font-sans text-[16px] font-medium text-muted">из {total}</span> : null}
            </p>
            <p className="mt-2 text-[13px] text-muted">Зафиксируйте ответ конкурсанта: Да или Нет.</p>
          </div>
          {official ? (
            <Badge tone="brand">Ваш лог видит администратор</Badge>
          ) : (
            <p className="text-[13px] text-muted">Отметки пишутся в ваш лог. Жизни считает ответственное жюри.</p>
          )}
        </CardBody>
      </Card>

      {snap.contestants.length === 0 ? (
        <EmptyState title="Нет конкурсантов" description="Организатор ещё не добавил участников." />
      ) : (
        <ul className="flex flex-col gap-2">
          {snap.contestants.map((c) => {
            const lives = c.lives ?? starting
            const out = !!c.eliminated || lives <= 0
            const restoreQs = c.restore_questions ?? []
            return (
              <li key={c.user_id}>
                <Card>
                  <CardBody className="flex flex-col gap-3 py-4 sm:flex-row sm:items-center">
                    <div className="flex min-w-0 items-center gap-3 sm:max-w-sm sm:flex-1">
                      <UserAvatar src={c.avatar_url} name={c.full_name} size={48} />
                      <div className="min-w-0">
                        <p className="truncate text-[16px] font-semibold text-ink">{c.full_name}</p>
                        <p className="truncate text-[13px] text-muted">
                          {c.organization?.trim() || 'Организация не указана'}
                          {c.draw_number != null ? ` · №${c.draw_number}` : ''}
                        </p>
                        <div className="mt-1">
                          <LivesHearts lives={out ? 0 : lives} starting={starting} />
                        </div>
                      </div>
                    </div>
                    <div className="shrink-0 sm:pl-2">
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={pending || finished || restoreQs.length === 0}
                        onClick={() => setRestoreFor(c)}
                      >
                        Восстановить жизнь
                      </Button>
                    </div>
                    {!out ? (
                      <div className="flex flex-wrap items-center gap-2 sm:ml-auto">
                        <YesNoButtons
                          value={c.answer}
                          disabled={finished || pending}
                          onSelect={(answer) => onAnswer(c, answer)}
                        />
                        {official && c.can_reveal ? (
                          <Button
                            size="sm"
                            variant="secondary"
                            disabled={pending}
                            onClick={() => toast.message('Команда «Показать» будет подключена позже')}
                          >
                            Показать
                          </Button>
                        ) : null}
                      </div>
                    ) : null}
                  </CardBody>
                </Card>
              </li>
            )
          })}
        </ul>
      )}

      <Dialog open={!!restoreFor} onOpenChange={(open) => !open && setRestoreFor(null)}>
        <DialogContent
          title="Восстановить жизнь"
          description={
            restoreFor
              ? `Выберите вопрос, за который вернуть жизнь: ${contestantLabel(restoreFor)}`
              : undefined
          }
        >
          <div className="flex flex-col gap-2">
            {(restoreFor?.restore_questions ?? []).map((n) => (
              <Button key={n} variant="secondary" disabled={pending} onClick={() => onRestore(n)}>
                Вопрос {n}
              </Button>
            ))}
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
