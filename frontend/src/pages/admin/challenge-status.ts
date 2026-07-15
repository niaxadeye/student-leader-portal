import type { ChallengeStatus } from '@/entities/challenge/admin-types'
import type { BadgeProps } from '@/shared/ui/badge'

/** Статус испытания → метка и тон бейджа (SITE.md §«Испытание»). */
export const challengeStatusMeta: Record<
  ChallengeStatus,
  { label: string; tone: BadgeProps['tone'] }
> = {
  DRAFT: { label: 'Черновик', tone: 'neutral' },
  PUBLISHED: { label: 'Опубликовано', tone: 'success' },
  CLOSED: { label: 'Закрыто', tone: 'warning' },
  ARCHIVED: { label: 'В архиве', tone: 'neutral' },
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
