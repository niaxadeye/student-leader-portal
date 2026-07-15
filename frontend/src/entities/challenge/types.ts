// Типы полей конструктора формы — SITE.md §11.1.
export type FieldType =
  | 'SHORT_TEXT'
  | 'LONG_TEXT'
  | 'NUMBER'
  | 'URL'
  | 'EMAIL'
  | 'PHONE'
  | 'DATE'
  | 'SELECT'
  | 'RADIO'
  | 'CHECKBOX'
  | 'FILE_GROUP'
  | 'SECTION'
  | 'INFO_BLOCK'

export interface FieldOption {
  value: string
  label: string
}

// Структура поля — SITE.md §11.3.
export interface FormField {
  id: string
  key: string
  type: FieldType
  label: string
  description?: string
  help_text?: string
  placeholder?: string
  required?: boolean
  sort_order: number
  options?: FieldOption[]
  settings?: {
    multiple?: boolean
    allowed_extensions?: string[]
    max_file_size_mb?: number
  }
}

export type SubmissionStatus = 'NOT_STARTED' | 'DRAFT' | 'SUBMITTED' | 'LOCKED'
export type ChallengeStatus = 'DRAFT' | 'PUBLISHED' | 'CLOSED' | 'ARCHIVED'

export interface Challenge {
  id: string
  contestId: string
  title: string
  short_description?: string
  full_description?: string
  instructions?: string
  status: ChallengeStatus
  deadline_at?: string
  allow_edit_after_submission: boolean
  allow_late_submission: boolean
  schema_version: number
  my_submission_status?: SubmissionStatus
  fields: FormField[]
}
