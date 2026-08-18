export type LectureStatus = 'DRAFT' | 'ACTIVE' | 'FINISHED'
export type ScannerType = 'CAMERA' | 'USB' | 'MANUAL'

export interface Lecture {
  id: string
  event_id: string
  title: string
  description: string | null
  points: number
  starts_at: string | null
  ends_at: string | null
  attendance_starts_at: string | null
  attendance_ends_at: string | null
  status: LectureStatus
  direction_ids: string[]
  directions: LectureDirection[]
  created_at: string
  updated_at: string
}

export interface LectureDirection {
  id: string
  name: string
}

export interface LectureInput {
  title: string
  description: string | null
  points: number
  starts_at: string | null
  ends_at: string | null
  attendance_starts_at: string | null
  attendance_ends_at: string | null
  direction_ids: string[]
}

export interface LectureAttendance {
  id: string
  event_id: string
  lecture_id: string
  participant_id: string
  participant_name: string
  scanned_by_user_id: string
  scanner_type: ScannerType
  points_awarded: number
  created_at: string
}

export interface ScanResult {
  attendance: LectureAttendance
  already_checked: boolean
}

export interface ParticipantQRCode {
  token: string
  expires_at: string
  ttl_seconds: number
}

export interface ParticipantLecture {
  lecture: Lecture
  attendance: LectureAttendance | null
}
