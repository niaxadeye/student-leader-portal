import { Link } from 'react-router-dom'
import { useJuryContests } from '@/entities/evaluation/jury-queries'
import { Card, CardBody } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'

export function JuryHomePage() {
  const q = useJuryContests()
  if (q.isLoading) return <Skeleton className="h-48 w-full" />
  if (q.isError) return <ErrorState onRetry={() => q.refetch()} />
  const contests = q.data ?? []

  return (
    <div>
      <header className="mb-6">
        <h1 className="text-[28px] font-bold tracking-tight text-ink">Кабинет жюри</h1>
        <p className="mt-1 text-[15px] text-muted">
          Откройте испытание: live — следить за выступлением, заочное — оценить сданные работы.
        </p>
      </header>
      {contests.length === 0 ? (
        <EmptyState
          title="Нет назначенных конкурсов"
          description="Организатор назначит вас жюри live-испытаний или заочно — на конкретное испытание."
        />
      ) : (
        <div className="flex flex-col gap-4">
          {contests.map((c) => (
            <Card key={c.id}>
              <CardBody className="py-5">
                <h2 className="text-[18px] font-semibold text-ink">{c.name}</h2>
                {c.challenges.length === 0 ? (
                  <p className="mt-2 text-[14px] text-muted">Испытаний пока нет.</p>
                ) : (
                  <ul className="mt-3 flex flex-col gap-2">
                    {c.challenges.map((ch) => (
                      <li
                        key={ch.id}
                        className="flex items-center justify-between rounded-[10px] border border-border px-3 py-2"
                      >
                        <Link
                          to={`/jury/challenges/${ch.id}`}
                          className="text-[14px] text-ink hover:text-brand"
                        >
                          {ch.title}
                        </Link>
                        <span className="flex items-center gap-2">
                          {ch.scheme_type === 'REMOTE_CRITERIA' && <Badge tone="brand">Заочное</Badge>}
                          {ch.has_scheme ? (
                            <Badge tone="success">Схема готова</Badge>
                          ) : (
                            <Badge tone="neutral">Нет схемы</Badge>
                          )}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </CardBody>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
