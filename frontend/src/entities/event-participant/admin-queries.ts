import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  changeAdminParticipantStatus,
  createAdminParticipant,
  createEventDirection,
  deleteEventDirection,
  importAdminParticipants,
  listAdminParticipants,
  listEventDirections,
  updateAdminParticipant,
  updateEventDirection,
} from './admin-api'
import type {
  AdminParticipantFilters,
  AdminParticipantInput,
  ParticipantStatusAction,
} from './admin-types'

const participantsKey = (contestId: string) => ['admin', 'event-participants', contestId]
export const eventDirectionsKey = (contestId: string) => ['admin', 'event-directions', contestId]

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

export function useEventDirections(contestId: string | undefined) {
  return useQuery({
    queryKey: eventDirectionsKey(contestId ?? ''),
    queryFn: () => listEventDirections(contestId!),
    enabled: !!contestId,
  })
}

function useInvalidateParticipants(contestId: string) {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: participantsKey(contestId) })
}

function useInvalidateDirections(contestId: string) {
  const queryClient = useQueryClient()
  return () => {
    void queryClient.invalidateQueries({ queryKey: eventDirectionsKey(contestId) })
    void queryClient.invalidateQueries({ queryKey: participantsKey(contestId) })
    void queryClient.invalidateQueries({ queryKey: ['admin', 'lectures', contestId] })
  }
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
  const invalidateParticipants = useInvalidateParticipants(contestId)
  const invalidateDirections = useInvalidateDirections(contestId)
  return useMutation({
    mutationFn: (file: File) => importAdminParticipants(contestId, file),
    onSuccess: () => {
      invalidateParticipants()
      invalidateDirections()
    },
  })
}

export function useCreateEventDirection(contestId: string) {
  const invalidate = useInvalidateDirections(contestId)
  return useMutation({
    mutationFn: (name: string) => createEventDirection(contestId, name),
    onSuccess: invalidate,
  })
}

export function useUpdateEventDirection(contestId: string) {
  const invalidate = useInvalidateDirections(contestId)
  return useMutation({
    mutationFn: ({ directionId, name }: { directionId: string; name: string }) =>
      updateEventDirection(contestId, directionId, name),
    onSuccess: invalidate,
  })
}

export function useDeleteEventDirection(contestId: string) {
  const invalidate = useInvalidateDirections(contestId)
  return useMutation({
    mutationFn: (directionId: string) => deleteEventDirection(contestId, directionId),
    onSuccess: invalidate,
  })
}
