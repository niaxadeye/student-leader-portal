import { participantApiRequest } from '@/entities/event-participant/api'
import {
  API_BASE_URL,
  ApiRequestError,
  apiPostForm,
  apiRequest,
  type ApiError,
} from '@/shared/api/client'
import type {
  EventTask,
  EventTaskInput,
  ModerationResult,
  TaskSubmission,
  TaskSubmissionStatus,
} from './types'

function tasksPath(contestId: string): string {
  return `/admin/contests/${encodeURIComponent(contestId)}/tasks`
}

function taskPath(contestId: string, taskId: string): string {
  return `${tasksPath(contestId)}/${encodeURIComponent(taskId)}`
}

function moderationPath(contestId: string): string {
  return `/admin/contests/${encodeURIComponent(contestId)}/task-submissions`
}

export const listAdminTasks = (contestId: string): Promise<EventTask[]> =>
  apiRequest(tasksPath(contestId))

export const createAdminTask = (contestId: string, input: EventTaskInput): Promise<EventTask> =>
  apiRequest(tasksPath(contestId), { method: 'POST', body: input })

export const updateAdminTask = (
  contestId: string,
  taskId: string,
  input: EventTaskInput,
): Promise<EventTask> => apiRequest(taskPath(contestId, taskId), { method: 'PATCH', body: input })

export const transitionAdminTask = (
  contestId: string,
  taskId: string,
  action: 'activate' | 'disable' | 'archive',
): Promise<EventTask> => apiRequest(`${taskPath(contestId, taskId)}/${action}`, { method: 'POST' })

export const deleteAdminTask = (contestId: string, taskId: string): Promise<void> =>
  apiRequest(taskPath(contestId, taskId), { method: 'DELETE' })

export const uploadTaskImage = (
  contestId: string,
  taskId: string,
  image: File,
): Promise<EventTask> => {
  const form = new FormData()
  form.append('image', image)
  return apiPostForm(`${taskPath(contestId, taskId)}/image`, form)
}

export const deleteTaskImage = (contestId: string, taskId: string): Promise<EventTask> =>
  apiRequest(`${taskPath(contestId, taskId)}/image`, { method: 'DELETE' })

export const uploadTaskIcon = (
  contestId: string,
  taskId: string,
  image: File,
): Promise<EventTask> => {
  const form = new FormData()
  form.append('image', image)
  return apiPostForm(`${taskPath(contestId, taskId)}/icon`, form)
}

export const deleteTaskIcon = (contestId: string, taskId: string): Promise<EventTask> =>
  apiRequest(`${taskPath(contestId, taskId)}/icon`, { method: 'DELETE' })

export const listTaskSubmissions = (
  contestId: string,
  status: TaskSubmissionStatus,
): Promise<TaskSubmission[]> =>
  apiRequest(`${moderationPath(contestId)}?status=${encodeURIComponent(status)}`)

export const getTaskSubmission = (
  contestId: string,
  submissionId: string,
): Promise<TaskSubmission> =>
  apiRequest(`${moderationPath(contestId)}/${encodeURIComponent(submissionId)}`)

export const moderateTaskSubmission = (
  contestId: string,
  submissionId: string,
  action: 'approve' | 'reject',
  comment: string,
): Promise<ModerationResult> =>
  apiRequest(`${moderationPath(contestId)}/${encodeURIComponent(submissionId)}/${action}`, {
    method: 'POST',
    body: { comment },
  })

export const getAdminTaskAssetURL = (
  contestId: string,
  submissionId: string,
  assetId: string,
): Promise<{ download_url: string }> =>
  apiRequest(
    `${moderationPath(contestId)}/${encodeURIComponent(submissionId)}/assets/${encodeURIComponent(assetId)}`,
  )

export const listParticipantTasks = (): Promise<EventTask[]> =>
  participantApiRequest('/participant/tasks')

export const getParticipantTask = (taskId: string): Promise<EventTask> =>
  participantApiRequest(`/participant/tasks/${encodeURIComponent(taskId)}`)

export const getParticipantTaskAssetURL = (assetId: string): Promise<{ download_url: string }> =>
  participantApiRequest(`/participant/task-assets/${encodeURIComponent(assetId)}`)

export async function submitParticipantTask(
  taskId: string,
  form: FormData,
): Promise<TaskSubmission> {
  let response: Response
  try {
    response = await fetch(
      `${API_BASE_URL}/participant/tasks/${encodeURIComponent(taskId)}/submissions`,
      { method: 'POST', credentials: 'include', body: form },
    )
  } catch {
    throw new ApiRequestError(0, {
      code: 'NETWORK_ERROR',
      message: 'Не удалось связаться с сервером',
    })
  }
  const json = await response.json().catch(() => null)
  if (!response.ok) {
    const error: ApiError = json?.error ?? {
      code: 'INTERNAL_ERROR',
      message: 'Не удалось отправить подтверждение',
    }
    throw new ApiRequestError(response.status, error)
  }
  return json.data as TaskSubmission
}
