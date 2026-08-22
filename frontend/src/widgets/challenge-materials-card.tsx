import { useNavigate } from 'react-router-dom'
import { ChevronRight, Clock, FileText } from 'lucide-react'
import { formatDateTime } from '@/shared/lib/format'
import type { Challenge } from '@/entities/challenge/types'
import { Card } from '@/shared/ui/card'

export function ChallengeMaterialsCard({ challenge }: { challenge: Challenge }) {
  const navigate = useNavigate()
  const briefing = challenge.briefing
  if (!briefing) return null

  const scheduled = !briefing.visible && briefing.scheduled && briefing.publish_at

  return (
    <Card
      className="cursor-pointer p-5 transition-shadow hover:shadow-micro"
      onClick={() => navigate(`/contestant/challenges/${challenge.id}`)}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-[18px] font-semibold text-ink">{challenge.title}</h3>
            {briefing.visible ? (
              <span className="inline-flex items-center gap-1 rounded-full bg-brand-subtle px-2 py-0.5 text-[12px] font-medium text-brand">
                <FileText className="h-3 w-3" /> Доступны
              </span>
            ) : (
              <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[12px] font-medium text-amber-700">
                Скоро
              </span>
            )}
          </div>
          {scheduled && (
            <p className="mt-3 inline-flex items-center gap-1.5 text-[13px] text-amber-700">
              <Clock className="h-4 w-4" />
              Появятся {formatDateTime(briefing.publish_at!)}
            </p>
          )}
          {briefing.visible && briefing.personalized && (
            <p className="mt-3 text-[13px] text-muted">Персональная выдача</p>
          )}
        </div>
        <ChevronRight className="mt-1 h-5 w-5 shrink-0 text-muted-2" />
      </div>
    </Card>
  )
}
