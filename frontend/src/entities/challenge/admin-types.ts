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
  held_at: string | null
  venue: string | null
  accepts_submissions: boolean
  scheme_type: string | null
  live_state: string | null
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
  held_at?: string | null
  venue?: string | null
  accepts_submissions?: boolean
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

export interface AdminBriefingFile {
  file_id: string
  original_name: string
  size_bytes?: number | null
  mime_type?: string | null
  download_url?: string | null
}

export interface AdminBriefingOverride {
  custom_text: boolean
  body_text: string
  custom_publish: boolean
  publish_at: string | null
  hidden: boolean
  replace_files: boolean
  files: AdminBriefingFile[]
}

export interface AdminBriefingContestant {
  user_id: string
  login: string
  full_name: string
  organization: string | null
  visible: boolean
  publish_at: string | null
  personalized: boolean
  override: AdminBriefingOverride | null
}

export interface AdminBriefing {
  body_text: string
  publish_at: string | null
  updated_at: string
  files: AdminBriefingFile[]
  contestants: AdminBriefingContestant[]
}

export interface BriefingInput {
  body_text: string
  publish_at: string | null
}

export interface OverrideInput {
  custom_text: boolean
  body_text: string
  custom_publish: boolean
  publish_at: string | null
  hidden: boolean
  replace_files: boolean
}
