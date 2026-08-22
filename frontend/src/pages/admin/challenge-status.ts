import type { ChallengeStatus } from '@/entities/challenge/admin-types'
import {
  evaluationTypeLabels,
  type EvaluationType,
} from '@/entities/evaluation/types'
import type { BadgeProps } from '@/shared/ui/badge'

/** Статус приёма файлов и ТЗ (не всего испытания). */
export const challengeStatusMeta: Record<
  ChallengeStatus,
  { label: string; tone: BadgeProps['tone'] }
> = {
  DRAFT: { label: 'Черновик', tone: 'neutral' },
  PUBLISHED: { label: 'Опубликовано', tone: 'success' },
  CLOSED: { label: 'Приём закрыт', tone: 'warning' },
  ARCHIVED: { label: 'В архиве', tone: 'neutral' },
}

/** Бейдж приёма: выключатель важнее статуса публикации. */
export function intakeBadge(ch: {
  status: ChallengeStatus
  accepts_submissions?: boolean
}): { label: string; tone: BadgeProps['tone'] } {
  if (ch.accepts_submissions === false) return { label: 'Приём выключен', tone: 'neutral' }
  return challengeStatusMeta[ch.status]
}

/** Строка для списка испытаний. */
export function intakeStatusLine(ch: {
  status: ChallengeStatus
  accepts_submissions?: boolean
}): string {
  if (ch.accepts_submissions === false) return 'Без приёма файлов'
  switch (ch.status) {
    case 'PUBLISHED':
      return 'Приём опубликован'
    case 'CLOSED':
      return 'Приём закрыт'
    case 'ARCHIVED':
      return 'Приём в архиве'
    default:
      return 'Приём: черновик'
  }
}

/** Тип оценивания для карточки испытания. */
export function schemeTypeLabel(type: string | null | undefined): string {
  if (!type) return 'Тип оценивания не задан'
  return evaluationTypeLabels[type as EvaluationType] ?? type
}

/** Проведено / идёт / ещё нет — по состоянию live-сессии. */
export function liveConductedMeta(state: string | null | undefined): {
  label: string
  tone: BadgeProps['tone']
} {
  if (!state || state === 'NOT_STARTED') return { label: 'Не проведено', tone: 'neutral' }
  if (state === 'FINISHED') return { label: 'Проведено', tone: 'success' }
  return { label: 'Идёт', tone: 'brand' }
}

/** Человекочитаемые названия типов полей (SITE.md §11.1). */
export const fieldTypeLabels: Record<string, string> = {
  SHORT_TEXT: 'Короткий текст',
  LONG_TEXT: 'Длинный текст',
  NUMBER: 'Число',
  URL: 'Ссылка',
  EMAIL: 'E-mail',
  PHONE: 'Телефон',
  DATE: 'Дата',
  SELECT: 'Выпадающий список',
  RADIO: 'Один из вариантов',
  CHECKBOX: 'Флажок',
  FILE_GROUP: 'Загрузка файлов',
  SECTION: 'Секция',
  INFO_BLOCK: 'Инфоблок',
}
