import { apiRequest } from '@/shared/api/client'
import { participantApiRequest } from '@/entities/event-participant/api'
import type {
  Lecture,
  LectureAttendance,
  LectureInput,
  ParticipantLecture,
  ParticipantQRCode,
  ScannerType,
  ScanResult,
} from './types'

function lecturesPath(contestId: string): string {
  return `/admin/contests/${encodeURIComponent(contestId)}/lectures`
}

function lecturePath(contestId: string, lectureId: string): string {
  return `${lecturesPath(contestId)}/${encodeURIComponent(lectureId)}`
}

export function listAdminLectures(contestId: string): Promise<Lecture[]> {
  return apiRequest(lecturesPath(contestId))
}

export function getAdminLecture(contestId: string, lectureId: string): Promise<Lecture> {
  return apiRequest(lecturePath(contestId, lectureId))
}

export function createAdminLecture(contestId: string, input: LectureInput): Promise<Lecture> {
  return apiRequest(lecturesPath(contestId), { method: 'POST', body: input })
}

export function updateAdminLecture(
  contestId: string,
  lectureId: string,
  input: LectureInput,
): Promise<Lecture> {
  return apiRequest(lecturePath(contestId, lectureId), { method: 'PATCH', body: input })
}

export function transitionAdminLecture(
  contestId: string,
  lectureId: string,
  action: 'activate' | 'finish',
): Promise<Lecture> {
  return apiRequest(`${lecturePath(contestId, lectureId)}/${action}`, { method: 'POST' })
}

export function deleteAdminLecture(contestId: string, lectureId: string): Promise<void> {
  return apiRequest(lecturePath(contestId, lectureId), { method: 'DELETE' })
}

export function scanLectureAttendance(
  contestId: string,
  lectureId: string,
  token: string,
  scannerType: ScannerType,
): Promise<ScanResult> {
  return apiRequest(`${lecturePath(contestId, lectureId)}/attendance/scan`, {
    method: 'POST',
    body: { token, scanner_type: scannerType },
  })
}

export function listLectureAttendance(
  contestId: string,
  lectureId: string,
): Promise<LectureAttendance[]> {
  return apiRequest(`${lecturePath(contestId, lectureId)}/attendance`)
}

export function issueParticipantQRCode(): Promise<ParticipantQRCode> {
  return participantApiRequest('/participant/qr', { method: 'POST' })
}

export function listParticipantLectures(): Promise<ParticipantLecture[]> {
  return participantApiRequest('/participant/lectures')
}
