import { useEffect, useState } from 'react'
import { Field } from '@/shared/ui/field'
import { Input } from '@/shared/ui/input'
import { Button } from '@/shared/ui/button'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { YesNoButtons, type YesNoAnswer } from '@/entities/evaluation/yes-no-buttons'
import type { LiveSnapshot } from '@/entities/evaluation/types'

const MIN_QUESTIONS = 1
const MAX_QUESTIONS = 50

function clampCount(n: number): number {
  if (!Number.isFinite(n)) return MIN_QUESTIONS
  return Math.min(MAX_QUESTIONS, Math.max(MIN_QUESTIONS, Math.round(n)))
}

function answersFromSnap(snap: LiveSnapshot, count: number): Record<number, YesNoAnswer | ''> {
  const next: Record<number, YesNoAnswer | ''> = {}
  for (let i = 1; i <= count; i++) next[i] = ''
  for (const item of snap.lives?.questions ?? []) {
    if (item.correct_answer === 'YES' || item.correct_answer === 'NO') {
      next[item.question_number] = item.correct_answer
    }
  }
  for (const key of snap.question_keys ?? []) {
    if (key.correct_answer === 'YES' || key.correct_answer === 'NO') {
      next[key.question_number] = key.correct_answer
    }
  }
  return next
}

function resizeAnswers(
  prev: Record<number, YesNoAnswer | ''>,
  count: number,
): Record<number, YesNoAnswer | ''> {
  const next: Record<number, YesNoAnswer | ''> = {}
  for (let i = 1; i <= count; i++) next[i] = prev[i] ?? ''
  return next
}

export function QuestionPlanDialog({
  open,
  snap,
  pending,
  onClose,
  onSave,
}: {
  open: boolean
  snap: LiveSnapshot
  pending: boolean
  onClose: () => void
  onSave: (questionCount: number, answers: { question_number: number; correct_answer: YesNoAnswer }[]) => void
}) {
  const [count, setCount] = useState(MIN_QUESTIONS)
  const [countText, setCountText] = useState(String(MIN_QUESTIONS))
  const [answers, setAnswers] = useState<Record<number, YesNoAnswer | ''>>({})
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    const planned = snap.question_count && snap.question_count > 0 ? snap.question_count : MIN_QUESTIONS
    setCount(planned)
    setCountText(String(planned))
    setAnswers(answersFromSnap(snap, planned))
    setError('')
    // Снимок Live обновляется каждые 2с — не сбрасывать ввод, пока окно открыто.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  function applyCount(n: number) {
    const next = clampCount(n)
    setCount(next)
    setAnswers((prev) => resizeAnswers(prev, next))
    setError('')
    return next
  }

  function save() {
    const items: { question_number: number; correct_answer: YesNoAnswer }[] = []
    for (let i = 1; i <= count; i++) {
      const answer = answers[i]
      if (answer !== 'YES' && answer !== 'NO') {
        setError(`Задайте ответ Да или Нет для вопроса ${i}`)
        return
      }
      items.push({ question_number: i, correct_answer: answer })
    }
    onSave(count, items)
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent
        className="max-w-lg"
        title="Вопросы 2 к 1"
        description="Задайте количество вопросов и правильный ответ на каждый. Жюри будет фиксировать ответы конкурсантов."
      >
        <div className="flex max-h-[min(70vh,32rem)] flex-col gap-4">
          <Field
            label="Количество вопросов"
            helpText="От 1 до 50. На Live можно переключать только эти вопросы."
          >
            {(props) => (
              <Input
                {...props}
                type="number"
                min={MIN_QUESTIONS}
                max={MAX_QUESTIONS}
                value={countText}
                disabled={pending}
                onChange={(e) => {
                  const raw = e.target.value
                  setCountText(raw)
                  const parsed = Number.parseInt(raw, 10)
                  if (Number.isFinite(parsed)) applyCount(parsed)
                }}
                onBlur={() => {
                  const next = applyCount(Number.parseInt(countText, 10))
                  setCountText(String(next))
                }}
              />
            )}
          </Field>
          <div className="min-h-0 flex-1 overflow-y-auto pr-1">
            <ul className="flex flex-col gap-2">
              {Array.from({ length: count }, (_, i) => i + 1).map((n) => (
                <li
                  key={n}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-[10px] border border-border px-3 py-2"
                >
                  <p className="text-[14px] font-medium text-ink">Вопрос {n}</p>
                  <YesNoButtons
                    value={answers[n] || null}
                    disabled={pending}
                    onSelect={(answer) => {
                      setAnswers((prev) => ({ ...prev, [n]: answer }))
                      setError('')
                    }}
                  />
                </li>
              ))}
            </ul>
          </div>
          {error ? <p className="text-[13px] text-danger">{error}</p> : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" disabled={pending} onClick={onClose}>
              Отмена
            </Button>
            <Button type="button" disabled={pending} onClick={save}>
              Сохранить
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
