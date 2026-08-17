export type PointsEntryType =
  'LECTURE_ATTENDANCE' | 'TASK_REWARD' | 'MERCH_PURCHASE' | 'ADMIN_ADJUSTMENT' | 'REFUND'

export interface PointsBalance {
  ledger_balance: number
  reserved_points: number
  available_points: number
}

export interface PointsEntry {
  id: string
  event_id: string
  participant_id: string
  amount: number
  type: PointsEntryType
  source_type: string | null
  source_id: string | null
  description: string
  created_at: string
}

export interface PointsOverview {
  balance: PointsBalance
  entries: PointsEntry[]
}

export interface AdminAdjustmentInput {
  amount: number
  reason: string
  idempotency_key: string
}

export interface AdminAdjustmentResult {
  entry: PointsEntry
  balance: PointsBalance
  replayed: boolean
}
