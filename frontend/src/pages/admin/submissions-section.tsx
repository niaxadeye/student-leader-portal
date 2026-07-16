import { useState } from 'react'
import { Paperclip, Eye } from 'lucide-react'
import { useAdminSubmissions } from '@/entities/submission/admin-queries'
import type { AdminSubmissionRow } from '@/entities/submission/admin-api'
import { Card } from '@/shared/ui/card'
import { Badge } from '@/shared/ui/badge'
import { Button } from '@/shared/ui/button'
import { EmptyState, Skeleton, ErrorState } from '@/shared/ui/states'
import { formatDate } from '@/shared/lib/format'
import { SubmissionDetailDialog } from './submission-detail-dialog'

const statusMeta: Record<
  AdminSubmissionRow['status'],
  { label: string; tone: 'neutral' | 'brand' | 'success' | 'warning' }
> = {
  DRAFT: { label: 'Черновик', tone: 'neutral' },
  SUBMITTED: { label: 'Отправлено', tone: 'success' },
  LOCKED: { label: 'Заблокировано', tone: 'warning' },
}

export function SubmissionsSection({ challengeId }: { challengeId: string }) {
  const [openId, setOpenId] = useState<string | null>(null)
  // Дирекция видит только отправленные работы — черновики конкурсантов не показываем.
  const q = useAdminSubmissions(challengeId, 'SUBMITTED')

  return (
    <div>
      <div className="mb-3 flex items-center justify-end">
        {q.data && <p className="text-[13px] text-muted">Всего: {q.data.total}</p>}
      </div>

      {q.isLoading && <Skeleton className="h-40 w-full" />}
      {q.isError && <ErrorState onRetry={() => q.refetch()} />}
      {q.data && q.data.rows.length === 0 && (
        <EmptyState title="Ответов пока нет" description="Работы конкурсантов появятся здесь после отправки." />
      )}

      {q.data && q.data.rows.length > 0 && (
        <Card className="overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[520px] text-left text-[14px]">
              <thead className="text-[12px] uppercase tracking-wide text-muted-2">
                <tr className="border-b border-border">
                  <th className="px-4 py-2 font-medium">Конкурсант</th>
                  <th className="hidden px-4 py-2 font-medium md:table-cell">Организация</th>
                  <th className="px-4 py-2 font-medium">Статус</th>
                  <th className="hidden px-4 py-2 font-medium lg:table-cell">Ревизия</th>
                  <th className="hidden px-4 py-2 font-medium lg:table-cell">Отправлено</th>
                  <th className="px-4 py-2 font-medium">Файлы</th>
                  <th className="px-4 py-2 text-right font-medium">Действия</th>
                </tr>
              </thead>
              <tbody>
                {q.data.rows.map((r) => (
                  <tr key={r.id} className="border-b border-border last:border-0 hover:bg-surface-2/50">
                    <td className="px-4 py-2.5">
                      <div className="font-medium text-ink">{r.full_name}</div>
                      <div className="text-[12px] text-muted">{r.login}</div>
                    </td>
                    <td className="hidden px-4 py-2.5 text-muted md:table-cell">
                      {r.organization ?? '—'}
                    </td>
                    <td className="px-4 py-2.5">
                      <Badge tone={statusMeta[r.status].tone}>{statusMeta[r.status].label}</Badge>
                    </td>
                    <td className="hidden whitespace-nowrap px-4 py-2.5 text-muted lg:table-cell">
                      №{r.current_revision_number} (v{r.version})
                    </td>
                    <td className="hidden whitespace-nowrap px-4 py-2.5 text-muted lg:table-cell">
                      {r.submitted_at ? formatDate(r.submitted_at) : '—'}
                    </td>
                    <td className="px-4 py-2.5">
                      {r.file_count > 0 ? (
                        <span className="inline-flex items-center gap-1 text-muted">
                          <Paperclip className="h-3.5 w-3.5" /> {r.file_count}
                        </span>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <Button size="sm" variant="secondary" onClick={() => setOpenId(r.id)}>
                        <Eye className="h-4 w-4" /> Открыть
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <SubmissionDetailDialog
        submissionId={openId}
        open={!!openId}
        onOpenChange={(v) => !v && setOpenId(null)}
      />
    </div>
  )
}
