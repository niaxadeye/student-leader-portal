export type EventTaskStatus = 'DRAFT' | 'ACTIVE' | 'DISABLED' | 'ARCHIVED'
export type TaskSubmissionStatus = 'PENDING' | 'APPROVED' | 'REJECTED'
export type TaskAssetType = 'IMAGE' | 'LINK'

export interface TaskAsset {
  id: string
  attempt_id: string
  type: TaskAssetType
  url?: string
  original_name?: string
  mime_type?: string
  size_bytes?: number
  sort_order: number
  created_at: string
  download_path?: string
}

export interface TaskSubmissionAttempt {
  id: string
  attempt_number: number
  status: TaskSubmissionStatus
  participant_comment: string | null
  moderator_comment: string | null
  reviewed_by_user_id: string | null
  submitted_at: string
  reviewed_at: string | null
  assets: TaskAsset[]
}

export interface TaskSubmission {
  id: string
  event_id: string
  task_id: string
  participant_id: string
  participant_name?: string
  task_title?: string
  status: TaskSubmissionStatus
  current_attempt: number
  participant_comment: string | null
  moderator_comment: string | null
  reviewed_by_user_id: string | null
  submitted_at: string | null
  reviewed_at: string | null
  reward_granted_at: string | null
  created_at: string
  updated_at: string
  points: number
  allowed_submission_types?: TaskAssetType[]
  attempts?: TaskSubmissionAttempt[]
}

export interface EventTask {
  id: string
  event_id: string
  title: string
  description: string
  image_url: string | null
  icon: string | null
  points: number
  starts_at: string | null
  ends_at: string | null
  status: EventTaskStatus
  sort_order: number
  allowed_submission_types: TaskAssetType[]
  created_at: string
  updated_at: string
  available: boolean
  submission?: TaskSubmission
}

export interface EventTaskInput {
  title: string
  description: string
  icon: string | null
  points: number
  starts_at: string | null
  ends_at: string | null
  sort_order: number
  allowed_submission_types: TaskAssetType[]
}

export interface ModerationResult {
  submission: TaskSubmission
  replayed: boolean
}
