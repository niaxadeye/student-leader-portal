import { Download, FileText, History } from 'lucide-react'
import { useAdminSubmissionDetail } from '@/entities/submission/admin-queries'
import { fileDownloadUrl } from '@/entities/submission/admin-api'
import type { AnswerValue } from '@/entities/submission/types'
import { Dialog, DialogContent } from '@/shared/ui/dialog'
import { Badge } from '@/shared/ui/badge'
import { Skeleton } from '@/shared/ui/states'
import { formatDate, formatBytes } from '@/shared/lib/format'

function renderValue(v: AnswerValue): string {
  if (v === null || v === undefined || v === '') return '—'
  if (Array.isArray(v)) return v.join(', ')
  if (typeof v === 'boolean') return v ? 'Да' : 'Нет'
  return String(v)
}

interface Props {
  submissionId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SubmissionDetailDialog({ submissionId, open, onOpenChange }: Props) {
  const q = useAdminSubmissionDetail(open ? submissionId ?? undefined : undefined)
  const d = q.data

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        title={d ? d.contestant.full_name : 'Работа конкурсанта'}
        description={d ? `${d.contestant.login}${d.contestant.organization ? ` · ${d.contestant.organization}` : ''}` : undefined}
      >
        {q.isLoading && <Skeleton className="h-64 w-full" />}
        {d && (
          <div className="flex max-h-[70vh] flex-col gap-5 overflow-y-auto pr-1">
            <div className="flex flex-wrap items-center gap-2 text-[13px] text-muted">
              <Badge tone={d.status === 'SUBMITTED' ? 'success' : d.status === 'LOCKED' ? 'warning' : 'neutral'}>
                {d.status}
              </Badge>
              <span>Ревизия №{d.current_revision_number} · версия {d.version}</span>
              {d.submitted_at && <span>· отправлено {formatDate(d.submitted_at)}</span>}
            </div>

            {/* Ответы */}
            <section>
              <h4 className="mb-2 text-[13px] font-semibold uppercase tracking-wide text-muted-2">
                Ответы
              </h4>
              <dl className="grid grid-cols-1 gap-2">
                {Object.entries(d.answers).length === 0 && (
                  <p className="text-[14px] text-muted">Ответы не заполнены.</p>
                )}
                {Object.entries(d.answers).map(([key, value]) => (
                  <div key={key} className="rounded-card bg-surface-2 px-3 py-2">
                    <dt className="text-[12px] text-muted">{key}</dt>
                    <dd className="whitespace-pre-wrap text-[14px] text-ink">{renderValue(value)}</dd>
                  </div>
                ))}
              </dl>
            </section>

            {/* Файлы */}
            {d.files.length > 0 && (
              <section>
                <h4 className="mb-2 text-[13px] font-semibold uppercase tracking-wide text-muted-2">
                  Файлы
                </h4>
                <ul className="flex flex-col gap-1.5">
                  {d.files.map((f) => (
                    <li
                      key={f.file_id}
                      className="flex items-center justify-between rounded-card border border-border px-3 py-2"
                    >
                      <span className="flex min-w-0 items-center gap-2">
                        <FileText className="h-4 w-4 shrink-0 text-muted" />
                        <span className="truncate text-[14px] text-ink">{f.original_name}</span>
                        {f.size_bytes != null && (
                          <span className="shrink-0 text-[12px] text-muted">
                            {formatBytes(f.size_bytes)}
                          </span>
                        )}
                      </span>
                      <a
                        href={fileDownloadUrl(d.id, f.file_id)}
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1 text-[13px] text-brand hover:underline"
                      >
                        <Download className="h-4 w-4" /> Скачать
                      </a>
                    </li>
                  ))}
                </ul>
              </section>
            )}

            {/* История ревизий */}
            {d.revisions.length > 0 && (
              <section>
                <h4 className="mb-2 flex items-center gap-1.5 text-[13px] font-semibold uppercase tracking-wide text-muted-2">
                  <History className="h-3.5 w-3.5" /> История ревизий
                </h4>
                <ol className="flex flex-col gap-1.5">
                  {d.revisions.map((rev) => (
                    <li
                      key={rev.id}
                      className="flex items-center justify-between rounded-card bg-surface-2 px-3 py-2 text-[13px]"
                    >
                      <span className="text-ink">
                        №{rev.revision_number} · {rev.action_type === 'SUBMIT' ? 'Отправка' : 'Обновление'}
                      </span>
                      <span className="text-muted">{formatDate(rev.created_at)}</span>
                    </li>
                  ))}
                </ol>
              </section>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
