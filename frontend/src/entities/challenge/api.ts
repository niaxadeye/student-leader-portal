// Контестант-API испытаний (бэкенд: modules/challenges, чтение по участию).
import { apiRequest } from '@/shared/api/client'
import type { Challenge, FormField, FieldOption } from './types'
import type { MyContest } from './contest-types'

// Сырое поле с бэкенда: options лежат в settings.options (админ-редактор).
interface RawField {
  id: string
  key: string
  type: FormField['type']
  label: string
  description?: string | null
  help_text?: string | null
  placeholder?: string | null
  required?: boolean
  sort_order: number
  settings?: Record<string, unknown> | null
}

interface RawChallenge {
  id: string
  contest_id: string
  title: string
  short_description?: string | null
  full_description?: string | null
  instructions?: string | null
  status: Challenge['status']
  deadline_at?: string | null
  current_schema_version: number
  my_submission_status?: Challenge['my_submission_status']
  settings?: Record<string, unknown> | null
  fields?: RawField[]
}

function mapField(f: RawField): FormField {
  const s = f.settings ?? {}
  return {
    id: f.id,
    key: f.key,
    type: f.type,
    label: f.label,
    description: f.description ?? undefined,
    help_text: f.help_text ?? undefined,
    placeholder: f.placeholder ?? undefined,
    required: f.required,
    sort_order: f.sort_order,
    options: Array.isArray(s.options) ? (s.options as FieldOption[]) : undefined,
    settings: {
      multiple: s.multiple as boolean | undefined,
      allowed_extensions: s.allowed_extensions as string[] | undefined,
      max_file_size_mb: s.max_file_size_mb as number | undefined,
    },
  }
}

export function mapChallenge(c: RawChallenge): Challenge {
  const s = c.settings ?? {}
  return {
    id: c.id,
    contestId: c.contest_id,
    title: c.title,
    short_description: c.short_description ?? undefined,
    full_description: c.full_description ?? undefined,
    instructions: c.instructions ?? undefined,
    status: c.status,
    deadline_at: c.deadline_at ?? undefined,
    allow_edit_after_submission: (s.allow_edit_after_submission as boolean) ?? true,
    allow_late_submission: (s.allow_late_submission as boolean) ?? false,
    schema_version: c.current_schema_version,
    my_submission_status: c.my_submission_status,
    fields: (c.fields ?? []).map(mapField),
  }
}

// Список конкурсов, где пользователь участник.
export function fetchMyContests(): Promise<MyContest[]> {
  return apiRequest<MyContest[]>('/my/contests')
}

// Опубликованные испытания конкурса (для дашборда).
export async function fetchChallenges(contestId: string): Promise<Challenge[]> {
  const raw = await apiRequest<RawChallenge[]>(`/contests/${contestId}/challenges`)
  return raw.map(mapChallenge)
}

// Одно испытание с полями.
export async function fetchChallenge(challengeId: string): Promise<Challenge> {
  const raw = await apiRequest<RawChallenge>(`/challenges/${challengeId}`)
  return mapChallenge(raw)
}
