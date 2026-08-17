export type EventParticipantStatus = 'ACTIVE' | 'BLOCKED' | 'ARCHIVED'

export interface EventParticipant {
  id: string
  event_id: string
  full_name: string
  birth_date: string
  union_card_number: string | null
  sks_barcode: string | null
  status: EventParticipantStatus
  created_at: string
  updated_at: string
  archived_at: string | null
}

export interface ParticipantEvent {
  id: string
  slug: string
  name: string
  timezone: string
}

export interface ParticipantSession {
  participant: EventParticipant
  event: ParticipantEvent
  expires_at?: string
}

export interface NameLoginInput {
  full_name: string
  birth_date: string
}

export interface IdentifierLoginInput {
  value: string
}
