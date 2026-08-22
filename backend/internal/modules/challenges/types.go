// Package challenges реализует испытания и конструктор полей (SITE.md §10–12, §21.8–21.10).
package challenges

import (
	"errors"
	"io"
	"time"
)

// Доменные ошибки — маппятся на error codes API (SITE.md §50).
var (
	ErrNotFound     = errors.New("challenge not found")
	ErrForbidden    = errors.New("no access to challenge")
	ErrSlugTaken    = errors.New("slug already taken")
	ErrValidation   = errors.New("validation error")
	ErrBadStatus    = errors.New("invalid status transition")
	ErrFieldKey     = errors.New("field key already exists")
	ErrNoStorage    = errors.New("file storage unavailable")
	ErrBriefingFile = errors.New("briefing file rejected")
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
	HeldAt               *time.Time
	Venue                *string
	AcceptsSubmissions   bool
	Settings             map[string]any
	CurrentSchemaVersion int
	CreatedAt            time.Time
	UpdatedAt            time.Time
	PublishedAt          *time.Time
	ArchivedAt           *time.Time
	FieldsCount          int
	// MySubmissionStatus — статус работы текущего конкурсанта (NOT_STARTED, если нет). Транзиентное.
	MySubmissionStatus string
	// SchemeType / LiveState — сводка оценивания для админ-списка (подзапросы, могут быть nil).
	SchemeType *string
	LiveState  *string
	// Briefing — сводка материалов для кабинета конкурсанта (заполняется отдельно).
	Briefing *ResolvedBriefing
}

const (
	MaxBriefingTextRunes = 20000
	MaxBriefingFiles     = 15
)

type BriefingFile struct {
	FileID       string
	OriginalName string
	SizeBytes    *int64
	MimeType     *string
	DownloadURL  *string
	ObjectKey    string
}

type Briefing struct {
	ChallengeID string
	BodyText    string
	PublishAt   *time.Time
	Files       []BriefingFile
	UpdatedAt   time.Time
}

type BriefingOverride struct {
	ID               string
	ChallengeID      string
	ContestantUserID string
	CustomText       bool
	BodyText         string
	CustomPublish    bool
	PublishAt        *time.Time
	Hidden           bool
	ReplaceFiles     bool
	Files            []BriefingFile
	UpdatedAt        time.Time
}

type BriefingContestant struct {
	UserID       string
	Login        string
	FullName     string
	Organization *string
	Override     *BriefingOverride
	Visible      bool
	PublishAt    *time.Time
	Personalized bool
}

type ResolvedBriefing struct {
	Visible      bool
	Scheduled    bool
	Hidden       bool
	Personalized bool
	PublishAt    *time.Time
	BodyText     string
	Files        []BriefingFile
}

type BriefingInput struct {
	BodyText  string
	PublishAt *time.Time
}

type OverrideInput struct {
	CustomText    bool
	BodyText      string
	CustomPublish bool
	PublishAt     *time.Time
	Hidden        bool
	ReplaceFiles  bool
}

type BriefingUpload struct {
	OriginalName string
	ContentType  string
	Size         int64
	Reader       io.Reader
	KeySuffix    string
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
