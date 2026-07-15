// Package challenges реализует испытания и конструктор полей (SITE.md §10–12, §21.8–21.10).
package challenges

import (
	"errors"
	"time"
)

// Доменные ошибки — маппятся на error codes API (SITE.md §50).
var (
	ErrNotFound   = errors.New("challenge not found")
	ErrForbidden  = errors.New("no access to challenge")
	ErrSlugTaken  = errors.New("slug already taken")
	ErrValidation = errors.New("validation error")
	ErrBadStatus  = errors.New("invalid status transition")
	ErrFieldKey   = errors.New("field key already exists")
)

// Статусы испытания (SITE.md §«Испытание»).
const (
	StatusDraft     = "DRAFT"
	StatusPublished = "PUBLISHED"
	StatusClosed    = "CLOSED"
	StatusArchived  = "ARCHIVED"
)

// ValidFieldTypes — набор типов полей v1 (подмножество SITE.md §11.1).
// RICH_TEXT/TIME/DATETIME/MULTISELECT/FILE отложены (долг Этапа 3).
var ValidFieldTypes = map[string]bool{
	"SHORT_TEXT": true, "LONG_TEXT": true, "NUMBER": true, "URL": true,
	"EMAIL": true, "PHONE": true, "DATE": true, "SELECT": true, "RADIO": true,
	"CHECKBOX": true, "FILE_GROUP": true, "SECTION": true, "INFO_BLOCK": true,
}

// Challenge — испытание с агрегатом числа полей для списков.
type Challenge struct {
	ID                   string
	ContestID            string
	Title                string
	Slug                 string
	ShortDescription     *string
	FullDescription      *string
	Instructions         *string
	Status               string
	SortOrder            int
	OpenAt               *time.Time
	DeadlineAt           *time.Time
	CloseAt              *time.Time
	Settings             map[string]any
	CurrentSchemaVersion int
	CreatedAt            time.Time
	UpdatedAt            time.Time
	PublishedAt          *time.Time
	ArchivedAt           *time.Time
	FieldsCount          int
	// MySubmissionStatus — статус работы текущего конкурсанта (NOT_STARTED, если нет). Транзиентное.
	MySubmissionStatus string
}

// Field — поле конструктора (SITE.md §11.3).
type Field struct {
	ID          string
	ChallengeID string
	Key         string
	Type        string
	Label       string
	Description *string
	HelpText    *string
	Placeholder *string
	Required    bool
	SortOrder   int
	Settings    map[string]any
	Validation  map[string]any
	Visibility  map[string]any
}
