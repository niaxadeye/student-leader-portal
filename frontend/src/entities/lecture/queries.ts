import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createAdminLecture,
  deleteAdminLecture,
  getAdminLecture,
  issueParticipantQRCode,
  listAdminLectures,
  listLectureAttendance,
  listParticipantLectures,
  scanLectureAttendance,
  transitionAdminLecture,
  updateAdminLecture,
} from './api'
import type { LectureInput, ScannerType } from './types'

const adminLecturesKey = (contestId: string) => ['admin', 'lectures', contestId]
const lectureKey = (contestId: string, lectureId: string) => [
  'admin',
  'lecture',
  contestId,
  lectureId,
]

export function useAdminLectures(contestId: string | undefined) {
  return useQuery({
    queryKey: adminLecturesKey(contestId ?? ''),
    queryFn: () => listAdminLectures(contestId!),
    enabled: !!contestId,
  })
}

export function useAdminLecture(contestId: string | undefined, lectureId: string | undefined) {
  return useQuery({
    queryKey: lectureKey(contestId ?? '', lectureId ?? ''),
    queryFn: () => getAdminLecture(contestId!, lectureId!),
    enabled: !!contestId && !!lectureId,
  })
}

function useInvalidateLectures(contestId: string) {
  const queryClient = useQueryClient()
  return (lectureId?: string) => {
    void queryClient.invalidateQueries({ queryKey: adminLecturesKey(contestId) })
    if (lectureId)
      void queryClient.invalidateQueries({ queryKey: lectureKey(contestId, lectureId) })
  }
}

export function useCreateLecture(contestId: string) {
  const invalidate = useInvalidateLectures(contestId)
  return useMutation({
    mutationFn: (input: LectureInput) => createAdminLecture(contestId, input),
    onSuccess: () => invalidate(),
  })
}

export function useUpdateLecture(contestId: string, lectureId: string) {
  const invalidate = useInvalidateLectures(contestId)
  return useMutation({
    mutationFn: (input: LectureInput) => updateAdminLecture(contestId, lectureId, input),
    onSuccess: () => invalidate(lectureId),
  })
}

export function useTransitionLecture(contestId: string) {
  const invalidate = useInvalidateLectures(contestId)
  return useMutation({
    mutationFn: ({ lectureId, action }: { lectureId: string; action: 'activate' | 'finish' }) =>
      transitionAdminLecture(contestId, lectureId, action),
    onSuccess: (lecture) => invalidate(lecture.id),
  })
}

export function useDeleteLecture(contestId: string) {
  const invalidate = useInvalidateLectures(contestId)
  return useMutation({
    mutationFn: (lectureId: string) => deleteAdminLecture(contestId, lectureId),
    onSuccess: () => invalidate(),
  })
}

export function useScanLecture(contestId: string, lectureId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ token, scannerType }: { token: string; scannerType: ScannerType }) =>
      scanLectureAttendance(contestId, lectureId, token, scannerType),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ['admin', 'lecture-attendance', contestId, lectureId],
      })
    },
  })
}

export function useLectureAttendance(contestId: string | undefined, lectureId: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'lecture-attendance', contestId ?? '', lectureId ?? ''],
    queryFn: () => listLectureAttendance(contestId!, lectureId!),
    enabled: !!contestId && !!lectureId,
  })
}

export function useParticipantQRCode(
  eventId: string | undefined,
  participantId: string | undefined,
) {
  return useQuery({
    queryKey: ['participant', 'qr-code', eventId ?? '', participantId ?? ''],
    queryFn: issueParticipantQRCode,
    enabled: !!eventId && !!participantId,
    refetchInterval: 25_000,
    staleTime: 20_000,
    refetchOnWindowFocus: true,
  })
}

export function useParticipantLectures(
  eventId: string | undefined,
  participantId: string | undefined,
) {
  return useQuery({
    queryKey: ['participant', 'lectures', eventId ?? '', participantId ?? ''],
    queryFn: listParticipantLectures,
    enabled: !!eventId && !!participantId,
  })
}
