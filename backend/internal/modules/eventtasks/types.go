// Package eventtasks implements participant tasks, evidence attempts and moderation.
package eventtasks

import (
	"errors"
	"io"
	"time"
)

var (
	ErrNotFound          = errors.New("event task not found")
	ErrForbidden         = errors.New("no access to event tasks")
	ErrValidation        = errors.New("event task validation error")
	ErrInvalidTransition = errors.New("invalid task or submission transition")
	ErrSubmissionClosed  = errors.New("task submission window closed")
	ErrStorageDisabled   = errors.New("task image storage unavailable")
)

const (
	StatusDraft    = "DRAFT"
	StatusActive   = "ACTIVE"
	StatusDisabled = "DISABLED"
	StatusArchived = "ARCHIVED"

	SubmissionPending  = "PENDING"
	SubmissionApproved = "APPROVED"
	SubmissionRejected = "REJECTED"

	AssetImage = "IMAGE"
	AssetLink  = "LINK"

	PermissionManage   = "event.tasks.manage"
	PermissionModerate = "event.tasks.moderate"
)

type Actor struct {
	UserID string
	IsMega bool
}

type Task struct {
	ID                     string      `json:"id"`
	ContestID              string      `json:"event_id"`
	Title                  string      `json:"title"`
	Description            string      `json:"description"`
	ImageKey               *string     `json:"-"`
	ImageURL               *string     `json:"image_url"`
	Icon                   *string     `json:"icon"`
	Points                 int64       `json:"points"`
	StartsAt               *time.Time  `json:"starts_at"`
	EndsAt                 *time.Time  `json:"ends_at"`
	Status                 string      `json:"status"`
	SortOrder              int         `json:"sort_order"`
	AllowedSubmissionTypes []string    `json:"allowed_submission_types"`
	CreatedAt              time.Time   `json:"created_at"`
	UpdatedAt              time.Time   `json:"updated_at"`
	Available              bool        `json:"available"`
	Submission             *Submission `json:"submission,omitempty"`
}

type TaskInput struct {
	Title                  string     `json:"title"`
	Description            string     `json:"description"`
	Icon                   *string    `json:"icon"`
	Points                 int64      `json:"points"`
	StartsAt               *time.Time `json:"starts_at"`
	EndsAt                 *time.Time `json:"ends_at"`
	SortOrder              int        `json:"sort_order"`
	AllowedSubmissionTypes []string   `json:"allowed_submission_types"`
}

type Submission struct {
	ID                     string     `json:"id"`
	ContestID              string     `json:"event_id"`
	TaskID                 string     `json:"task_id"`
	EventParticipantID     string     `json:"participant_id"`
	ParticipantName        string     `json:"participant_name,omitempty"`
	TaskTitle              string     `json:"task_title,omitempty"`
	Status                 string     `json:"status"`
	CurrentAttempt         int        `json:"current_attempt"`
	ParticipantComment     *string    `json:"participant_comment"`
	ModeratorComment       *string    `json:"moderator_comment"`
	ReviewedByUserID       *string    `json:"reviewed_by_user_id"`
	SubmittedAt            *time.Time `json:"submitted_at"`
	ReviewedAt             *time.Time `json:"reviewed_at"`
	RewardGrantedAt        *time.Time `json:"reward_granted_at"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	Points                 int64      `json:"points"`
	AllowedSubmissionTypes []string   `json:"allowed_submission_types,omitempty"`
	Attempts               []Attempt  `json:"attempts,omitempty"`
}

type Attempt struct {
	ID                 string     `json:"id"`
	AttemptNumber      int        `json:"attempt_number"`
	Status             string     `json:"status"`
	ParticipantComment *string    `json:"participant_comment"`
	ModeratorComment   *string    `json:"moderator_comment"`
	ReviewedByUserID   *string    `json:"reviewed_by_user_id"`
	SubmittedAt        time.Time  `json:"submitted_at"`
	ReviewedAt         *time.Time `json:"reviewed_at"`
	Assets             []Asset    `json:"assets"`
}

type Asset struct {
	ID           string    `json:"id"`
	AttemptID    string    `json:"attempt_id"`
	Type         string    `json:"type"`
	ObjectKey    *string   `json:"-"`
	ExternalURL  *string   `json:"url,omitempty"`
	OriginalName *string   `json:"original_name,omitempty"`
	MimeType     *string   `json:"mime_type,omitempty"`
	SizeBytes    *int64    `json:"size_bytes,omitempty"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	DownloadPath *string   `json:"download_path,omitempty"`
}

type ImageUpload struct {
	OriginalName string
	ContentType  string
	Size         int64
	Reader       io.Reader
	KeySuffix    string
}

type SubmitInput struct {
	ParticipantComment string
	Links              []string
	Images             []ImageUpload
}

type StoredAsset struct {
	Type         string
	ObjectKey    *string
	ExternalURL  *string
	OriginalName *string
	MimeType     *string
	SizeBytes    *int64
	SortOrder    int
}

type SubmitParams struct {
	ContestID          string
	TaskID             string
	EventParticipantID string
	ParticipantComment *string
	Assets             []StoredAsset
	Now                time.Time
}

type ModerationInput struct {
	Comment string `json:"comment"`
}

type ModerationResult struct {
	Submission Submission `json:"submission"`
	Replayed   bool       `json:"replayed"`
}
