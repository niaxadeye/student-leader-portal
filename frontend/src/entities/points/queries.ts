import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  adjustAdminParticipantPoints,
  getAdminParticipantPoints,
  getParticipantPoints,
} from './api'
import type { AdminAdjustmentInput } from './types'

export function useParticipantPoints(eventId?: string, participantId?: string) {
  return useQuery({
    // Scope ключа исключает показ кеша предыдущего участника после logout/login.
    queryKey: ['participant', 'points', eventId ?? '', participantId ?? ''],
    queryFn: () => getParticipantPoints(10),
    enabled: !!eventId && !!participantId,
  })
}

const adminPointsKey = (contestId: string, participantId: string) => [
  'admin',
  'event-participant-points',
  contestId,
  participantId,
]

export function useAdminParticipantPoints(
  contestId: string,
  participantId: string | undefined,
  enabled: boolean,
) {
  return useQuery({
    queryKey: adminPointsKey(contestId, participantId ?? ''),
    queryFn: () => getAdminParticipantPoints(contestId, participantId!),
    enabled: enabled && !!participantId,
  })
}

export function useAdjustAdminParticipantPoints(contestId: string, participantId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: AdminAdjustmentInput) =>
      adjustAdminParticipantPoints(contestId, participantId, input),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: adminPointsKey(contestId, participantId) }),
        queryClient.invalidateQueries({
          queryKey: ['participant', 'points', contestId, participantId],
        }),
      ])
    },
  })
}
