import { useCallback, useEffect, useMemo, useSyncExternalStore } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { EMPTY_SNAPSHOT, JuryScoreSync, type EnqueueScoreInput } from './score-sync'
import type { JuryScorecard } from './types'

export function useJuryScoreAutosave(
  challengeId: string,
  evaluatorUserId: string,
  performanceId: string | null,
) {
  const queryClient = useQueryClient()
  const sync = useMemo(
    () =>
      performanceId && evaluatorUserId
        ? new JuryScoreSync({
            challengeId,
            evaluatorUserId,
            activePerformanceId: performanceId,
            onAcknowledged: (result) => {
              queryClient.setQueriesData<JuryScorecard>(
                { queryKey: ['jury', 'scorecard', challengeId] },
                (previous) => {
                  if (!previous || previous.performance_id !== performanceId) return previous
                  let filled = 0
                  const criteria = previous.criteria.map((criterion) => {
                    const next =
                      criterion.id === result.criterion_id && criterion.revision <= result.revision
                        ? { ...criterion, score: result.score, revision: result.revision }
                        : criterion
                    if (next.score != null) filled++
                    return next
                  })
                  return { ...previous, criteria, filled, total: result.total }
                },
              )
            },
          })
        : null,
    [challengeId, evaluatorUserId, performanceId, queryClient],
  )

  useEffect(() => {
    sync?.start()
    return () => sync?.stop()
  }, [sync])

  const snapshot = useSyncExternalStore(
    sync?.subscribe ?? emptySubscribe,
    sync?.getSnapshot ?? getEmptySnapshot,
    getEmptySnapshot,
  )

  const enqueue = useCallback(
    (input: Omit<EnqueueScoreInput, 'performanceId'>) => {
      if (!sync || !performanceId) return Promise.resolve()
      return sync.enqueue({ ...input, performanceId })
    },
    [performanceId, sync],
  )

  return {
    snapshot,
    enqueue,
    flush: () => sync?.flush() ?? Promise.resolve(),
    retryFailed: () => sync?.retryFailed() ?? Promise.resolve(),
  }
}

function emptySubscribe(): () => void {
  return () => undefined
}

function getEmptySnapshot() {
  return EMPTY_SNAPSHOT
}
