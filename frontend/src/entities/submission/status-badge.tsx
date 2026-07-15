import { Badge } from '@/shared/ui/badge'
import type { SubmissionStatus } from '@/entities/challenge/types'

// Маппинг статуса формы (SITE.md §8) на подпись и тон бейджа.
const map: Record<SubmissionStatus | 'OVERDUE', { label: string; tone: any }> = {
  NOT_STARTED: { label: 'Не начато', tone: 'neutral' },
  DRAFT: { label: 'Черновик', tone: 'brand' },
  SUBMITTED: { label: 'Отправлено', tone: 'success' },
  LOCKED: { label: 'Заблокировано', tone: 'warning' },
  OVERDUE: { label: 'Просрочено', tone: 'danger' },
}

export function StatusBadge({ status }: { status: SubmissionStatus | 'OVERDUE' }) {
  const { label, tone } = map[status]
  return <Badge tone={tone}>{label}</Badge>
}
