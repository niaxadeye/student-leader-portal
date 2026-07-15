// Контестант-API подачи ответов (бэкенд: modules/submissions).
import { apiRequest, apiPostForm } from '@/shared/api/client'
import type { AnswerValue } from './types'

// Файл, привязанный к работе (ответ бэкенда).
export interface SubmissionFileDTO {
  file_id: string
  field_id: string | null
  field_key: string
  original_name: string
  size_bytes: number | null
  mime_type: string | null
  download_url: string
}

export interface SubmissionDTO {
  id: string
  challenge_id: string
  status: 'DRAFT' | 'SUBMITTED' | 'LOCKED'
  answers: Record<string, AnswerValue>
  schema_version: number
  version: number
  current_revision_number: number
  first_opened_at?: string | null
  last_saved_at?: string | null
  submitted_at?: string | null
  last_resubmitted_at?: string | null
  locked: boolean
  lock_reason?: string | null
  files: SubmissionFileDTO[]
}

// Открыть/создать черновик работы по испытанию.
export function getSubmission(challengeId: string): Promise<SubmissionDTO> {
  return apiRequest<SubmissionDTO>(`/challenges/${challengeId}/submission`)
}

// Сохранить черновик (без ревизии).
export function saveDraft(
  challengeId: string,
  answers: Record<string, AnswerValue>,
): Promise<SubmissionDTO> {
  return apiRequest<SubmissionDTO>(`/challenges/${challengeId}/submission/draft`, {
    method: 'PUT',
    body: { answers },
  })
}

// Отправить (создаёт immutable-ревизию, статус SUBMITTED).
export function submitSubmission(
  challengeId: string,
  answers: Record<string, AnswerValue>,
): Promise<SubmissionDTO> {
  return apiRequest<SubmissionDTO>(`/challenges/${challengeId}/submission/submit`, {
    method: 'POST',
    body: { answers },
  })
}

// Загрузить файл в поле (multipart → MinIO через API).
export function uploadSubmissionFile(
  challengeId: string,
  fieldId: string,
  file: File,
): Promise<SubmissionFileDTO> {
  const form = new FormData()
  form.append('field_id', fieldId)
  form.append('file', file)
  return apiPostForm<SubmissionFileDTO>(`/challenges/${challengeId}/submission/files`, form)
}

// Удалить файл из черновика.
export function deleteSubmissionFile(challengeId: string, fileId: string): Promise<void> {
  return apiRequest<void>(`/challenges/${challengeId}/submission/files/${fileId}`, {
    method: 'DELETE',
  })
}
