import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createAdminTask,
  deleteAdminTask,
  deleteTaskImage,
  getParticipantTask,
  getTaskSubmission,
  listAdminTasks,
  listParticipantTasks,
  listTaskSubmissions,
  moderateTaskSubmission,
  submitParticipantTask,
  transitionAdminTask,
  updateAdminTask,
  uploadTaskImage,
} from './api'
import type { EventTaskInput, TaskSubmissionStatus } from './types'

const adminTasksKey = (contestId: string) => ['admin', 'event-tasks', contestId]
const moderationKey = (contestId: string, status: TaskSubmissionStatus) => [
  'admin',
  'event-task-submissions',
  contestId,
  status,
]
const participantTasksKey = (eventId: string, participantId: string) => [
  'participant',
  'event-tasks',
  eventId,
  participantId,
]

export function useAdminTasks(contestId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: adminTasksKey(contestId ?? ''),
    queryFn: () => listAdminTasks(contestId!),
    enabled: !!contestId && enabled,
  })
}

function useInvalidateAdminTasks(contestId: string) {
  const client = useQueryClient()
  return () => void client.invalidateQueries({ queryKey: adminTasksKey(contestId) })
}

export function useCreateTask(contestId: string) {
  const invalidate = useInvalidateAdminTasks(contestId)
  return useMutation({
    mutationFn: (input: EventTaskInput) => createAdminTask(contestId, input),
    onSuccess: invalidate,
  })
}

export function useUpdateTask(contestId: string, taskId: string) {
  const invalidate = useInvalidateAdminTasks(contestId)
  return useMutation({
    mutationFn: (input: EventTaskInput) => updateAdminTask(contestId, taskId, input),
    onSuccess: invalidate,
  })
}

export function useTransitionTask(contestId: string) {
  const invalidate = useInvalidateAdminTasks(contestId)
  return useMutation({
    mutationFn: ({
      taskId,
      action,
    }: {
      taskId: string
      action: 'activate' | 'disable' | 'archive'
    }) => transitionAdminTask(contestId, taskId, action),
    onSuccess: invalidate,
  })
}

export function useDeleteTask(contestId: string) {
  const invalidate = useInvalidateAdminTasks(contestId)
  return useMutation({
    mutationFn: (taskId: string) => deleteAdminTask(contestId, taskId),
    onSuccess: invalidate,
  })
}

export function useTaskImage(contestId: string, taskId: string) {
  const invalidate = useInvalidateAdminTasks(contestId)
  return useMutation({
    mutationFn: (image: File | null) =>
      image ? uploadTaskImage(contestId, taskId, image) : deleteTaskImage(contestId, taskId),
    onSuccess: invalidate,
  })
}

export function useTaskModeration(
  contestId: string | undefined,
  status: TaskSubmissionStatus,
  enabled = true,
) {
  return useQuery({
    queryKey: moderationKey(contestId ?? '', status),
    queryFn: () => listTaskSubmissions(contestId!, status),
    enabled: !!contestId && enabled,
  })
}

export function useTaskSubmission(contestId: string | undefined, submissionId: string | null) {
  return useQuery({
    queryKey: ['admin', 'event-task-submission', contestId ?? '', submissionId ?? ''],
    queryFn: () => getTaskSubmission(contestId!, submissionId!),
    enabled: !!contestId && !!submissionId,
  })
}

export function useModerateTask(contestId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: ({
      submissionId,
      action,
      comment,
    }: {
      submissionId: string
      action: 'approve' | 'reject'
      comment: string
    }) => moderateTaskSubmission(contestId, submissionId, action, comment),
    onSuccess: (result) => {
      void client.invalidateQueries({ queryKey: ['admin', 'event-task-submissions', contestId] })
      void client.setQueryData(
        ['admin', 'event-task-submission', contestId, result.submission.id],
        result.submission,
      )
    },
  })
}

export function useParticipantTasks(
  eventId: string | undefined,
  participantId: string | undefined,
) {
  return useQuery({
    queryKey: participantTasksKey(eventId ?? '', participantId ?? ''),
    queryFn: listParticipantTasks,
    enabled: !!eventId && !!participantId,
  })
}

export function useParticipantTask(
  eventId: string | undefined,
  participantId: string | undefined,
  taskId: string | undefined,
) {
  return useQuery({
    queryKey: [...participantTasksKey(eventId ?? '', participantId ?? ''), taskId ?? ''],
    queryFn: () => getParticipantTask(taskId!),
    enabled: !!eventId && !!participantId && !!taskId,
  })
}

export function useSubmitParticipantTask(eventId: string, participantId: string, taskId: string) {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (form: FormData) => submitParticipantTask(taskId, form),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: participantTasksKey(eventId, participantId) })
    },
  })
}
