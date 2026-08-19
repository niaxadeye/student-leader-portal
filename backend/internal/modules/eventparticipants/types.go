// Package eventparticipants реализует независимых от users участников мероприятий.
// В существующей системе мероприятием является Contest.
package eventparticipants

import (
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("event participant not found")
	ErrForbidden          = errors.New("no access to event participants")
	ErrValidation         = errors.New("validation error")
	ErrIdentifierTaken    = errors.New("participant identifier already used")
	ErrInvalidCredentials = errors.New("invalid participant credentials")
	ErrAmbiguousIdentity  = errors.New("multiple participants match identity")
	ErrSessionExpired     = errors.New("participant session expired")
	ErrEventUnavailable   = errors.New("event is unavailable")
	ErrRateLimited        = errors.New("participant login rate limit exceeded")
	ErrDirectionNotFound  = errors.New("event direction not found")
	ErrDirectionTaken     = errors.New("event direction name already used")
	ErrDirectionInUse     = errors.New("event direction is in use")
	ErrSocialUnavailable  = errors.New("social auth is not configured")
)

const (
	StatusActive   = "ACTIVE"
	StatusBlocked  = "BLOCKED"
	StatusArchived = "ARCHIVED"

	PermissionManageParticipants = "event.participants.manage"
)

// Participant — отдельный участник одного мероприятия, не являющийся User.
type Participant struct {
	ID                 string
	ContestID          string
	FullName           string
	FullNameNormalized string
	BirthDate          time.Time
	UnionCardNumber    *string
	SKSBarcode         *string
	VKURL              *string
	TelegramURL        *string
	VKUserID           *int64
	TelegramUserID     *int64
	Status             string
	DirectionID        *string
	DirectionName      *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ArchivedAt         *time.Time
}

// Direction — направление (трек) внутри одного мероприятия.
type Direction struct {
	ID        string    `json:"id"`
	ContestID string    `json:"event_id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EventRef — минимальные данные существующего Contest для participant flow.
type EventRef struct {
	ID       string
	Slug     string
	Name     string
	Status   string
	Timezone string
}

type Actor struct {
	UserID string
	IsMega bool
}

type ListFilter struct {
	Search      string
	Status      string
	DirectionID string
	Limit       int
	Offset      int
}

type ListResult struct {
	Participants []Participant
	Total        int
	Limit        int
	Offset       int
}

type CreateInput struct {
	FullName        string
	BirthDate       time.Time
	UnionCardNumber *string
	SKSBarcode      *string
	VKURL           *string
	TelegramURL     *string
	DirectionID     *string
}

type UpdateInput = CreateInput

// ImportRecord — одна логическая строка CSV/XLSX до бизнес-валидации.
type ImportRecord struct {
	Line            int
	FullName        string
	BirthDate       string
	UnionCardNumber string
	SKSBarcode      string
	VKURL           string
	TelegramURL     string
	Direction       string
}

type ImportRowResult struct {
	Line          int    `json:"line"`
	Status        string `json:"status"` // added | updated | error | duplicate
	ParticipantID string `json:"participant_id,omitempty"`
	FullName      string `json:"full_name,omitempty"`
	Direction     string `json:"direction,omitempty"`
	Message       string `json:"message,omitempty"`
}

type ImportResult struct {
	Added      int               `json:"added"`
	Updated    int               `json:"updated"`
	Errors     int               `json:"errors"`
	Duplicates int               `json:"duplicates"`
	Rows       []ImportRowResult `json:"rows"`
}

type ExportFile struct {
	Name        string
	ContentType string
	Data        []byte
}

// ClientInfo сохраняется в participant session без хранения открытого IP.
type ClientInfo struct {
	UserAgent string
	IP        string
}

// SessionResult содержит сырой token только в момент логина; в БД хранится его hash.
type SessionResult struct {
	Token       string
	ExpiresAt   time.Time
	Participant Participant
	Event       EventRef
}

// ParticipantEventMatch — участник в конкретном активном мероприятии.
type ParticipantEventMatch struct {
	Participant Participant
	Event       EventRef
}

// SocialAuthResult — либо готовая сессия, либо список мероприятий для выбора.
type SocialAuthResult struct {
	Session       *SessionResult
	Events        []PublicEvent
	ContinueToken string
}

// Principal — аутентифицированный participant, полученный из отдельной cookie session.
type Principal struct {
	SessionID   string
	Participant Participant
	Event       EventRef
}
