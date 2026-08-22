import { useEffect, useMemo, useState } from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { useJuryScorecard, usePutJuryScore } from '@/entities/evaluation/jury-queries'
import { contestantLabel, type JuryScorecardCriterion, type LiveState } from '@/entities/evaluation/types'
import { ApiRequestError } from '@/shared/api/client'
import { Card, CardBody } from '@/shared/ui/card'
import { Input } from '@/shared/ui/input'
import { ErrorState, Skeleton } from '@/shared/ui/states'
import { cn } from '@/shared/lib/cn'
import { toast } from 'sonner'

function integerOptions(min: number, max: number): number[] {
  if (!Number.isFinite(min) || !Number.isFinite(max) || max < min) return []
  const a = Math.round(min)
  const b = Math.round(max)
  if (Math.abs(a - min) > 1e-9 || Math.abs(b - max) > 1e-9) return []
  if (b - a > 20) return []
  return Array.from({ length: b - a + 1 }, (_, i) => a + i)
}

function fmtTotal(n: number): string {
  return Number.isInteger(n) ? String(n) : n.toFixed(1)
}

export function JuryScorecardSection({
  challengeId,
  contestantUserId,
  sessionState,
}: {
  challengeId: string
  contestantUserId: string | null
  sessionState: LiveState
}) {
  const q = useJuryScorecard(challengeId, contestantUserId, sessionState)
  if (q.isLoading) return <Skeleton className="h-64 w-full" />
  if (q.isError) return <ErrorState onRetry={() => q.refetch()} />
  const card = q.data
  if (!card?.configured) {
    return (
      <Card>
        <CardBody className="py-6">
          <h2 className="text-[18px] font-semibold text-ink">Оценивание</h2>
          <p className="mt-2 text-[14px] text-muted">Организатор ещё не настроил схему оценивания.</p>
        </CardBody>
      </Card>
    )
  }
  if (card.scoring_ui !== 'CRITERIA') {
    return (
      <Card>
        <CardBody className="py-6">
          <h2 className="text-[18px] font-semibold text-ink">Оценивание</h2>
          <p className="mt-2 text-[14px] text-muted">
            Поля баллов для этого типа схемы появятся позже. Сейчас доступно только оценивание по критериям.
          </p>
        </CardBody>
      </Card>
    )
  }
  if (card.criteria.length === 0) {
    return (
      <Card>
        <CardBody className="py-6">
          <h2 className="text-[18px] font-semibold text-ink">Оценивание</h2>
          <p className="mt-2 text-[14px] text-muted">В схеме пока нет критериев.</p>
        </CardBody>
      </Card>
    )
  }
  return <CriteriaForm challengeId={challengeId} card={card} />
}

function CriteriaForm({
  challengeId,
  card,
}: {
  challengeId: string
  card: NonNullable<ReturnType<typeof useJuryScorecard>['data']>
}) {
  const save = usePutJuryScore(challengeId)
  const [local, setLocal] = useState<Record<string, number>>({})
  const [pendingId, setPendingId] = useState<string | null>(null)

  useEffect(() => {
    setLocal({})
  }, [card.performance_id])

  const filled = useMemo(() => {
    return card.criteria.filter((c) => (local[c.id] ?? c.score) != null).length
  }, [card.criteria, local])

  const canScore = card.editable && !!card.performance_id

  function pick(criterion: JuryScorecardCriterion, score: number) {
    if (!canScore || !card.performance_id) return
    setLocal((prev) => ({ ...prev, [criterion.id]: score }))
    setPendingId(criterion.id)
    save.mutate(
      {
        performance_id: card.performance_id,
        criterion_id: criterion.id,
        score,
        mutation_id: crypto.randomUUID(),
      },
      {
        onSettled: () => setPendingId((id) => (id === criterion.id ? null : id)),
        onError: (err) => {
          setLocal((prev) => {
            const next = { ...prev }
            delete next[criterion.id]
            return next
          })
          if (err instanceof ApiRequestError && err.code === 'EVALUATION_SCORING_CLOSED') {
            toast.error('Оценивание закрыто')
            return
          }
          toast.error('Не удалось сохранить оценку')
        },
      },
    )
  }

  return (
    <Card>
      <CardBody className="flex flex-col gap-4 py-6">
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div>
            <h2 className="text-[18px] font-semibold text-ink">Оценки</h2>
            <p className="mt-1 text-[13px] text-muted">
              {card.contestant
                ? contestantLabel(card.contestant)
                : 'Конкурсант ещё не выбран — поля появятся после выбора.'}
            </p>
          </div>
          <p className="text-[13px] text-muted">
            {filled} из {card.criteria.length}
            {card.total != null ? ` · сумма ${fmtTotal(card.total)}` : ''}
          </p>
        </div>
        {!card.editable && (
          <p className="text-[13px] text-muted">Испытание завершено, оценки больше нельзя менять.</p>
        )}
        <ul className="flex flex-col gap-3">
          {card.criteria.map((c, i) => (
            <CriterionScore
              key={c.id}
              index={i + 1}
              criterion={c}
              value={local[c.id] ?? c.score}
              disabled={!canScore}
              saving={pendingId === c.id}
              onPick={(score) => pick(c, score)}
            />
          ))}
        </ul>
      </CardBody>
    </Card>
  )
}

function CriterionScore({
  index,
  criterion,
  value,
  disabled,
  saving,
  onPick,
}: {
  index: number
  criterion: JuryScorecardCriterion
  value: number | null
  disabled: boolean
  saving: boolean
  onPick: (score: number) => void
}) {
  const [open, setOpen] = useState(false)
  const options = integerOptions(criterion.min_score, criterion.max_score)
  const hasDetails = !!(criterion.description || criterion.bands.length)
  const [draft, setDraft] = useState(value != null ? String(value) : '')
  useEffect(() => {
    setDraft(value != null ? String(value) : '')
  }, [value])

  return (
        <li className="rounded-card border border-border bg-surface px-3 py-3 sm:px-4">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="text-[15px] font-medium text-ink">
            {index}. {criterion.title}
          </p>
          <p className="text-[12px] text-muted-2">
            {criterion.min_score}–{criterion.max_score}
            {criterion.weight !== 1 ? ` · вес ${criterion.weight}` : ''}
            {criterion.is_required ? '' : ' · необязательный'}
          </p>
        </div>
        {hasDetails && (
          <button
            type="button"
            className="shrink-0 rounded-btn p-1 text-muted hover:bg-muted/10 hover:text-ink"
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
          >
            {open ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </button>
        )}
      </div>
      {open && hasDetails && (
        <div className="mt-2 flex flex-col gap-1 text-[13px] text-muted">
          {criterion.description ? <p>{criterion.description}</p> : null}
          {criterion.bands.map((b) => (
            <p key={`${b.min_score}-${b.max_score}-${b.description}`}>
              {b.min_score}–{b.max_score}: {b.description}
            </p>
          ))}
        </div>
      )}
      <div className={cn('mt-3', saving && 'opacity-70')}>
        {options.length > 0 ? (
          <div className="grid grid-cols-5 gap-2 sm:grid-cols-10">
            {options.map((n) => {
              const active = value === n
              return (
                <button
                  key={n}
                  type="button"
                  disabled={disabled}
                  aria-pressed={active}
                  onClick={() => onPick(n)}
                  className={cn(
                    'h-12 rounded-[10px] text-[16px] font-semibold transition-colors disabled:opacity-50',
                    active
                      ? 'bg-brand text-white'
                      : 'border border-border bg-surface text-ink hover:bg-brand-subtle',
                  )}
                >
                  {n}
                </button>
              )
            })}
          </div>
        ) : (
          <label className="flex items-center gap-2 text-[13px] text-muted">
            Балл
            <Input
              type="number"
              min={criterion.min_score}
              max={criterion.max_score}
              step="any"
              className="h-11 w-28"
              disabled={disabled}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onBlur={() => {
                const n = Number(draft)
                if (Number.isFinite(n)) onPick(n)
              }}
            />
          </label>
        )}
      </div>
    </li>
  )
}
