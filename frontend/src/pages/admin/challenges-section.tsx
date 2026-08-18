import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, ListChecks, ChevronRight, Copy } from 'lucide-react'
import { useAdminChallenges, useDuplicateChallenge } from '@/entities/challenge/admin-queries'
import { Card, CardBody } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'
import { formatDate } from '@/shared/lib/format'
import { toast } from 'sonner'
import { challengeStatusMeta } from './challenge-status'
import { CreateChallengeDialog } from './create-challenge-dialog'
import type { AdminChallenge } from '@/entities/challenge/admin-types'

/** Список испытаний конкурса + вход в конструктор. canEdit скрывает создание (VIEW-режим). */
export function ChallengesSection({ contestId, canEdit }: { contestId: string; canEdit: boolean }) {
  const { data, isLoading, isError, refetch } = useAdminChallenges(contestId)
  const duplicate = useDuplicateChallenge(contestId)
  const [createOpen, setCreateOpen] = useState(false)
  const [copyingId, setCopyingId] = useState<string>()

  function onDuplicate(ch: AdminChallenge) {
    setCopyingId(ch.id)
    duplicate.mutate(ch.id, {
      onSuccess: (copy) => toast.success(`Создана копия: ${copy.title}`),
      onError: () => toast.error('Не удалось дублировать испытание'),
      onSettled: () => setCopyingId(undefined),
    })
  }

  return (
    <section>
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-[20px] font-semibold text-ink">Испытания</h2>
        {canEdit && (
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            <Plus className="h-4 w-4" /> Новое испытание
          </Button>
        )}
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
              <Card key={ch.id} className="transition hover:border-brand/40">
                <CardBody className="flex items-center gap-2 py-3.5 pr-2">
                  <Link
                    to={`/admin/challenges/${ch.id}`}
                    className="flex min-w-0 flex-1 items-center gap-4"
                  >
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
                  </Link>
                  {canEdit && (
                    <button
                      type="button"
                      title="Дублировать"
                      aria-label={`Дублировать «${ch.title}»`}
                      disabled={duplicate.isPending && copyingId === ch.id}
                      onClick={() => onDuplicate(ch)}
                      className="shrink-0 rounded-md p-2 text-muted transition hover:bg-muted/10 hover:text-ink disabled:opacity-40"
                    >
                      <Copy className="h-4 w-4" />
                    </button>
                  )}
                </CardBody>
              </Card>
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
