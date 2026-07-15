import type { SubmissionStatus } from '@/entities/challenge/types'
import type { UploadedFile } from '@/shared/ui/file-upload'

// Значение ответа поля.
export type AnswerValue = string | number | boolean | string[] | undefined

export interface Submission {
  id: string
  challengeId: string
  status: SubmissionStatus
  version: number
  schema_version: number
  current_revision_number: number
  answers: Record<string, AnswerValue>
  files: Record<string, UploadedFile[]> // по field key
  last_saved_at?: string
  submitted_at?: string
  locked: boolean
  lock_reason?: string
}
