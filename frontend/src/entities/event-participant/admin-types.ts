import type { EventParticipant, EventParticipantStatus } from './types'

export interface AdminParticipantFilters {
  search: string
  status: '' | EventParticipantStatus
  directionId: string
  limit: number
  offset: number
}

export interface AdminParticipantList {
  participants: EventParticipant[]
  total: number
  limit: number
  offset: number
}

export interface AdminParticipantInput {
  full_name: string
  birth_date: string
  union_card_number?: string
  sks_barcode?: string
  direction_id?: string | null
}

export type ParticipantStatusAction = 'block' | 'unblock' | 'archive'

export type ParticipantImportRowStatus = 'added' | 'updated' | 'error' | 'duplicate'

export interface ParticipantImportRow {
  line: number
  status: ParticipantImportRowStatus
  participant_id?: string
  full_name?: string
  direction?: string
  message?: string
}

export interface ParticipantImportResult {
  added: number
  updated: number
  errors: number
  duplicates: number
  rows: ParticipantImportRow[]
}

export type ParticipantExportFormat = 'csv' | 'xlsx'
