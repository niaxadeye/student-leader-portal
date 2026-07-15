import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, ListChecks, ChevronRight } from 'lucide-react'
import { useAdminChallenges } from '@/entities/challenge/admin-queries'
import { Card, CardBody } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'
import { formatDate } from '@/shared/lib/format'
import { challengeStatusMeta } from './challenge-status'
import { CreateChallengeDialog } from './create-challenge-dialog'

/** Список испытаний конкурса + вход в конструктор. */
export function ChallengesSection({ contestId }: { contestId: string }) {
  const { data, isLoading, isError, refetch } = useAdminChallenges(contestId)
  const [createOpen, setCreateOpen] = useState(false)

  return (
    <section>
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-[20px] font-semibold text-ink">Испытания</h2>
        <Button size="sm" onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" /> Новое испытание
        </Button>
      </div>

      {isLoading && <Skeleton className="h-32 w-full" />}
      {isError && <ErrorState onRetry={() => refetch()} />}
      {data && data.length === 0 && (
        <EmptyState
          title="Пока нет испытаний"
          description="Создайте первое испытание и соберите форму в конструкторе."
        />
      )}

      {data && data.length > 0 && (
        <div className="flex flex-col gap-2">
          {data.map((ch) => {
            const meta = challengeStatusMeta[ch.status]
            return (
              <Link key={ch.id} to={`/admin/challenges/${ch.id}`} className="block">
                <Card className="transition hover:border-brand/40">
                  <CardBody className="flex items-center gap-4 py-3.5">
                    <ListChecks className="h-5 w-5 shrink-0 text-brand" />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <p className="truncate text-[15px] font-medium text-ink">{ch.title}</p>
                        <Badge tone={meta.tone}>{meta.label}</Badge>
                      </div>
                      <p className="mt-0.5 text-[13px] text-muted">
                        {ch.fields_count} {pluralFields(ch.fields_count)}
                        {ch.deadline_at ? ` · дедлайн ${formatDate(ch.deadline_at)}` : ''}
                      </p>
                    </div>
                    <ChevronRight className="h-4 w-4 shrink-0 text-muted-2" />
                  </CardBody>
                </Card>
              </Link>
            )
          })}
        </div>
      )}

      <CreateChallengeDialog contestId={contestId} open={createOpen} onOpenChange={setCreateOpen} />
    </section>
  )
}

function pluralFields(n: number): string {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return 'поле'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return 'поля'
  return 'полей'
}
