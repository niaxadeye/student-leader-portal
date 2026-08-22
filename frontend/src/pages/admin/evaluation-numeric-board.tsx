import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { useSetNumericResult } from '@/entities/evaluation/admin-queries'
import {
  contestantLabel,
  type AdminScoreboard,
  type AdminScoreboardContestant,
} from '@/entities/evaluation/types'
import { UserAvatar } from '@/shared/ui/avatar'
import { Badge } from '@/shared/ui/badge'
import { Input } from '@/shared/ui/input'
import { EmptyState } from '@/shared/ui/states'
import { toast } from 'sonner'
import type { PendingScoreOverride } from './evaluation-score-corrections'

function fmtScore(n: number | null | undefined): string {
  if (n == null) return '—'
  if (Number.isInteger(n)) return String(n)
  return String(n)
}

type SortMode = 'name' | 'rank'

function sortRows(list: AdminScoreboardContestant[], mode: SortMode): AdminScoreboardContestant[] {
  const rows = [...list]
  if (mode === 'name') {
    rows.sort((a, b) => a.full_name.localeCompare(b.full_name, 'ru'))
    return rows
  }
  rows.sort((a, b) => {
    if (a.rank == null && b.rank == null) return a.full_name.localeCompare(b.full_name, 'ru')
    if (a.rank == null) return 1
    if (b.rank == null) return -1
    if (a.rank !== b.rank) return a.rank - b.rank
    return (b.numeric_score ?? 0) - (a.numeric_score ?? 0)
  })
  return rows
}

export function NumericResultsBoard({
  challengeId,
  board,
  canEdit,
  canOverride = false,
  onOverride,
}: {
  challengeId: string
  board: AdminScoreboard
  canEdit: boolean
  canOverride?: boolean
  onOverride?: (next: PendingScoreOverride) => void
}) {
  const [sortMode, setSortMode] = useState<SortMode>('name')
  const rows = useMemo(() => sortRows(board.contestants, sortMode), [board.contestants, sortMode])
  const max = board.max_score ?? 100
  const filled = board.contestants.filter((c) => c.numeric_score != null).length

  if (board.contestants.length === 0) {
    return <EmptyState title="Нет конкурсантов" description="Добавьте их в карточке конкурса." />
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 className="text-[18px] font-semibold text-ink">Результаты</h2>
          <p className="mt-1 text-[13px] text-muted">
            Максимум {fmtScore(max)}. Заполнено {filled} из {board.contestants.length}.
          </p>
        </div>
        <div className="flex gap-1 rounded-[12px] border border-border p-0.5">
          <SortBtn active={sortMode === 'name'} onClick={() => setSortMode('name')}>
            По имени
          </SortBtn>
          <SortBtn active={sortMode === 'rank'} onClick={() => setSortMode('rank')}>
            По месту
          </SortBtn>
        </div>
      </div>
      <ul className="flex flex-col gap-2">
        {rows.map((c) => (
          <NumericRow
            key={c.user_id}
            challengeId={challengeId}
            contestant={c}
            max={max}
            canEdit={canEdit && !canOverride}
            canOverride={canOverride}
            onOverride={onOverride}
          />
        ))}
      </ul>
    </div>
  )
}

function NumericRow({
  challengeId,
  contestant,
  max,
  canEdit,
  canOverride,
  onOverride,
}: {
  challengeId: string
  contestant: AdminScoreboardContestant
  max: number
  canEdit: boolean
  canOverride: boolean
  onOverride?: (next: PendingScoreOverride) => void
}) {
  const save = useSetNumericResult(challengeId)
  const current = contestant.numeric_score
  const [value, setValue] = useState(current == null ? '' : String(current))
  const [focused, setFocused] = useState(false)

  useEffect(() => {
    if (!focused) setValue(current == null ? '' : String(current))
  }, [current, focused])

  function persist() {
    const trimmed = value.trim()
    let next: number | null
    if (trimmed === '') {
      if (current == null) return
      next = null
    } else {
      const n = Number(trimmed.replace(',', '.'))
      if (!Number.isFinite(n) || n < 0 || n > max) {
        toast.error(`Балл — число от 0 до ${fmtScore(max)}`)
        setValue(current == null ? '' : String(current))
        return
      }
      if (current != null && Math.abs(current - n) < 1e-9) return
      next = n
    }
    if (canOverride) {
      onOverride?.({
        kind: 'NUMERIC',
        contestantUserId: contestant.user_id,
        contestantName: contestantLabel(contestant),
        min: 0,
        max,
        oldScore: current ?? null,
        newScore: next,
      })
      setValue(current == null ? '' : String(current))
      return
    }
    save.mutate(
      { contestantUserId: contestant.user_id, score: next },
      {
        onError: () => {
          toast.error('Не удалось сохранить балл')
          setValue(current == null ? '' : String(current))
        },
      },
    )
  }

  return (
    <li className="flex flex-wrap items-center gap-3 rounded-card border border-border bg-surface px-4 py-3">
      <UserAvatar name={contestant.full_name} src={contestant.avatar_url} size={28} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-[14px] font-medium text-ink">{contestantLabel(contestant)}</p>
      </div>
      {contestant.rank != null ? (
        <Badge tone="brand">{contestant.rank} место</Badge>
      ) : (
        <Badge tone="neutral">Нет балла</Badge>
      )}
      <div className="flex items-center gap-2">
        <Input
          className="w-[7.5rem]"
          type="number"
          min={0}
          max={max}
          step="0.1"
          disabled={(!canEdit && !canOverride) || save.isPending}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onFocus={() => setFocused(true)}
          onBlur={() => {
            persist()
            setFocused(false)
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              ;(e.target as HTMLInputElement).blur()
            }
          }}
          aria-label={`Балл ${contestant.full_name}`}
        />
        <span className="text-[13px] text-muted">/ {fmtScore(max)}</span>
      </div>
    </li>
  )
}

function SortBtn({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        'rounded-[10px] px-3 py-1.5 text-[13px] font-medium transition ' +
        (active ? 'bg-brand-subtle text-brand' : 'text-muted hover:text-ink')
      }
    >
      {children}
    </button>
  )
}
