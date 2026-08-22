import { useQuery } from '@tanstack/react-query'
import { getChallengeDraw, listContestDraws } from './contestant-api'

export function useChallengeDraw(challengeId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ['challenge-draw', challengeId],
    queryFn: () => getChallengeDraw(challengeId!),
    enabled: !!challengeId && enabled,
  })
}

export function useContestDraws(contestId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ['contest-draws', contestId],
    queryFn: () => listContestDraws(contestId!),
    enabled: !!contestId && enabled,
  })
}
