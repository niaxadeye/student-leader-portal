import { useEffect, useState } from 'react'
import { RotateCcw, Users } from 'lucide-react'
import { useEvaluation, useReplaceEvaluationJury, useResetEvaluationResults } from '@/entities/evaluation/admin-queries'
import type { AdminScoreboardJury } from '@/entities/evaluation/types'
import { ApiRequestError } from '@/shared/api/client'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { toast } from 'sonner'

type ActionKind = 'results' | 'jury'

function personLabel(j: AdminScoreboardJury): string {
  return j.full_name.trim() || j.login
}

export function ChallengeDangerZone({ challengeId }: { challengeId: string }) {
  const q = useEvaluation(challengeId)
  const [action, setAction] = useState<ActionKind | null>(null)
  const jury = q.data?.jury ?? []
  const contestJury = q.data?.contest_jury ?? jury
  const scope = q.data?.jury_scope ?? 'CONTEST'
  const remote = q.data?.scheme?.type === 'REMOTE_CRITERIA'

  return (
    <>
      <Card className="border-danger/20">
        <CardBody className="flex flex-col gap-4 py-5">
          <div>
            <h2 className="text-[18px] font-semibold text-ink">Сброс испытания</h2>
            <p className="mt-1 text-[13px] text-muted">
              Опасные действия. Сначала подтверждение, затем ваш пароль.
            </p>
            <p className="mt-2 text-[13px] text-muted">
              {scope === 'CHALLENGE'
                ? `Жюри этого испытания: ${jury.length ? jury.map(personLabel).join(', ') : 'никто не назначен'}.`
                : `Жюри конкурса (${contestJury.length}): ${contestJury.length ? contestJury.map(personLabel).join(', ') : 'пока никого'}.`}
              {remote ? ' Состав заочного жюри меняется на вкладке «Оценивание».' : ''}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="secondary"
              className="text-danger hover:bg-danger/10"
              onClick={() => setAction('results')}
            >
              <RotateCcw className="h-4 w-4" /> Сбросить результаты и оценки
            </Button>
            {!remote && (
            <Button
              size="sm"
              variant="secondary"
              className="text-danger hover:bg-danger/10"
              onClick={() => setAction('jury')}
            >
              <Users className="h-4 w-4" /> Сбросить и назначить жюри
            </Button>
            )}
          </div>
        </CardBody>
      </Card>
      <ConfirmPasswordDialog
        challengeId={challengeId}
        action={action}
        contestJury={contestJury}
        currentJuryIds={scope === 'CHALLENGE' ? jury.map((j) => j.user_id) : []}
        onClose={() => setAction(null)}
      />
    </>
  )
}

function ConfirmPasswordDialog({
  challengeId,
  action,
  contestJury,
  currentJuryIds,
  onClose,
}: {
  challengeId: string
  action: ActionKind | null
  contestJury: AdminScoreboardJury[]
  currentJuryIds: string[]
  onClose: () => void
}) {
  const reset = useResetEvaluationResults(challengeId)
  const replace = useReplaceEvaluationJury(challengeId)
  const [step, setStep] = useState<'confirm' | 'password'>('confirm')
  const [password, setPassword] = useState('')
  const [selected, setSelected] = useState<string[]>(currentJuryIds)
  const [error, setError] = useState<string>()

  useEffect(() => {
    if (!action) return
    setStep('confirm')
    setPassword('')
    setError(undefined)
    setSelected(currentJuryIds)
    // сбрасываем форму только при смене действия
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [action])

  const loading = reset.isPending || replace.isPending
  const isJury = action === 'jury'

  function toggle(id: string) {
    setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  }

  function submitPassword() {
    if (!password.trim()) {
      setError('Введите пароль')
      return
    }
    setError(undefined)
    const onError = (err: unknown) => {
      if (err instanceof ApiRequestError && err.code === 'AUTH_WRONG_PASSWORD') {
        setError('Неверный пароль')
        return
      }
      toast.error('Не удалось выполнить действие')
    }
    if (isJury) {
      replace.mutate(
        { password, juryUserIds: selected },
        {
          onSuccess: () => {
            toast.success(
              selected.length
                ? 'Жюри испытания обновлено, оценки сброшены'
                : 'Назначение сброшено: испытание снова видят все жюри конкурса',
            )
            onClose()
          },
          onError,
        },
      )
      return
    }
    reset.mutate(password, {
      onSuccess: () => {
        toast.success('Результаты и оценки сброшены')
        onClose()
      },
      onError,
    })
  }

  return (
    <Dialog open={action != null} onOpenChange={(open) => !open && !loading && onClose()}>
      <DialogContent
        className={isJury ? 'max-w-lg' : undefined}
        title={isJury ? 'Сбросить и назначить жюри' : 'Сбросить результаты и оценки'}
        description={
          step === 'confirm'
            ? isJury
              ? 'Оценки, числовые баллы, жизни и live-сессия этого испытания будут удалены. Схема и жеребьёвка сохранятся.'
              : 'Оценки жюри, числовые баллы, жизни и live-сессия будут удалены. Схема, жеребьёвка и состав жюри сохранятся.'
            : 'Введите свой пароль, чтобы выполнить действие.'
        }
      >
        {step === 'confirm' ? (
          <div className="flex flex-col gap-4">
            {isJury && (
              <div className="flex max-h-56 flex-col gap-1 overflow-auto rounded-btn border border-border p-2">
                {contestJury.length === 0 ? (
                  <p className="px-1 py-2 text-[13px] text-muted">
                    На конкурс пока не назначено жюри. Назначьте роль «Жюри» в разделе пользователей.
                  </p>
                ) : (
                  contestJury.map((j) => (
                    <label
                      key={j.user_id}
                      className="flex cursor-pointer items-center gap-2 rounded-btn px-2 py-1.5 text-[14px] hover:bg-muted/10"
                    >
                      <input
                        type="checkbox"
                        checked={selected.includes(j.user_id)}
                        onChange={() => toggle(j.user_id)}
                      />
                      <span className="min-w-0 truncate">{personLabel(j)}</span>
                      <span className="text-[12px] text-muted">{j.login}</span>
                    </label>
                  ))
                )}
              </div>
            )}
            {isJury && (
              <p className="text-[13px] text-muted">
                {selected.length
                  ? `Доступ к этому испытанию получат выбранные (${selected.length}). Остальные члены жюри конкурса его не увидят.`
                  : 'Никто не выбран — испытание снова увидят все жюри конкурса.'}
              </p>
            )}
            <div className="flex justify-end gap-2">
              <Button variant="secondary" onClick={onClose}>
                Отмена
              </Button>
              <Button onClick={() => setStep('password')}>Продолжить</Button>
            </div>
          </div>
        ) : (
          <form
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault()
              submitPassword()
            }}
          >
            <Field label="Ваш пароль" required error={error}>
              {(p) => (
                <Input
                  {...p}
                  type="password"
                  autoComplete="current-password"
                  autoFocus
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              )}
            </Field>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="secondary" disabled={loading} onClick={() => setStep('confirm')}>
                Назад
              </Button>
              <Button type="submit" loading={loading} className="bg-danger text-white hover:bg-danger/90">
                Выполнить
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  )
}
