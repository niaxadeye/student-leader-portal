import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  createContest,
  getContest,
  listContests,
  transitionContest,
  updateContest,
} from './api'
import type { CreateContestInput, UpdateContestInput } from './types'

/** Список конкурсов. Scope (SUPER_ADMIN vs ADMIN) применяется на бэкенде. */
export function useAdminContests(status?: string) {
  return useQuery({
    queryKey: ['admin', 'contests', status ?? 'all'],
    queryFn: () => listContests(status),
  })
}

export function useAdminContest(id: string | undefined) {
  return useQuery({
    queryKey: ['admin', 'contest', id],
    queryFn: () => getContest(id!),
    enabled: !!id,
  })
}

export function useCreateContest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateContestInput) => createContest(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'contests'] }),
  })
}

export function useUpdateContest(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: UpdateContestInput) => updateContest(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'contest', id] })
      qc.invalidateQueries({ queryKey: ['admin', 'contests'] })
    },
  })
}

export function useTransitionContest(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (action: 'publish' | 'finish' | 'archive') => transitionContest(id, action),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'contest', id] })
      qc.invalidateQueries({ queryKey: ['admin', 'contests'] })
    },
  })
}
