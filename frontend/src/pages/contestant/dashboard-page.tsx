import { FileText, FileEdit, CheckCircle2, AlertTriangle, ArrowRight } from 'lucide-react'
import { useChallenges, useContest } from '@/entities/challenge/queries'
import { StatCard } from '@/widgets/stat-card'
import { ChallengeCard } from '@/widgets/challenge-card'
import { Card } from '@/shared/ui/card'
import { Skeleton, EmptyState } from '@/shared/ui/states'
import { timeUntil } from '@/shared/lib/format'

export function DashboardPage() {
  const { data: contest } = useContest()
  const { data: challenges, isLoading } = useChallenges()

  const nearest = challenges
    ?.filter((c) => c.deadline_at && !timeUntil(c.deadline_at).overdue)
    .sort((a, b) => (a.deadline_at! < b.deadline_at! ? -1 : 1))[0]

  // Сводка по статусам работ конкурсанта.
  const drafts = challenges?.filter((c) => c.my_submission_status === 'DRAFT').length ?? 0
  const submitted =
    challenges?.filter(
      (c) => c.my_submission_status === 'SUBMITTED' || c.my_submission_status === 'LOCKED',
    ).length ?? 0
  const overdue =
    challenges?.filter(
      (c) =>
        c.deadline_at &&
        timeUntil(c.deadline_at).overdue &&
        c.my_submission_status !== 'SUBMITTED' &&
        c.my_submission_status !== 'LOCKED',
    ).length ?? 0

  return (
    <div className="flex flex-col gap-8">
      <div>
        <p className="text-[14px] font-medium text-brand">{contest?.name ?? '…'}</p>
        <h1 className="mt-1 text-[32px] font-bold text-ink">Мой кабинет</h1>
      </div>

      {/* Предупреждение о ближайшем дедлайне */}
      {nearest && (
        <Card className="flex items-center gap-3 border-amber-200 bg-amber-50 p-4">
          <AlertTriangle className="h-5 w-5 shrink-0 text-amber-600" />
          <p className="text-[14px] text-amber-800">
            Ближайший дедлайн: <b>{nearest.title}</b> — через {timeUntil(nearest.deadline_at!).text}
          </p>
        </Card>
      )}

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <StatCard label="Испытаний" value={challenges?.length ?? '—'} icon={FileText} accent />
        <StatCard label="Черновики" value={drafts} icon={FileEdit} />
        <StatCard label="Отправлено" value={submitted} icon={CheckCircle2} />
        <StatCard label="Просрочено" value={overdue} icon={AlertTriangle} />
      </div>

      <section>
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-[22px] font-semibold text-ink">Конкурсные испытания</h2>
          <a href="/reference" className="flex items-center gap-1 text-[14px] text-brand hover:text-brand-dark">
            Справка <ArrowRight className="h-4 w-4" />
          </a>
        </div>

        {isLoading ? (
          <div className="flex flex-col gap-3">
            <Skeleton className="h-28 w-full" />
            <Skeleton className="h-28 w-full" />
          </div>
        ) : challenges && challenges.length > 0 ? (
          <div className="flex flex-col gap-3">
            {challenges.map((c) => (
              <ChallengeCard key={c.id} challenge={c} />
            ))}
          </div>
        ) : (
          <EmptyState title="Пока нет испытаний" description="Испытания появятся, когда дирекция их опубликует." />
        )}
      </section>
    </div>
  )
}
