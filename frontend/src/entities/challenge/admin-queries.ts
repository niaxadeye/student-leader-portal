import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  addField,
  clearBriefingOverride,
  createChallenge,
  deleteBriefingFile,
  deleteField,
  duplicateChallenge,
  getChallenge,
  getChallengeBriefing,
  listChallenges,
  listFields,
  reorderFields,
  saveBriefingOverride,
  saveChallengeBriefing,
  transitionChallenge,
  updateChallenge,
  updateField,
  uploadBriefingFile,
  uploadOverrideFile,
} from './admin-api'
import type { BriefingInput, ChallengeInput, FieldInput, OverrideInput } from './admin-types'

const listKey = (contestId: string) => ['admin', 'challenges', contestId]
const oneKey = (challengeId: string) => ['admin', 'challenge', challengeId]
const fieldsKey = (challengeId: string) => ['admin', 'challenge-fields', challengeId]

export function useAdminChallenges(contestId: string | undefined) {
  return useQuery({
    queryKey: listKey(contestId ?? ''),
    queryFn: () => listChallenges(contestId!),
    enabled: !!contestId,
  })
}

export function useAdminChallenge(challengeId: string | undefined) {
  return useQuery({
    queryKey: oneKey(challengeId ?? ''),
    queryFn: () => getChallenge(challengeId!),
    enabled: !!challengeId,
  })
}

export function useChallengeFields(challengeId: string | undefined) {
  return useQuery({
    queryKey: fieldsKey(challengeId ?? ''),
    queryFn: () => listFields(challengeId!),
    enabled: !!challengeId,
  })
}

export function useCreateChallenge(contestId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ChallengeInput) => createChallenge(contestId, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: listKey(contestId) }),
  })
}

export function useUpdateChallenge(challengeId: string, contestId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: ChallengeInput) => updateChallenge(challengeId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: oneKey(challengeId) })
      qc.invalidateQueries({ queryKey: listKey(contestId) })
    },
  })
}

export function useTransitionChallenge(challengeId: string, contestId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (action: 'publish' | 'close' | 'archive') =>
      transitionChallenge(challengeId, action),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: oneKey(challengeId) })
      qc.invalidateQueries({ queryKey: listKey(contestId) })
    },
  })
}

export function useDuplicateChallenge(contestId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (challengeId: string) => duplicateChallenge(challengeId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: listKey(contestId) })
      qc.invalidateQueries({ queryKey: ['admin', 'contest', contestId] })
      qc.invalidateQueries({ queryKey: ['admin', 'contests'] })
    },
  })
}

// ── Поля ─────────────────────────────────────────────────────────────────
export function useAddField(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: FieldInput) => addField(challengeId, input),
    onSuccess: () => invalidateFields(qc, challengeId),
  })
}

export function useUpdateField(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { fieldId: string; input: FieldInput }) =>
      updateField(challengeId, v.fieldId, v.input),
    onSuccess: () => invalidateFields(qc, challengeId),
  })
}

export function useDeleteField(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (fieldId: string) => deleteField(challengeId, fieldId),
    onSuccess: () => invalidateFields(qc, challengeId),
  })
}

export function useReorderFields(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (fieldIds: string[]) => reorderFields(challengeId, fieldIds),
    onSuccess: () => invalidateFields(qc, challengeId),
  })
}

function invalidateFields(qc: ReturnType<typeof useQueryClient>, challengeId: string) {
  qc.invalidateQueries({ queryKey: fieldsKey(challengeId) })
  qc.invalidateQueries({ queryKey: oneKey(challengeId) })
}

const briefingKey = (challengeId: string) => ['admin', 'challenge-briefing', challengeId]

export function useAdminBriefing(challengeId: string | undefined) {
  return useQuery({
    queryKey: briefingKey(challengeId ?? ''),
    queryFn: () => getChallengeBriefing(challengeId!),
    enabled: !!challengeId,
  })
}

export function useSaveBriefing(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: BriefingInput) => saveChallengeBriefing(challengeId, input),
    onSuccess: (data) => qc.setQueryData(briefingKey(challengeId), data),
  })
}

export function useUploadBriefingFile(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (file: File) => uploadBriefingFile(challengeId, file),
    onSuccess: (data) => qc.setQueryData(briefingKey(challengeId), data),
  })
}

export function useDeleteBriefingFile(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (fileId: string) => deleteBriefingFile(challengeId, fileId),
    onSuccess: (data) => qc.setQueryData(briefingKey(challengeId), data),
  })
}

export function useSaveBriefingOverride(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, input }: { userId: string; input: OverrideInput }) =>
      saveBriefingOverride(challengeId, userId, input),
    onSuccess: (data) => qc.setQueryData(briefingKey(challengeId), data),
  })
}

export function useClearBriefingOverride(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => clearBriefingOverride(challengeId, userId),
    onSuccess: (data) => qc.setQueryData(briefingKey(challengeId), data),
  })
}

export function useUploadOverrideFile(challengeId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, file }: { userId: string; file: File }) =>
      uploadOverrideFile(challengeId, userId, file),
    onSuccess: (data) => qc.setQueryData(briefingKey(challengeId), data),
  })
}
