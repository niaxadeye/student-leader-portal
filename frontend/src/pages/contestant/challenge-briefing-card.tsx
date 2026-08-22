import { Clock, Download, FileText } from 'lucide-react'
import { formatBytes, formatDateTime } from '@/shared/lib/format'
import type { ChallengeBriefing } from '@/entities/challenge/types'
import { Card, CardBody } from '@/shared/ui/card'

export function ChallengeBriefingCard({ briefing }: { briefing?: ChallengeBriefing | null }) {
  if (!briefing) return null
  if (!briefing.visible && !briefing.scheduled) return null

  if (!briefing.visible && briefing.scheduled && briefing.publish_at) {
    return (
      <Card className="border-amber-200 bg-amber-50">
        <CardBody className="flex items-start gap-3 py-5">
          <Clock className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
          <div>
            <h2 className="text-[16px] font-semibold text-ink">Материалы испытания</h2>
            <p className="mt-1 text-[14px] text-amber-800">
              Появятся {formatDateTime(briefing.publish_at)}
            </p>
          </div>
        </CardBody>
      </Card>
    )
  }

  if (!briefing.visible) return null
  const text = briefing.body_text?.trim() ?? ''
  const files = briefing.files ?? []
  if (!text && files.length === 0) return null

  return (
    <Card>
      <CardBody className="flex flex-col gap-4 py-5">
        <div>
          <h2 className="text-[16px] font-semibold text-ink">Материалы испытания</h2>
          {briefing.personalized && (
            <p className="mt-1 text-[13px] text-muted">Персональная выдача</p>
          )}
        </div>
        {text && (
          <p className="whitespace-pre-wrap text-[15px] leading-6 text-ink">{text}</p>
        )}
        {files.length > 0 && (
          <ul className="flex flex-col gap-2">
            {files.map((f) => (
              <li key={f.file_id}>
                {f.download_url ? (
                  <a
                    href={f.download_url}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center gap-2 rounded-[10px] border border-border px-3 py-2 text-[14px] text-ink hover:border-brand/40"
                  >
                    <FileText className="h-4 w-4 shrink-0 text-brand" />
                    <span className="min-w-0 flex-1 truncate">{f.original_name}</span>
                    {f.size_bytes != null && (
                      <span className="text-[12px] text-muted">{formatBytes(f.size_bytes)}</span>
                    )}
                    <Download className="h-4 w-4 shrink-0 text-muted-2" />
                  </a>
                ) : (
                  <span className="flex items-center gap-2 text-[14px] text-muted">
                    <FileText className="h-4 w-4" />
                    {f.original_name}
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
      </CardBody>
    </Card>
  )
}
