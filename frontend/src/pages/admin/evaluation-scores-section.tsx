import { useMemo, useState, type ReactNode } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useAdminScoreboard, useOverrideScore } from '@/entities/evaluation/admin-queries'
import { contestantLabel, formatLiveTimer, type AdminScoreboard, type AdminScoreboardContestant, type CombinedRanking, type LivesQuestionLog } from '@/entities/evaluation/types'
import { answerLabel } from '@/entities/evaluation/yes-no-buttons'
import { UserAvatar } from '@/shared/ui/avatar'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { Card, CardBody } from '@/shared/ui/card'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'
import { LivesHearts, eliminatedLabel } from '@/entities/evaluation/lives-hearts'
import { NumericResultsBoard } from './evaluation-numeric-board'
import {
  OverrideScoreInput,
  ScoreCorrectionsLog,
  ScoreOverrideDialog,
  ScoreOverrideHint,
  type PendingScoreOverride,
} from './evaluation-score-corrections'
import { toast } from 'sonner'

function fmtScore(n: number | null | undefined): string {
  if (n == null) return '—'
  return Number.isInteger(n) ? String(n) : n.toFixed(1)
}

function CombinedRankingCard({ combined }: { combined?: CombinedRanking | null }) {
  if (!combined) return null
  const rows = [...combined.rows].sort((a, b) => {
    if (a.rank == null && b.rank == null) return a.full_name.localeCompare(b.full_name, 'ru')
    if (a.rank == null) return 1
    if (b.rank == null) return -1
    if (a.rank !== b.rank) return a.rank - b.rank
    return (a.combined ?? 0) - (b.combined ?? 0)
  })
  const rankMode = combined.combine_mode === 'RANK_SUM'
  return (
    <Card className="overflow-hidden">
      <CardBody className="flex flex-col gap-3 py-5">
        <div>
          <h2 className="text-[18px] font-semibold text-ink">Итоговый рейтинг с заочным этапом</h2>
          <p className="mt-1 text-[13px] text-muted">
            {combined.remote_challenge_title} · вес основного {fmtScore(combined.main_weight)} · вес
            заочного {fmtScore(combined.remote_weight)} ·{' '}
            {rankMode
              ? 'сумма рейтингов, меньше — выше'
              : 'сумма баллов, больше — выше'}
            . Место появляется, когда есть результат в обоих этапах.
          </p>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full min-w-[560px] text-left text-[14px]">
            <thead className="text-[12px] uppercase tracking-wide text-muted-2">
              <tr className="border-b border-border">
                <th className="px-3 py-2 font-medium">Место</th>
                <th className="px-3 py-2 font-medium">Конкурсант</th>
                <th className="px-3 py-2 font-medium">Основной, место</th>
                <th className="px-3 py-2 font-medium">Заочный, место</th>
                <th className="px-3 py-2 font-medium">{rankMode ? 'Сумма мест' : 'Сумма баллов'}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.user_id} className="border-b border-border last:border-0">
                  <td className="px-3 py-2 font-medium">{r.rank ?? '—'}</td>
                  <td className="px-3 py-2">{r.full_name}</td>
                  <td className="px-3 py-2">
                    {r.main_rank ?? '—'}
                    {r.main_score != null ? ` · ${fmtScore(r.main_score)}` : ''}
                  </td>
                  <td className="px-3 py-2">
                    {r.remote_rank ?? '—'}
                    {r.remote_score != null ? ` · ${fmtScore(r.remote_score)}` : ''}
                  </td>
                  <td className="px-3 py-2">{fmtScore(r.combined)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardBody>
    </Card>
  )
}

function juryLabel(fullName: string, login: string): string {
  const name = fullName.trim()
  if (!name) return login
  const parts = name.split(/\s+/)
  if (parts.length >= 2) return `${parts[0]} ${parts[1][0]}.`
  return name
}

type SortMode = 'draw' | 'rank'

function sortContestants(list: AdminScoreboardContestant[], mode: SortMode): AdminScoreboardContestant[] {
  const rows = [...list]
  if (mode === 'draw') {
    rows.sort((a, b) => (a.draw_number ?? 9999) - (b.draw_number ?? 9999))
    return rows
  }
  rows.sort((a, b) => {
    if (a.rank == null && b.rank == null) return (a.draw_number ?? 9999) - (b.draw_number ?? 9999)
    if (a.rank == null) return 1
    if (b.rank == null) return -1
    if (a.rank !== b.rank) return a.rank - b.rank
    return (b.lives ?? b.sum ?? 0) - (a.lives ?? a.sum ?? 0)
  })
  return rows
}

export function EvaluationScoresSection({
  challengeId,
  canEdit = false,
}: {
  challengeId: string
  canEdit?: boolean
}) {
  const q = useAdminScoreboard(challengeId)
  const override = useOverrideScore(challengeId)
  const [openId, setOpenId] = useState<string | null>(null)
  const [sortMode, setSortMode] = useState<SortMode>('draw')
  const [pending, setPending] = useState<PendingScoreOverride | null>(null)
  const rows = useMemo(
    () => (q.data?.contestants ? sortContestants(q.data.contestants, sortMode) : []),
    [q.data?.contestants, sortMode],
  )

  if (q.isLoading) return <Skeleton className="h-64 w-full" />
  if (q.isError) return <ErrorState onRetry={() => q.refetch()} />
  const board = q.data
  const canOverride = !!board?.can_override

  function wrap(inner: ReactNode) {
    return (
      <div className="flex flex-col gap-6">
        <ScoreOverrideHint
          visible={canOverride && (board?.scoring_ui === 'CRITERIA' || board?.scoring_ui === 'NUMERIC')}
        />
        {inner}
        <ScoreCorrectionsLog items={board?.corrections ?? []} />
        <ScoreOverrideDialog
          pending={pending}
          loading={override.isPending}
          onClose={() => setPending(null)}
          onConfirm={(reason) => {
            if (!pending) return
            override.mutate(
              {
                kind: pending.kind,
                contestant_user_id: pending.contestantUserId,
                jury_user_id: pending.juryUserId,
                criterion_id: pending.criterionId,
                score: pending.newScore,
                reason,
              },
              {
                onSuccess: () => {
                  toast.success('Балл изменён')
                  setPending(null)
                },
                onError: () => toast.error('Не удалось сохранить правку. Проверьте причину и диапазон балла.'),
              },
            )
          }}
        />
      </div>
    )
  }

  if (!board?.configured) {
    return wrap(
      <EmptyState
        title="Схема не настроена"
        description="Сначала задайте тип оценивания на вкладке «Оценивание»."
      />,
    )
  }
  if (board.scoring_ui === 'LIVES') {
    return wrap(
      <>
        <LivesScoreboard board={board} />
        <CombinedRankingCard combined={board.combined} />
      </>,
    )
  }
  if (board.scoring_ui === 'NUMERIC') {
    return wrap(
      <>
        <NumericResultsBoard
          challengeId={challengeId}
          board={board}
          canEdit={canEdit}
          canOverride={canOverride}
          onOverride={(next) => setPending(next)}
        />
        <CombinedRankingCard combined={board.combined} />
      </>,
    )
  }
  if (board.scoring_ui !== 'CRITERIA') {
    return wrap(
      <>
        <EmptyState
          title="Просмотр баллов недоступен"
          description="Сводная таблица пока работает только для типов «Критерии», «2 к 1» и «Числовой результат»."
        />
        <CombinedRankingCard combined={board.combined} />
      </>,
    )
  }
  if (board.contestants.length === 0) {
    return wrap(
      <>
        <EmptyState title="Нет конкурсантов" description="Добавьте их в карточке конкурса." />
        <CombinedRankingCard combined={board.combined} />
      </>,
    )
  }
  if (board.jury.length === 0) {
    return wrap(
      <>
        <EmptyState
          title="Оценок пока нет"
          description={
            board.scheme_type === 'REMOTE_CRITERIA'
              ? 'Назначьте заочное жюри на вкладке «Оценивание» — их баллы появятся здесь.'
              : 'Назначьте членов жюри на конкурс — их баллы появятся здесь.'
          }
        />
        <CombinedRankingCard combined={board.combined} />
      </>,
    )
  }

  const critCount = board.criteria.length

  return wrap(
    <>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-[13px] text-muted">
          Место считается по сумме баллов жюри: равные суммы делят место, следующие номера пропускаются.
        </p>
        <div className="flex gap-2">
          <Button
            size="sm"
            variant={sortMode === 'draw' ? 'primary' : 'secondary'}
            onClick={() => setSortMode('draw')}
          >
            По жеребьёвке
          </Button>
          <Button
            size="sm"
            variant={sortMode === 'rank' ? 'primary' : 'secondary'}
            onClick={() => setSortMode('rank')}
          >
            По рейтингу
          </Button>
        </div>
      </div>
      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-left text-[14px]">
            <thead className="text-[12px] uppercase tracking-wide text-muted-2">
              <tr className="border-b border-border">
                <th className="sticky left-0 bg-surface px-4 py-2 font-medium">Конкурсант</th>
                <th className="px-3 py-2 font-medium">Выступление</th>
                {board.jury.map((j) => (
                  <th key={j.user_id} className="px-3 py-2 font-medium" title={`${j.full_name} · ${j.login}`}>
                    {juryLabel(j.full_name, j.login)}
                  </th>
                ))}
                <th className="px-4 py-2 font-medium">Сумма</th>
                <th className="px-4 py-2 font-medium">Место</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((c) => {
                const open = openId === c.user_id
                const onStage = c.user_id === board.current_contestant_user_id
                return (
                  <ScoreboardRow
                    key={c.user_id}
                    open={open}
                    onToggle={() => setOpenId(open ? null : c.user_id)}
                    onStage={onStage}
                    contestantId={c.user_id}
                    name={contestantLabel(c)}
                    avatarUrl={c.avatar_url}
                    speechSeconds={c.speech_duration_seconds}
                    sheets={c.sheets}
                    sum={c.sum}
                    rank={c.rank}
                    critCount={critCount}
                    criteria={board.criteria}
                    jury={board.jury}
                    canOverride={canOverride}
                    onOverride={setPending}
                  />
                )
              })}
            </tbody>
          </table>
        </div>
      </Card>
      <CombinedRankingCard combined={board.combined} />
    </>,
  )
}

function ScoreboardRow({
  open,
  onToggle,
  onStage,
  contestantId,
  name,
  avatarUrl,
  speechSeconds,
  sheets,
  sum,
  rank,
  critCount,
  criteria,
  jury,
  canOverride,
  onOverride,
}: {
  open: boolean
  onToggle: () => void
  onStage: boolean
  contestantId: string
  name: string
  avatarUrl: string | null
  speechSeconds: number | null
  sheets: {
    jury_user_id: string
    filled: number
    total: number | null
    values: { criterion_id: string; score: number | null }[]
  }[]
  sum: number | null
  rank: number | null
  critCount: number
  criteria: { id: string; title: string; weight: number; min_score: number; max_score: number }[]
  jury: { user_id: string; full_name: string; login: string }[]
  canOverride: boolean
  onOverride: (next: PendingScoreOverride) => void
}) {
  return (
    <>
      <tr className="border-b border-border">
        <td className="sticky left-0 bg-surface px-4 py-2">
          <button type="button" onClick={onToggle} className="flex items-center gap-2 text-left">
            {open ? <ChevronDown className="h-4 w-4 shrink-0 text-muted" /> : <ChevronRight className="h-4 w-4 shrink-0 text-muted" />}
            <UserAvatar src={avatarUrl} name={name} size={28} />
            <span className="font-medium text-ink">{name}</span>
            {onStage ? <Badge tone="success">на сцене</Badge> : null}
          </button>
        </td>
        <td className="px-3 py-2 font-mono tabular-nums text-ink">
          {speechSeconds != null ? formatLiveTimer(speechSeconds) : '—'}
        </td>
        {sheets.map((sh) => (
          <td key={sh.jury_user_id} className="px-3 py-2 tabular-nums">
            {sh.filled === 0 ? (
              <span className="text-muted-2">—</span>
            ) : (
              <span>
                {fmtScore(sh.total)}
                <span className="ml-1 text-[12px] text-muted-2">
                  {sh.filled}/{critCount}
                </span>
              </span>
            )}
          </td>
        ))}
        <td className="px-4 py-2 font-medium tabular-nums">{fmtScore(sum)}</td>
        <td className="px-4 py-2 font-semibold tabular-nums">{rank ?? '—'}</td>
      </tr>
      {open && (
        <tr className="border-b border-border bg-muted/5">
          <td colSpan={sheets.length + 4} className="px-4 py-3">
            <table className="w-full min-w-[480px] text-left text-[13px]">
              <thead className="text-muted-2">
                <tr>
                  <th className="py-1 pr-3 font-medium">Критерий</th>
                  {jury.map((j) => (
                    <th key={j.user_id} className="px-2 py-1 font-medium">
                      {juryLabel(j.full_name, j.login)}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {criteria.map((crit, ci) => (
                  <tr key={crit.id}>
                    <td className="py-1 pr-3 text-ink">
                      {crit.title}
                      {crit.weight !== 1 ? <span className="text-muted-2"> · вес {crit.weight}</span> : null}
                    </td>
                    {sheets.map((sh, ji) => (
                      <td key={sh.jury_user_id} className="px-2 py-1">
                        {canOverride ? (
                          <OverrideScoreInput
                            value={sh.values[ci]?.score ?? null}
                            min={crit.min_score}
                            max={crit.max_score}
                            ariaLabel={`${crit.title}, ${juryLabel(jury[ji]?.full_name ?? '', jury[ji]?.login ?? '')}`}
                            onCommit={(next) =>
                              onOverride({
                                kind: 'CRITERION',
                                contestantUserId: contestantId,
                                contestantName: name,
                                juryUserId: sh.jury_user_id,
                                juryName: juryLabel(jury[ji]?.full_name ?? '', jury[ji]?.login ?? ''),
                                criterionId: crit.id,
                                criterionTitle: crit.title,
                                min: crit.min_score,
                                max: crit.max_score,
                                oldScore: sh.values[ci]?.score ?? null,
                                newScore: next,
                              })
                            }
                          />
                        ) : (
                          <span className="tabular-nums">{fmtScore(sh.values[ci]?.score)}</span>
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </td>
        </tr>
      )}
    </>
  )
}

function LivesScoreboard({ board }: { board: AdminScoreboard }) {
  const [sortMode, setSortMode] = useState<SortMode>('rank')
  const [juryId, setJuryId] = useState(board.operator_user_id ?? board.jury[0]?.user_id ?? '')
  const rows = useMemo(() => sortContestants(board.contestants, sortMode), [board.contestants, sortMode])
  const starting = board.starting_lives ?? 3
  const log = (board.life_logs ?? []).find((l) => l.jury_user_id === juryId)
  const jury = board.jury.find((j) => j.user_id === juryId)

  if (board.contestants.length === 0) {
    return <EmptyState title="Нет конкурсантов" description="Добавьте их в карточке конкурса." />
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-[13px] text-muted">
          Место с конца: выбывшие получают последний свободный номер. Несколько выбывших на одном вопросе делят место
          (12, 12 → следующий 10). Оставшиеся сравниваются по жизням. Официальный рейтинг — лог ответственного жюри.
        </p>
        <div className="flex gap-2">
          <Button size="sm" variant={sortMode === 'draw' ? 'primary' : 'secondary'} onClick={() => setSortMode('draw')}>
            По жеребьёвке
          </Button>
          <Button size="sm" variant={sortMode === 'rank' ? 'primary' : 'secondary'} onClick={() => setSortMode('rank')}>
            По рейтингу
          </Button>
        </div>
      </div>

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[520px] text-left text-[14px]">
            <thead className="text-[12px] uppercase tracking-wide text-muted-2">
              <tr className="border-b border-border">
                <th className="px-4 py-2 font-medium">Конкурсант</th>
                <th className="px-3 py-2 font-medium">Жизни</th>
                <th className="px-3 py-2 font-medium">Статус</th>
                <th className="px-4 py-2 font-medium">Место</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((c) => (
                <tr key={c.user_id} className="border-b border-border">
                  <td className="px-4 py-2">
                    <span className="flex items-center gap-2">
                      <UserAvatar src={c.avatar_url} name={c.full_name} size={28} />
                      <span className="font-medium text-ink">{contestantLabel(c)}</span>
                    </span>
                  </td>
                  <td className="px-3 py-2">
                    <LivesHearts lives={c.eliminated ? 0 : (c.lives ?? starting)} starting={starting} />
                  </td>
                  <td className="px-3 py-2">
                    {c.eliminated ? (
                      <Badge tone="danger">{eliminatedLabel(c.eliminated_question)}</Badge>
                    ) : (
                      <Badge tone="success">в игре</Badge>
                    )}
                  </td>
                  <td className="px-4 py-2 font-semibold tabular-nums">{c.rank ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      <div>
        <h3 className="text-[16px] font-semibold text-ink">Логи жюри</h3>
        <p className="mt-1 text-[13px] text-muted">
          Официальный лог отмечен. Вопрос {board.current_question_number ?? 1}. Жизнь снимается, если ответ ответственного жюри не совпал с ключом.
        </p>
        {board.jury.length === 0 ? (
          <p className="mt-3 text-[14px] text-muted">Назначьте жюри на конкурс, чтобы увидеть логи.</p>
        ) : (
          <>
            <div className="mt-3 flex flex-wrap gap-2">
              {board.jury.map((j) => (
                <Button
                  key={j.user_id}
                  size="sm"
                  variant={juryId === j.user_id ? 'primary' : 'secondary'}
                  onClick={() => setJuryId(j.user_id)}
                >
                  {juryLabel(j.full_name, j.login)}
                  {j.user_id === board.operator_user_id ? ' · офиц.' : ''}
                </Button>
              ))}
            </div>
            <Card className="mt-3">
              <CardBody className="py-5">
                <p className="text-[14px] font-medium text-ink">
                  {jury ? jury.full_name.trim() || jury.login : 'Жюри'}
                  {log?.official ? <span className="ml-2 text-[12px] font-normal text-brand">официальный</span> : null}
                </p>
                {!log || log.questions.length === 0 ? (
                  <p className="mt-2 text-[13px] text-muted">Пока нет отметок.</p>
                ) : (
                  <ol className="mt-3 flex flex-col gap-2">
                    {log.questions.map((item) => (
                      <ScoreQuestionLogRow key={item.question_number} item={item} />
                    ))}
                  </ol>
                )}
              </CardBody>
            </Card>
          </>
        )}
      </div>
    </div>
  )
}

function ScoreQuestionLogRow({ item }: { item: LivesQuestionLog }) {
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
        <span className="ml-2 text-[13px] font-normal text-muted">правильный: {answerLabel(item.correct_answer)}</span>
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

