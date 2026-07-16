// Админ-API просмотра ответов (бэкенд: modules/submissions, SITE.md §7.6).
import { apiRequest, apiRequestFull } from '@/shared/api/client'
import type { SubmissionFileDTO } from './api'
import type { AnswerValue } from './types'

export interface AdminSubmissionRow {
  id: string
  contestant_user_id: string
  full_name: string
  login: string
  organization: string | null
  status: 'DRAFT' | 'SUBMITTED' | 'LOCKED'
  version: number
  current_revision_number: number
  last_saved_at: string | null
  submitted_at: string | null
  last_resubmitted_at: string | null
  locked: boolean
  file_count: number
}

export interface AdminRevision {
  id: string
  revision_number: number
  action_type: 'SUBMIT' | 'RESUBMIT'
  schema_version: number
  checksum: string
  created_at: string
  answers: Record<string, AnswerValue>
  files: Array<Record<string, unknown>>
}

export interface AdminSubmissionDetail {
  id: string
  challenge_id: string
  contestant: { user_id: string; full_name: string; login: string; organization: string | null }
  status: 'DRAFT' | 'SUBMITTED' | 'LOCKED'
  answers: Record<string, AnswerValue>
  schema_version: number
  version: number
  current_revision_number: number
  submitted_at: string | null
  last_resubmitted_at: string | null
  last_saved_at: string | null
  locked: boolean
  lock_reason: string | null
  files: SubmissionFileDTO[]
  revisions: AdminRevision[]
}

export async function listSubmissions(
  challengeId: string,
  status = '',
): Promise<{ rows: AdminSubmissionRow[]; total: number }> {
  const qs = status ? `?status=${status}` : ''
  const { data, meta } = await apiRequestFull<AdminSubmissionRow[]>(
    `/admin/challenges/${challengeId}/submissions${qs}`,
  )
  return { rows: data, total: (meta?.total as number) ?? data.length }
}

export function getSubmissionDetail(submissionId: string): Promise<AdminSubmissionDetail> {
  return apiRequest<AdminSubmissionDetail>(`/admin/submissions/${submissionId}`)
}

// Запрашивает presigned-URL для скачивания файла (эндпоинт за Bearer-авторизацией).
// Возвращённый URL авторизуется подписью и открывается браузером напрямую.
export async function getFileDownloadUrl(submissionId: string, fileId: string): Promise<string> {
  const { download_url } = await apiRequest<{ download_url: string }>(
    `/admin/submissions/${submissionId}/files/${fileId}`,
  )
  return download_url
}
