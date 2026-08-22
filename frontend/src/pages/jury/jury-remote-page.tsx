import { Link } from 'react-router-dom'
import { ArrowLeft, Download, FileText, Paperclip } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'
import {
  useJuryBriefing,
  useJurySubmission,
  useJurySubmissions,
} from '@/entities/evaluation/jury-queries'
import { getJuryFileDownloadUrl } from '@/entities/evaluation/jury-api'
import { contestantLabel, type LiveContestant } from '@/entities/evaluation/types'
import type { AnswerValue } from '@/entities/submission/types'
import { formatBytes, formatDateTime } from '@/shared/lib/format'
import { Badge } from '@/shared/ui/badge'
import { Card, CardBody } from '@/shared/ui/card'
import { UserAvatar } from '@/shared/ui/avatar'
import { EmptyState, ErrorState, Skeleton } from '@/shared/ui/states'
import { cn } from '@/shared/lib/cn'
import { JuryScorecardSection } from './jury-scorecard'

function renderValue(v: AnswerValue): string {
  if (v === null || v === undefined || v === '') return '—'
  if (Array.isArray(v)) return v.join(', ')
  if (typeof v === 'boolean') return v ? 'Да' : 'Нет'
  return String(v)
}

export function JuryRemoteTrial({
  challengeId,
  title,
  contestants,
}: {
  challengeId: string
  title: string
  contestants: LiveContestant[]
}) {
  const briefingQ = useJuryBriefing(challengeId)
  const subsQ = useJurySubmissions(challengeId)
  const [selectedId, setSelectedId] = useState<string | null>(contestants[0]?.user_id ?? null)
  const selected = contestants.find((c) => c.user_id === selectedId) ?? null
  const byUser = useMemo(() => {
    const map = new Map<string, NonNullable<typeof subsQ.data>['rows'][number]>()
    for (const row of subsQ.data?.rows ?? []) {
      map.set(row.contestant_user_id, row)
    }
    return map
  }, [subsQ.data])
  const subRow = selectedId ? byUser.get(selectedId) : undefined

  return (
    <div className="flex flex-col gap-4">
      <Link to="/jury" className="inline-flex items-center gap-1 text-[14px] text-muted hover:text-ink">
        <ArrowLeft className="h-4 w-4" /> К испытаниям
      </Link>
      <header>
        <h1 className="text-[28px] font-bold tracking-tight text-ink">{title}</h1>
        <p className="mt-1 text-[15px] text-muted">
          Заочное оценивание. Откройте конкурсанта, посмотрите материалы и сданную работу, затем поставьте баллы.
        </p>
      </header>

      {briefingQ.isLoading ? (
        <Skeleton className="h-32 w-full" />
      ) : briefingQ.isError ? (
        <ErrorState onRetry={() => briefingQ.refetch()} />
      ) : briefingQ.data?.visible ? (
        <Card>
          <CardBody className="flex flex-col gap-3 py-5">
            <div className="flex items-center gap-2">
              <FileText className="h-5 w-5 text-muted" />
              <h2 className="text-[18px] font-semibold text-ink">Материалы испытания</h2>
            </div>
            {briefingQ.data.body_text ? (
              <p className="whitespace-pre-wrap text-[14px] text-ink">{briefingQ.data.body_text}</p>
            ) : null}
            {(briefingQ.data.files ?? []).length > 0 && (
              <ul className="flex flex-col gap-2">
                {briefingQ.data.files.map((f) => (
                  <li key={f.file_id}>
                    {f.download_url ? (
                      <a
                        href={f.download_url}
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-2 text-[14px] text-brand hover:underline"
                      >
                        <Download className="h-4 w-4" />
                        {f.original_name}
                        {f.size_bytes != null ? ` · ${formatBytes(f.size_bytes)}` : ''}
                      </a>
                    ) : (
                      <span className="text-[14px] text-muted">{f.original_name}</span>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </CardBody>
        </Card>
      ) : (
        <p className="text-[14px] text-muted">Материалы испытания пока не опубликованы.</p>
      )}

      {contestants.length === 0 ? (
        <EmptyState title="Нет конкурсантов" description="Организатор ещё не добавил участников." />
      ) : (
        <div className="grid gap-4 lg:grid-cols-[minmax(0,280px)_minmax(0,1fr)]">
          <Card>
            <CardBody className="py-4">
              <p className="mb-2 text-[13px] font-medium text-muted">Конкурсанты</p>
              {subsQ.isError && <ErrorState onRetry={() => subsQ.refetch()} />}
              <ul className="flex flex-col gap-1">
                {contestants.map((c) => {
                  const row = byUser.get(c.user_id)
                  const active = c.user_id === selectedId
                  return (
                    <li key={c.user_id}>
                      <button
                        type="button"
                        onClick={() => setSelectedId(c.user_id)}
                        className={cn(
                          'flex w-full items-center gap-2 rounded-[10px] px-2 py-2 text-left',
                          active ? 'bg-brand-subtle' : 'hover:bg-muted/10',
                        )}
                      >
                        <UserAvatar name={c.full_name} src={c.avatar_url} size={32} />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-[14px] text-ink">{c.full_name}</span>
                          <span className="block truncate text-[12px] text-muted">
                            {c.organization?.trim() || c.login}
                          </span>
                        </span>
                        {row ? (
                          <Badge tone="success">{row.file_count > 0 ? `${row.file_count}` : 'сдано'}</Badge>
                        ) : (
                          <Badge tone="neutral">нет работы</Badge>
                        )}
                      </button>
                    </li>
                  )
                })}
              </ul>
            </CardBody>
          </Card>
          <div className="flex flex-col gap-4">
            {selected ? (
              <>
                <Card>
                  <CardBody className="py-5">
                    <h2 className="text-[18px] font-semibold text-ink">{contestantLabel(selected)}</h2>
                    <p className="mt-1 text-[13px] text-muted">
                      {subRow
                        ? `Работа отправлена${subRow.submitted_at ? ` ${formatDateTime(subRow.submitted_at)}` : ''}`
                        : 'Сданной работы пока нет — оценки всё равно можно поставить.'}
                    </p>
                  </CardBody>
                </Card>
                {subRow ? <JurySubmissionView submissionId={subRow.id} /> : null}
                <JuryScorecardSection
                  challengeId={challengeId}
                  contestantUserId={selected.user_id}
                  sessionState="NOT_STARTED"
                />
              </>
            ) : (
              <EmptyState title="Выберите конкурсанта" description="Список слева." />
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function JurySubmissionView({ submissionId }: { submissionId: string }) {
  const q = useJurySubmission(submissionId)
  const [downloadingId, setDownloadingId] = useState<string | null>(null)
  if (q.isLoading) return <Skeleton className="h-40 w-full" />
  if (q.isError) return <ErrorState onRetry={() => q.refetch()} />
  const d = q.data
  if (!d) return null

  async function onDownload(fileId: string) {
    setDownloadingId(fileId)
    try {
      const url = await getJuryFileDownloadUrl(submissionId, fileId)
      window.open(url, '_blank', 'noopener,noreferrer')
    } catch {
      toast.error('Не удалось скачать файл')
    } finally {
      setDownloadingId(null)
    }
  }

  const answers = Object.entries(d.answers ?? {}).filter(([k]) => k)
  return (
    <Card>
      <CardBody className="flex flex-col gap-4 py-5">
        <h2 className="text-[18px] font-semibold text-ink">Сданная работа</h2>
        {answers.length > 0 && (
          <dl className="flex flex-col gap-2">
            {answers.map(([key, value]) => (
              <div key={key}>
                <dt className="text-[12px] uppercase tracking-wide text-muted-2">{key}</dt>
                <dd className="whitespace-pre-wrap text-[14px] text-ink">{renderValue(value)}</dd>
              </div>
            ))}
          </dl>
        )}
        {d.files.length > 0 ? (
          <ul className="flex flex-col gap-2">
            {d.files.map((f) => (
              <li key={f.file_id}>
                <button
                  type="button"
                  disabled={downloadingId === f.file_id}
                  onClick={() => void onDownload(f.file_id)}
                  className="inline-flex items-center gap-2 text-[14px] text-brand hover:underline disabled:opacity-50"
                >
                  <Paperclip className="h-4 w-4" />
                  {f.original_name}
                  {f.size_bytes != null ? ` · ${formatBytes(f.size_bytes)}` : ''}
                </button>
              </li>
            ))}
          </ul>
        ) : (
          answers.length === 0 && <p className="text-[14px] text-muted">В работе нет текстовых ответов и файлов.</p>
        )}
      </CardBody>
    </Card>
  )
}
