import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  changeAdminParticipantStatus,
  createAdminParticipant,
  importAdminParticipants,
  listAdminParticipants,
  updateAdminParticipant,
} from './admin-api'
import type {
  AdminParticipantFilters,
  AdminParticipantInput,
  ParticipantStatusAction,
} from './admin-types'

const participantsKey = (contestId: string) => ['admin', 'event-participants', contestId]

export function useAdminEventParticipants(
  contestId: string | undefined,
  filters: AdminParticipantFilters,
) {
  return useQuery({
    queryKey: [...participantsKey(contestId ?? ''), filters],
    queryFn: () => listAdminParticipants(contestId!, filters),
    enabled: !!contestId,
  })
}

function useInvalidateParticipants(contestId: string) {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: participantsKey(contestId) })
}

export function useCreateEventParticipant(contestId: string) {
  const invalidate = useInvalidateParticipants(contestId)
  return useMutation({
    mutationFn: (input: AdminParticipantInput) => createAdminParticipant(contestId, input),
    onSuccess: invalidate,
  })
}

export function useUpdateEventParticipant(contestId: string) {
  const invalidate = useInvalidateParticipants(contestId)
  return useMutation({
    mutationFn: ({
      participantId,
      input,
    }: {
      participantId: string
      input: AdminParticipantInput
    }) => updateAdminParticipant(contestId, participantId, input),
    onSuccess: invalidate,
  })
}

export function useEventParticipantStatus(contestId: string) {
  const invalidate = useInvalidateParticipants(contestId)
  return useMutation({
    mutationFn: ({
      participantId,
      action,
    }: {
      participantId: string
      action: ParticipantStatusAction
    }) => changeAdminParticipantStatus(contestId, participantId, action),
    onSuccess: invalidate,
  })
}

export function useImportEventParticipants(contestId: string) {
  const invalidate = useInvalidateParticipants(contestId)
  return useMutation({
    mutationFn: (file: File) => importAdminParticipants(contestId, file),
    onSuccess: invalidate,
  })
}
