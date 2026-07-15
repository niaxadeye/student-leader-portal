import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  addField,
  createChallenge,
  deleteField,
  duplicateChallenge,
  getChallenge,
  listChallenges,
  listFields,
  reorderFields,
  transitionChallenge,
  updateChallenge,
  updateField,
} from './admin-api'
import type { ChallengeInput, FieldInput } from './admin-types'

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
    onSuccess: () => qc.invalidateQueries({ queryKey: listKey(contestId) }),
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
