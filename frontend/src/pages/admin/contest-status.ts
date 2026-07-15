import type { ContestStatus } from '@/entities/contest/types'
import type { BadgeProps } from '@/shared/ui/badge'

/** Статус конкурса → человекочитаемая метка и тон бейджа (SITE.md §9). */
export const contestStatusMeta: Record<
  ContestStatus,
  { label: string; tone: BadgeProps['tone'] }
> = {
  DRAFT: { label: 'Черновик', tone: 'neutral' },
  ACTIVE: { label: 'Активен', tone: 'success' },
  FINISHED: { label: 'Завершён', tone: 'brand' },
  ARCHIVED: { label: 'В архиве', tone: 'neutral' },
}
