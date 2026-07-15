import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Trophy, Users, ChevronRight, Plus } from 'lucide-react'
import { useAuth } from '@/entities/auth/auth-context'
import { isMega, isSuper } from '@/entities/auth/roles'
import { useAdminContests } from '@/entities/contest/queries'
import { Card } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'
import { formatDate } from '@/shared/lib/format'
import { contestStatusMeta } from './contest-status'
import { CreateContestDialog } from './create-contest-dialog'
import type { AdminContest } from '@/entities/contest/types'

export function AdminContestsPage() {
  const { user } = useAuth()
  // Создавать конкурсы могут организатор (SUPER_ADMIN) и мега (§1.3).
  const canCreate = isSuper(user) || isMega(user)
  const { data: contests, isLoading, isError, refetch } = useAdminContests()
  const [createOpen, setCreateOpen] = useState(false)

  return (
    <div>
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-[28px] font-bold tracking-tight text-ink">Конкурсы</h1>
          <p className="mt-1 text-[15px] text-muted">Управление конкурсами и испытаниями.</p>
        </div>
        {canCreate && (
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" /> Новый конкурс
          </Button>
        )}
      </header>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-[92px]" />
          ))}
        </div>
      ) : isError ? (
        <ErrorState onRetry={() => refetch()} />
      ) : !contests?.length ? (
        <EmptyState
          icon={Trophy}
          title="Конкурсов пока нет"
          description={
            canCreate
              ? 'Создайте первый конкурс, чтобы начать.'
              : 'Вам не назначено ни одного конкурса. Обратитесь к суперадмину.'
          }
        />
      ) : (
        <div className="space-y-3">
          {contests.map((c) => (
            <ContestRow key={c.id} contest={c} />
          ))}
        </div>
      )}

      {canCreate && <CreateContestDialog open={createOpen} onOpenChange={setCreateOpen} />}
    </div>
  )
}

function ContestRow({ contest: c }: { contest: AdminContest }) {
  const status = contestStatusMeta[c.status]
  return (
    <Link to={`/admin/contests/${c.id}`} className="block">
      <Card className="flex items-center gap-4 p-4 transition-colors hover:border-brand/40">
        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-btn bg-brand-subtle text-brand">
          <Trophy className="h-5 w-5" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="truncate text-[16px] font-semibold text-ink">{c.name}</p>
            <Badge tone={status.tone}>{status.label}</Badge>
          </div>
          <p className="mt-0.5 text-[13px] text-muted-2">
            {c.start_at ? formatDate(c.start_at) : 'Даты не заданы'}
            {c.end_at ? ` — ${formatDate(c.end_at)}` : ''}
          </p>
        </div>
        <div className="hidden items-center gap-5 text-[13px] text-muted sm:flex">
          <span className="flex items-center gap-1.5">
            <Users className="h-4 w-4" /> {c.participants_count}
          </span>
        </div>
        <ChevronRight className="h-5 w-5 shrink-0 text-muted-2" />
      </Card>
    </Link>
  )
}
