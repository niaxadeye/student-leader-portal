import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getSubmission,
  saveDraft,
  submitSubmission,
  uploadSubmissionFile,
  deleteSubmissionFile,
} from './api'
import type { AnswerValue } from './types'

export function useSubmission(challengeId: string | undefined) {
  return useQuery({
    queryKey: ['submission', challengeId],
    queryFn: () => getSubmission(challengeId!),
    enabled: !!challengeId,
  })
}

export function useSaveDraft(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (answers: Record<string, AnswerValue>) => saveDraft(challengeId, answers),
    onSuccess: (data) => qc.setQueryData(['submission', challengeId], data),
  })
}

export function useSubmit(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (answers: Record<string, AnswerValue>) => submitSubmission(challengeId, answers),
    onSuccess: (data) => qc.setQueryData(['submission', challengeId], data),
  })
}

export function useUploadFile(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fieldId, file }: { fieldId: string; file: File }) =>
      uploadSubmissionFile(challengeId, fieldId, file),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['submission', challengeId] }),
  })
}

export function useDeleteFile(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (fileId: string) => deleteSubmissionFile(challengeId, fileId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['submission', challengeId] }),
  })
}
