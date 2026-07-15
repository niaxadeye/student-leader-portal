import { useQuery } from '@tanstack/react-query'
import { fetchMyContests, fetchChallenges, fetchChallenge } from './api'

// Конкурсы текущего конкурсанта; берём первый как активный (кабинет — один конкурс на вход).
export function useMyContests() {
  return useQuery({ queryKey: ['my-contests'], queryFn: fetchMyContests })
}

export function useContest() {
  const q = useMyContests()
  return { ...q, data: q.data?.[0] }
}

export function useChallenges() {
  const { data: contest } = useContest()
  return useQuery({
    queryKey: ['challenges', contest?.id],
    queryFn: () => fetchChallenges(contest!.id),
    enabled: !!contest?.id,
  })
}

export function useChallenge(id: string | undefined) {
  return useQuery({
    queryKey: ['challenge', id],
    queryFn: () => fetchChallenge(id!),
    enabled: !!id,
  })
}
