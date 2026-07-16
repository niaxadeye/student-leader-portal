import { ArrowLeft, Check, Loader2, Clock } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { StatusBadge } from '@/entities/submission/status-badge'
import { formatDateTime, timeUntil } from '@/shared/lib/format'
import type { Challenge, SubmissionStatus } from '@/entities/challenge/types'
import type { SaveState } from '@/features/submit-form/use-submission-form'
import { cn } from '@/shared/lib/cn'

export function ChallengeFormHeader({
  challenge,
  status,
  saveState,
  revision,
}: {
  challenge: Challenge
  status: SubmissionStatus
  saveState: SaveState
  revision: number
}) {
  const navigate = useNavigate()
  const deadline = challenge.deadline_at ? timeUntil(challenge.deadline_at) : null

  return (
    <div className="flex flex-col gap-4">
      <button
        onClick={() => navigate('/contestant')}
        className="flex w-fit items-center gap-1.5 text-[14px] text-muted hover:text-ink"
      >
        <ArrowLeft className="h-4 w-4" /> К списку испытаний
      </button>

      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-[28px] font-bold text-ink">{challenge.title}</h1>
            <StatusBadge status={status} />
          </div>
          {challenge.full_description && (
            <p className="mt-1.5 max-w-2xl text-[15px] text-muted">{challenge.full_description}</p>
          )}
        </div>
        <div className="flex flex-col items-end gap-1.5 text-[13px]">
          <SaveIndicator state={saveState} />
          {revision > 0 && <span className="text-muted">Ревизия №{revision}</span>}
          {deadline && (
            <span
              className={cn(
                'inline-flex items-center gap-1',
                deadline.overdue ? 'text-danger' : deadline.urgent ? 'text-amber-600' : 'text-muted',
              )}
            >
              <Clock className="h-3.5 w-3.5" />
              {deadline.overdue
                ? `Дедлайн истёк ${formatDateTime(challenge.deadline_at!)}`
                : `Срок сдачи: ${formatDateTime(challenge.deadline_at!)}`}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

function SaveIndicator({ state }: { state: SaveState }) {
  if (state === 'saving')
    return (
      <span className="inline-flex items-center gap-1 text-muted">
        <Loader2 className="h-3.5 w-3.5 animate-spin" /> Сохранение…
      </span>
    )
  if (state === 'saved')
    return (
      <span className="inline-flex items-center gap-1 text-success">
        <Check className="h-3.5 w-3.5" /> Черновик сохранён
      </span>
    )
  return null
}
