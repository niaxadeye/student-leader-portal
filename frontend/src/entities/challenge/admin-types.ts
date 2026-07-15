// Админ-типы конструктора испытаний (бэкенд: modules/challenges, SITE.md §10–12).
// Отдельно от контестант-типов (Challenge/FormField), чтобы не ломать мок-путь.
import type { FieldType } from './types'

export type ChallengeStatus = 'DRAFT' | 'PUBLISHED' | 'CLOSED' | 'ARCHIVED'

/** Испытание из админ-API. */
export interface AdminChallenge {
  id: string
  contest_id: string
  title: string
  slug: string
  short_description: string | null
  full_description: string | null
  instructions: string | null
  status: ChallengeStatus
  sort_order: number
  open_at: string | null
  deadline_at: string | null
  close_at: string | null
  current_schema_version: number
  fields_count: number
  created_at: string
  updated_at: string
  published_at: string | null
  archived_at: string | null
}

/** Поле конструктора (SITE.md §11.3). */
export interface AdminField {
  id: string
  key: string
  type: FieldType
  label: string
  description: string | null
  help_text: string | null
  placeholder: string | null
  required: boolean
  sort_order: number
  settings: Record<string, unknown>
  validation: Record<string, unknown>
  visibility: Record<string, unknown>
}

export interface ChallengeInput {
  title: string
  slug?: string
  short_description?: string | null
  full_description?: string | null
  instructions?: string | null
  open_at?: string | null
  deadline_at?: string | null
  close_at?: string | null
}

export interface FieldInput {
  key: string
  type: FieldType
  label: string
  description?: string | null
  help_text?: string | null
  placeholder?: string | null
  required?: boolean
  settings?: Record<string, unknown>
  validation?: Record<string, unknown>
  visibility?: Record<string, unknown>
}
