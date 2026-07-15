// Package outbox реализует транзакционный outbox для внешних уведомлений
// (Telegram при отправке формы) — SITE.md §15, §21.16.
package outbox

import "errors"

// Типы событий.
const (
	EventSubmissionSubmitted   = "submission.submitted"
	EventSubmissionResubmitted = "submission.resubmitted"
)

// ErrUnknownEvent — тип события не поддерживается диспетчером (помечаем DEAD, не ретраим).
var ErrUnknownEvent = errors.New("unknown event type")

// SubmissionPayload — тело события отправки формы (минимум; остальное дорезолвит диспетчер).
type SubmissionPayload struct {
	SubmissionID string `json:"submission_id"`
	Revision     int    `json:"revision"`
	Action       string `json:"action"`
}

// Статусы события.
const (
	StatusPending = "PENDING"
	StatusSent    = "SENT"
	StatusDead    = "DEAD"
)

// Event — строка outbox для обработки диспетчером.
type Event struct {
	ID            string
	EventType     string
	AggregateType string
	AggregateID   string
	Payload       []byte
	Attempts      int
}
