// Package submissions реализует подачу ответов конкурсантом: черновики,
// immutable-ревизии, файлы (SITE.md §7.3–7.6, §21.11–21.14, Этап 4).
package submissions

import (
	"context"
	"errors"
	"time"
)

// Доменные ошибки — маппятся на error codes API (SITE.md §50).
var (
	ErrNotFound   = errors.New("submission not found")
	ErrForbidden  = errors.New("no access")
	ErrValidation = errors.New("validation error")
	ErrClosed     = errors.New("submission window closed")
	ErrLocked     = errors.New("submission locked")
	ErrDeadline   = errors.New("deadline passed")
)

// Статусы работы (SITE.md §8 «Форма»).
const (
	StatusDraft     = "DRAFT"
	StatusSubmitted = "SUBMITTED"
	StatusLocked    = "LOCKED"
)

// Типы ревизий.
const (
	ActionSubmit   = "SUBMIT"
	ActionResubmit = "RESUBMIT"
)

// Actor — субъект операции.
type Actor struct {
	UserID  string
	IsSuper bool
}

// Submission — работа конкурсанта по испытанию.
type Submission struct {
	ID                    string
	ChallengeID           string
	ContestantUserID      string
	Status                string
	Answers               map[string]any
	SchemaVersion         int
	Version               int
	CurrentRevisionNumber int
	FirstOpenedAt         *time.Time
	LastSavedAt           *time.Time
	SubmittedAt           *time.Time
	LastResubmittedAt     *time.Time
	LockedAt              *time.Time
	LockReason            *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	// Присоединяемые поля (не в таблице submissions):
	Files       []SubmissionFile
	ContestName string
	FullName    string
	Login       string
	Organization *string
}

// SubmissionFile — файл, привязанный к работе и полю.
type SubmissionFile struct {
	FileID       string
	FieldID      *string
	FieldKey     string
	OriginalName string
	SizeBytes    *int64
	MimeType     *string
	SortOrder    int
	// DownloadURL заполняется presigned-ссылкой при отдаче наружу.
	DownloadURL string
}

// Revision — immutable-снимок отправки.
type Revision struct {
	ID              string
	RevisionNumber  int
	ActionType      string
	SchemaVersion   int
	Checksum        string
	CreatedAt       time.Time
	AnswersSnapshot map[string]any
	FilesSnapshot   []map[string]any
}

// ChallengeInfo — то, что submissions читает об испытании (реализует challenges.Repo-адаптер).
type ChallengeInfo struct {
	ID                   string
	ContestID            string
	Status               string
	OpenAt               *time.Time
	DeadlineAt           *time.Time
	CloseAt              *time.Time
	CurrentSchemaVersion int
	Settings             map[string]any
}

// FieldInfo — поле схемы, нужное для валидации и снапшота.
type FieldInfo struct {
	ID       string
	Key      string
	Type     string
	Label    string
	Required bool
	Settings map[string]any
}

// ChallengeSource — зависимость от модуля challenges (чтение испытания/схемы/участия).
type ChallengeSource interface {
	ChallengeInfo(ctx context.Context, challengeID string) (*ChallengeInfo, error)
	SchemaFields(ctx context.Context, challengeID string) ([]FieldInfo, error)
	IsParticipant(ctx context.Context, userID, contestID string) (bool, error)
	HasContestAccess(ctx context.Context, userID, contestID string, isSuper bool) (bool, error)
}

// Auditor пишет события аудита.
type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}
