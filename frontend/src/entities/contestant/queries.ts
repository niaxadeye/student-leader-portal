import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  addContestant,
  deleteContestantAvatar,
  importContestants,
  listContestants,
  removeContestant,
  uploadContestantAvatar,
} from './api'
import type { AddContestantInput } from './types'

/** Конкурсанты конкретного конкурса. */
export function useContestants(contestId: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'contestants', contestId],
    queryFn: () => listContestants(contestId!),
    enabled: !!contestId,
  })
}

function useInvalidate(contestId: string) {
  const qc = useQueryClient()
  return () => {
    qc.invalidateQueries({ queryKey: ['admin', 'contestants', contestId] })
    qc.invalidateQueries({ queryKey: ['admin', 'contest', contestId] })
    qc.invalidateQueries({ queryKey: ['admin', 'contests'] })
  }
}

export function useAddContestant(contestId: string) {
  const invalidate = useInvalidate(contestId)
  return useMutation({
    mutationFn: (input: AddContestantInput) => addContestant(contestId, input),
    onSuccess: invalidate,
  })
}

export function useRemoveContestant(contestId: string) {
  const invalidate = useInvalidate(contestId)
  return useMutation({
    mutationFn: (userId: string) => removeContestant(contestId, userId),
    onSuccess: invalidate,
  })
}

export function useImportContestants(contestId: string) {
  const invalidate = useInvalidate(contestId)
  return useMutation({
    mutationFn: (csv: string) => importContestants(contestId, csv),
    onSuccess: invalidate,
  })
}

export function useUploadContestantAvatar(contestId: string) {
  const qc = useQueryClient()
  const invalidate = useInvalidate(contestId)
  return useMutation({
    mutationFn: ({ userId, image }: { userId: string; image: File }) =>
      uploadContestantAvatar(contestId, userId, image),
    onSuccess: () => {
      invalidate()
      qc.invalidateQueries({ queryKey: ['admin', 'live'] })
      qc.invalidateQueries({ queryKey: ['admin', 'evaluation', 'scores'] })
      qc.invalidateQueries({ queryKey: ['jury', 'live'] })
    },
  })
}

export function useDeleteContestantAvatar(contestId: string) {
  const qc = useQueryClient()
  const invalidate = useInvalidate(contestId)
  return useMutation({
    mutationFn: (userId: string) => deleteContestantAvatar(contestId, userId),
    onSuccess: () => {
      invalidate()
      qc.invalidateQueries({ queryKey: ['admin', 'live'] })
      qc.invalidateQueries({ queryKey: ['admin', 'evaluation', 'scores'] })
      qc.invalidateQueries({ queryKey: ['jury', 'live'] })
    },
  })
}
