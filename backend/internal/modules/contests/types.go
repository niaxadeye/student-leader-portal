// Package contests реализует CRUD конкурсов и управление участниками (SITE.md §9, §21.6–21.7).
package contests

import (
	"errors"
	"time"
)

// Доменные ошибки — маппятся на error codes API (SITE.md §50).
var (
	ErrNotFound     = errors.New("contest not found")
	ErrForbidden    = errors.New("no access to contest")
	ErrSlugTaken    = errors.New("slug already taken")
	ErrValidation   = errors.New("validation error")
	ErrBadStatus    = errors.New("invalid status transition")
	ErrUserNotFound = errors.New("user not found")
)

// Статусы конкурса (SITE.md §9).
const (
	StatusDraft    = "DRAFT"
	StatusActive   = "ACTIVE"
	StatusFinished = "FINISHED"
	StatusArchived = "ARCHIVED"
)

// Типы участников (SITE.md §21.7).
const (
	ParticipantContestant  = "CONTESTANT"
	ParticipantParticipant = "PARTICIPANT"
	ParticipantStaff       = "STAFF"
	ParticipantJury        = "JURY"
)

// Contest — конкурс с агрегатами для списков.
type Contest struct {
	ID          string
	Name        string
	Slug        string
	Description *string
	Status      string
	StartAt     *time.Time
	EndAt       *time.Time
	Timezone    string
	ImageKey    *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ArchivedAt  *time.Time
	// Агрегаты (заполняются в списках).
	ParticipantsCount int
	ChallengesCount   int
}

// Participant — строка участника с данными пользователя для отображения.
type Participant struct {
	ID              string
	ContestID       string
	UserID          string
	ParticipantType string
	Login           string
	FullName        string
	Organization    *string
	UserStatus      string
	JoinedAt        time.Time
	LeftAt          *time.Time
}
