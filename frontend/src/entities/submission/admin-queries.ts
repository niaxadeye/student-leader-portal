import { useQuery } from '@tanstack/react-query'
import { listSubmissions, getSubmissionDetail } from './admin-api'

export function useAdminSubmissions(challengeId: string | undefined, status = '') {
  return useQuery({
    queryKey: ['admin-submissions', challengeId, status],
    queryFn: () => listSubmissions(challengeId!, status),
    enabled: !!challengeId,
  })
}

export function useAdminSubmissionDetail(submissionId: string | undefined) {
  return useQuery({
    queryKey: ['admin-submission', submissionId],
    queryFn: () => getSubmissionDetail(submissionId!),
    enabled: !!submissionId,
  })
}
