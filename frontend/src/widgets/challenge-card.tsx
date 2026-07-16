import { useNavigate } from 'react-router-dom'
import { Clock, ChevronRight } from 'lucide-react'
import { Card } from '@/shared/ui/card'
import { StatusBadge } from '@/entities/submission/status-badge'
import { formatDateTime, timeUntil } from '@/shared/lib/format'
import { cn } from '@/shared/lib/cn'
import type { Challenge, SubmissionStatus } from '@/entities/challenge/types'

export function ChallengeCard({ challenge }: { challenge: Challenge }) {
  const navigate = useNavigate()
  const status: SubmissionStatus = challenge.my_submission_status ?? 'NOT_STARTED'
  const deadline = challenge.deadline_at ? timeUntil(challenge.deadline_at) : null
  const isOverdue = deadline?.overdue && status !== 'SUBMITTED'

  return (
    <Card
      className="cursor-pointer p-5 transition-shadow hover:shadow-micro"
      onClick={() => navigate(`/contestant/challenges/${challenge.id}`)}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="mb-1.5 flex flex-wrap items-center gap-2">
            <h3 className="text-[18px] font-semibold text-ink">{challenge.title}</h3>
            <StatusBadge status={isOverdue ? 'OVERDUE' : status} />
          </div>
          <p className="text-[14px] text-muted">{challenge.short_description}</p>
          {deadline && (
            <div
              className={cn(
                'mt-3 inline-flex items-center gap-1.5 text-[13px]',
                deadline.overdue
                  ? 'text-danger'
                  : deadline.urgent
                    ? 'text-amber-600'
                    : 'text-muted',
              )}
            >
              <Clock className="h-4 w-4" />
              <span>
                {deadline.overdue
                  ? `Дедлайн истёк ${formatDateTime(challenge.deadline_at!)}`
                  : `Срок сдачи: ${formatDateTime(challenge.deadline_at!)}`}
              </span>
            </div>
          )}
        </div>
        <ChevronRight className="mt-1 h-5 w-5 shrink-0 text-muted-2" />
      </div>
    </Card>
  )
}
