import { ApiRequestError } from '@/shared/api/client'
import { putJuryScore } from './jury-api'
import type { JuryScoreWrite } from './types'

const DB_NAME = 'student-leader-evaluation'
const DB_VERSION = 1
const STORE = 'pending_scores'
const CONTEXT_INDEX = 'by_context'
const RETRY_BASE_MS = 1_000
const RETRY_MAX_MS = 30_000
const MAX_CONFLICT_REBASES = 5

export type PendingScoreStatus = 'PENDING' | 'SYNCING' | 'CONFLICT' | 'REJECTED'

export interface PendingScoreMutation {
  key: string
  mutation_id: string
  challenge_id: string
  performance_id: string
  criterion_id: string
  evaluator_user_id: string
  value: number
  base_revision: number
  local_sequence: number
  created_at_client: string
  status: PendingScoreStatus
  attempts: number
  conflict_rebases: number
  next_attempt_at: number
  last_error_code?: string
}

export interface ScoreSyncSnapshot {
  ready: boolean
  storageAvailable: boolean
  flushing: boolean
  online: boolean
  pendingCount: number
  rejectedCount: number
  conflictCount: number
  values: Record<string, PendingScoreMutation>
}

export interface EnqueueScoreInput {
  performanceId: string
  criterionId: string
  value: number
  baseRevision: number
}

interface ScoreSyncOptions {
  challengeId: string
  evaluatorUserId: string
  activePerformanceId: string
  send?: (mutation: PendingScoreMutation, keepalive: boolean) => Promise<JuryScoreWrite>
  onAcknowledged?: (result: JuryScoreWrite) => void
}

const EMPTY_SNAPSHOT: ScoreSyncSnapshot = {
  ready: false,
  storageAvailable: true,
  flushing: false,
  online: typeof navigator === 'undefined' ? true : navigator.onLine,
  pendingCount: 0,
  rejectedCount: 0,
  conflictCount: 0,
  values: {},
}

let lastSequence = 0

function nextSequence(): number {
  lastSequence = Math.max(lastSequence + 1, Date.now())
  return lastSequence
}

function scoreKey(
  evaluatorUserId: string,
  challengeId: string,
  performanceId: string,
  criterionId: string,
) {
  return [evaluatorUserId, challengeId, performanceId, criterionId].join(':')
}

function requestResult<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'))
  })
}

function transactionDone(tx: IDBTransaction): Promise<void> {
  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve()
    tx.onabort = () => reject(tx.error ?? new Error('IndexedDB transaction aborted'))
    tx.onerror = () => reject(tx.error ?? new Error('IndexedDB transaction failed'))
  })
}

let dbPromise: Promise<IDBDatabase> | null = null

function openScoreDB(): Promise<IDBDatabase> {
  if (dbPromise) return dbPromise
  dbPromise = new Promise((resolve, reject) => {
    if (typeof indexedDB === 'undefined') {
      reject(new Error('IndexedDB unavailable'))
      return
    }
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      const store = db.createObjectStore(STORE, { keyPath: 'key' })
      store.createIndex(CONTEXT_INDEX, ['evaluator_user_id', 'challenge_id'], { unique: false })
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => {
      dbPromise = null
      reject(request.error ?? new Error('IndexedDB open failed'))
    }
  })
  return dbPromise
}

async function listContext(
  evaluatorUserId: string,
  challengeId: string,
): Promise<PendingScoreMutation[]> {
  const db = await openScoreDB()
  const tx = db.transaction(STORE, 'readonly')
  const done = transactionDone(tx)
  const result = await requestResult(
    tx
      .objectStore(STORE)
      .index(CONTEXT_INDEX)
      .getAll(IDBKeyRange.only([evaluatorUserId, challengeId])),
  )
  await done
  return result.sort((a, b) => a.local_sequence - b.local_sequence)
}

async function getMutation(key: string): Promise<PendingScoreMutation | undefined> {
  const db = await openScoreDB()
  const tx = db.transaction(STORE, 'readonly')
  const done = transactionDone(tx)
  const result = await requestResult(tx.objectStore(STORE).get(key))
  await done
  return result
}

async function putMutation(mutation: PendingScoreMutation): Promise<void> {
  const db = await openScoreDB()
  const tx = db.transaction(STORE, 'readwrite')
  const done = transactionDone(tx)
  tx.objectStore(STORE).put(mutation)
  await done
}

async function deleteMutationIfCurrent(key: string, mutationId: string): Promise<boolean> {
  const db = await openScoreDB()
  const tx = db.transaction(STORE, 'readwrite')
  const done = transactionDone(tx)
  const store = tx.objectStore(STORE)
  const current = await requestResult(store.get(key))
  if (!current || current.mutation_id !== mutationId) {
    await done
    return false
  }
  store.delete(key)
  await done
  return true
}

function retryDelay(attempts: number): number {
  return Math.min(RETRY_MAX_MS, RETRY_BASE_MS * 2 ** Math.min(attempts, 5))
}

function isTransient(error: unknown): boolean {
  if (!(error instanceof ApiRequestError)) return true
  return error.status === 408 || error.status === 429 || error.status >= 500
}

export class JuryScoreSync {
  private readonly options: ScoreSyncOptions
  private readonly listeners = new Set<() => void>()
  private snapshot: ScoreSyncSnapshot = EMPTY_SNAPSHOT
  private timer: number | null = null
  private started = false
  private flushPromise: Promise<void> | null = null
  private enqueueChain: Promise<void> = Promise.resolve()
  private networkFailed = false

  constructor(options: ScoreSyncOptions) {
    this.options = options
  }

  subscribe = (listener: () => void): (() => void) => {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  getSnapshot = (): ScoreSyncSnapshot => this.snapshot

  start(): void {
    if (this.started) return
    this.started = true
    window.addEventListener('online', this.handleOnline)
    window.addEventListener('offline', this.handleOffline)
    document.addEventListener('visibilitychange', this.handleVisibility)
    window.addEventListener('pagehide', this.handlePageHide)
    void this.reload().then(() => this.schedule(100))
  }

  stop(): void {
    if (!this.started) return
    window.removeEventListener('online', this.handleOnline)
    window.removeEventListener('offline', this.handleOffline)
    document.removeEventListener('visibilitychange', this.handleVisibility)
    window.removeEventListener('pagehide', this.handlePageHide)
    if (this.timer != null) window.clearTimeout(this.timer)
    this.timer = null
    this.started = false
    void this.flush(true)
  }

  enqueue(input: EnqueueScoreInput): Promise<void> {
    const next = this.enqueueChain.then(() => this.doEnqueue(input))
    this.enqueueChain = next.catch(() => undefined)
    return next
  }

  private async doEnqueue(input: EnqueueScoreInput): Promise<void> {
    const key = scoreKey(
      this.options.evaluatorUserId,
      this.options.challengeId,
      input.performanceId,
      input.criterionId,
    )
    try {
      const current = await getMutation(key)
      const mutation: PendingScoreMutation = {
        key,
        mutation_id: crypto.randomUUID(),
        challenge_id: this.options.challengeId,
        performance_id: input.performanceId,
        criterion_id: input.criterionId,
        evaluator_user_id: this.options.evaluatorUserId,
        value: input.value,
        // Coalesced local edits still build on the last acknowledged revision.
        base_revision: current?.base_revision ?? input.baseRevision,
        local_sequence: nextSequence(),
        created_at_client: new Date().toISOString(),
        status: 'PENDING',
        attempts: 0,
        conflict_rebases: 0,
        next_attempt_at: Date.now() + RETRY_BASE_MS,
      }
      await putMutation(mutation)
      await this.reload()
      this.schedule(RETRY_BASE_MS)
    } catch {
      this.patchSnapshot({ storageAvailable: false, ready: true })
    }
  }

  async retryFailed(): Promise<void> {
    const list = await listContext(this.options.evaluatorUserId, this.options.challengeId)
    await Promise.all(
      list
        .filter((item) => item.status === 'CONFLICT' || item.status === 'REJECTED')
        .map((item) =>
          putMutation({
            ...item,
            status: 'PENDING',
            attempts: 0,
            conflict_rebases: 0,
            next_attempt_at: Date.now(),
            last_error_code: undefined,
          }),
        ),
    )
    await this.reload()
    this.schedule(0)
  }

  flush(keepalive = false): Promise<void> {
    if (this.flushPromise) return this.flushPromise
    this.flushPromise = this.doFlush(keepalive).finally(() => {
      this.flushPromise = null
    })
    return this.flushPromise
  }

  private async doFlush(keepalive: boolean): Promise<void> {
    if (!navigator.onLine) {
      this.patchSnapshot({ online: false })
      return
    }
    this.patchSnapshot({ flushing: true, online: navigator.onLine && !this.networkFailed })
    try {
      const list = await listContext(this.options.evaluatorUserId, this.options.challengeId)
      for (const queued of list) {
        if (queued.status === 'REJECTED' || queued.status === 'CONFLICT') continue
        if (!keepalive && queued.next_attempt_at > Date.now()) continue
        const current = await getMutation(queued.key)
        if (!current || current.mutation_id !== queued.mutation_id) continue
        await putMutation({ ...current, status: 'SYNCING' })
        await this.reload()
        try {
          const send = this.options.send ?? defaultSend
          const result = await send(current, keepalive)
          this.networkFailed = false
          this.patchSnapshot({ online: true })
          const deleted = await deleteMutationIfCurrent(current.key, current.mutation_id)
          if (!deleted) {
            const replacement = await getMutation(current.key)
            if (replacement) {
              await putMutation({
                ...replacement,
                base_revision: result.revision,
                status: 'PENDING',
                next_attempt_at: Date.now(),
              })
            }
          }
          this.options.onAcknowledged?.(result)
        } catch (error) {
          if (error instanceof ApiRequestError) {
            this.networkFailed = false
            this.patchSnapshot({ online: navigator.onLine })
          }
          await this.handleSendError(current, error)
        }
      }
    } catch {
      this.patchSnapshot({ storageAvailable: false })
    } finally {
      await this.reload()
      this.patchSnapshot({ flushing: false })
      await this.scheduleNext()
    }
  }

  private async handleSendError(sent: PendingScoreMutation, error: unknown): Promise<void> {
    const current = await getMutation(sent.key)
    if (!current || current.mutation_id !== sent.mutation_id) return

    if (error instanceof ApiRequestError && error.code === 'EVALUATION_REVISION_CONFLICT') {
      const revision = Number(error.details?.current_revision)
      const rebases = current.conflict_rebases + 1
      await putMutation({
        ...current,
        base_revision:
          Number.isInteger(revision) && revision >= 0 ? revision : current.base_revision,
        status: rebases >= MAX_CONFLICT_REBASES ? 'CONFLICT' : 'PENDING',
        conflict_rebases: rebases,
        next_attempt_at: Date.now(),
        last_error_code: error.code,
      })
      return
    }

    if (isTransient(error)) {
      const attempts = current.attempts + 1
      await putMutation({
        ...current,
        status: 'PENDING',
        attempts,
        next_attempt_at: Date.now() + retryDelay(attempts),
        last_error_code: error instanceof ApiRequestError ? error.code : 'NETWORK_ERROR',
      })
      if (!(error instanceof ApiRequestError) || !navigator.onLine) {
        this.networkFailed = true
        this.patchSnapshot({ online: false })
      }
      return
    }

    await putMutation({
      ...current,
      status: 'REJECTED',
      last_error_code: error instanceof ApiRequestError ? error.code : 'SYNC_REJECTED',
    })
  }

  private async reload(): Promise<void> {
    try {
      const list = await listContext(this.options.evaluatorUserId, this.options.challengeId)
      const values: Record<string, PendingScoreMutation> = {}
      for (const item of list) {
        if (item.performance_id === this.options.activePerformanceId) {
          values[item.criterion_id] = item
        }
      }
      this.setSnapshot({
        ...this.snapshot,
        ready: true,
        storageAvailable: true,
        online: navigator.onLine && !this.networkFailed,
        pendingCount: list.length,
        rejectedCount: list.filter((item) => item.status === 'REJECTED').length,
        conflictCount: list.filter((item) => item.status === 'CONFLICT').length,
        values,
      })
    } catch {
      this.patchSnapshot({ ready: true, storageAvailable: false })
    }
  }

  private schedule(delay: number): void {
    if (!this.started) return
    if (this.timer != null) window.clearTimeout(this.timer)
    this.timer = window.setTimeout(
      () => {
        this.timer = null
        void this.flush()
      },
      Math.max(0, delay),
    )
  }

  private async scheduleNext(): Promise<void> {
    if (!this.started || this.snapshot.pendingCount === 0) return
    try {
      // Retry work from every performance in this evaluator/challenge context,
      // not only the contestant currently rendered by the scorecard.
      const candidates = (
        await listContext(this.options.evaluatorUserId, this.options.challengeId)
      ).filter((item) => item.status === 'PENDING' || item.status === 'SYNCING')
      if (candidates.length === 0) return
      const next = Math.min(...candidates.map((item) => item.next_attempt_at))
      this.schedule(Math.max(100, next - Date.now()))
    } catch {
      this.patchSnapshot({ storageAvailable: false })
    }
  }

  private setSnapshot(next: ScoreSyncSnapshot): void {
    this.snapshot = next
    this.listeners.forEach((listener) => listener())
  }

  private patchSnapshot(patch: Partial<ScoreSyncSnapshot>): void {
    this.setSnapshot({ ...this.snapshot, ...patch })
  }

  private handleOnline = () => {
    this.networkFailed = false
    this.patchSnapshot({ online: true })
    this.schedule(0)
  }

  private handleOffline = () => this.patchSnapshot({ online: false })

  private handleVisibility = () => {
    if (document.visibilityState === 'hidden') void this.flush(true)
  }

  private handlePageHide = () => void this.flush(true)
}

async function defaultSend(
  mutation: PendingScoreMutation,
  keepalive: boolean,
): Promise<JuryScoreWrite> {
  return putJuryScore(
    mutation.challenge_id,
    {
      performance_id: mutation.performance_id,
      criterion_id: mutation.criterion_id,
      score: mutation.value,
      mutation_id: mutation.mutation_id,
      base_revision: mutation.base_revision,
    },
    keepalive,
  )
}

export { EMPTY_SNAPSHOT }
