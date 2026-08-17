// Package points реализует append-only журнал баллов участников мероприятий.
package points

import (
	"errors"
	"time"
)

var (
	ErrNotFound            = errors.New("points participant not found")
	ErrForbidden           = errors.New("no access to event points")
	ErrValidation          = errors.New("points validation error")
	ErrIdempotencyConflict = errors.New("idempotency key already used for another operation")
)

const (
	TypeLectureAttendance = "LECTURE_ATTENDANCE"
	TypeTaskReward        = "TASK_REWARD"
	TypeMerchPurchase     = "MERCH_PURCHASE"
	TypeAdminAdjustment   = "ADMIN_ADJUSTMENT"
	TypeRefund            = "REFUND"

	PermissionManagePoints = "event.points.manage"
)

type Actor struct {
	UserID string
	IsMega bool
}

type Entry struct {
	ID                 string    `json:"id"`
	ContestID          string    `json:"event_id"`
	EventParticipantID string    `json:"participant_id"`
	Amount             int64     `json:"amount"`
	Type               string    `json:"type"`
	SourceType         *string   `json:"source_type"`
	SourceID           *string   `json:"source_id"`
	Description        string    `json:"description"`
	CreatedByUserID    *string   `json:"-"`
	IdempotencyKey     string    `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
}

type Balance struct {
	LedgerBalance   int64 `json:"ledger_balance"`
	ReservedPoints  int64 `json:"reserved_points"`
	AvailablePoints int64 `json:"available_points"`
}

type Overview struct {
	Balance Balance `json:"balance"`
	Entries []Entry `json:"entries"`
	Total   int     `json:"-"`
	Limit   int     `json:"-"`
	Offset  int     `json:"-"`
}

// AppendInput — универсальная операция для следующих модулей attendance/tasks/merch.
type AppendInput struct {
	ContestID          string
	EventParticipantID string
	Amount             int64
	Type               string
	SourceType         *string
	SourceID           *string
	Description        string
	CreatedByUserID    *string
	IdempotencyKey     string
}

type AdjustmentInput struct {
	Amount         int64
	Reason         string
	IdempotencyKey string
}

type AdjustmentResult struct {
	Entry    Entry   `json:"entry"`
	Balance  Balance `json:"balance"`
	Replayed bool    `json:"replayed"`
}
