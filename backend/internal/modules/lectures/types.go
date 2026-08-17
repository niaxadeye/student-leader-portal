// Package lectures implements event lectures, participant QR codes and attendance.
package lectures

import (
	"errors"
	"time"
)

var (
	ErrNotFound             = errors.New("lecture not found")
	ErrForbidden            = errors.New("no access to event attendance")
	ErrValidation           = errors.New("lecture validation error")
	ErrInvalidTransition    = errors.New("invalid lecture status transition")
	ErrAttendanceClosed     = errors.New("lecture attendance is closed")
	ErrInvalidCode          = errors.New("participant qr code is invalid")
	ErrExpiredCode          = errors.New("participant qr code expired")
	ErrReplayedCode         = errors.New("participant qr code already used")
	ErrParticipantInactive  = errors.New("participant is unavailable")
	ErrLectureHasAttendance = errors.New("lecture with attendance cannot be deleted")
)

const (
	StatusDraft    = "DRAFT"
	StatusActive   = "ACTIVE"
	StatusFinished = "FINISHED"

	ScannerCamera = "CAMERA"
	ScannerUSB    = "USB"
	ScannerManual = "MANUAL"

	PermissionManage = "event.attendance.manage"
	PermissionScan   = "event.attendance.scan"
)

type Actor struct {
	UserID string
	IsMega bool
}

type Lecture struct {
	ID                 string     `json:"id"`
	ContestID          string     `json:"event_id"`
	Title              string     `json:"title"`
	Description        *string    `json:"description"`
	Points             int64      `json:"points"`
	StartsAt           *time.Time `json:"starts_at"`
	EndsAt             *time.Time `json:"ends_at"`
	AttendanceStartsAt *time.Time `json:"attendance_starts_at"`
	AttendanceEndsAt   *time.Time `json:"attendance_ends_at"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type LectureInput struct {
	Title              string     `json:"title"`
	Description        *string    `json:"description"`
	Points             int64      `json:"points"`
	StartsAt           *time.Time `json:"starts_at"`
	EndsAt             *time.Time `json:"ends_at"`
	AttendanceStartsAt *time.Time `json:"attendance_starts_at"`
	AttendanceEndsAt   *time.Time `json:"attendance_ends_at"`
}

type Attendance struct {
	ID                 string    `json:"id"`
	ContestID          string    `json:"event_id"`
	LectureID          string    `json:"lecture_id"`
	EventParticipantID string    `json:"participant_id"`
	ParticipantName    string    `json:"participant_name"`
	ScannedByUserID    string    `json:"scanned_by_user_id"`
	ScannerType        string    `json:"scanner_type"`
	PointsAwarded      int64     `json:"points_awarded"`
	CreatedAt          time.Time `json:"created_at"`
}

type ScanInput struct {
	Token       string `json:"token"`
	ScannerType string `json:"scanner_type"`
}

type ScanResult struct {
	Attendance     Attendance `json:"attendance"`
	AlreadyChecked bool       `json:"already_checked"`
}

type QRCode struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	TTL       int       `json:"ttl_seconds"`
}

type ParticipantLecture struct {
	Lecture    Lecture     `json:"lecture"`
	Attendance *Attendance `json:"attendance"`
}

type VerifiedCode struct {
	NonceHash string
	ExpiresAt time.Time
}

type ScanParams struct {
	Actor       Actor
	ContestID   string
	LectureID   string
	ScannerType string
	Code        VerifiedCode
	Now         time.Time
}
