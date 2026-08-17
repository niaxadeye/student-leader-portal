// Package audit — append-only журнал действий (SITE.md §21.17).
package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// Entry поддерживает как staff actor, так и отдельного участника мероприятия.
// Старый метод Log сохранён для обратной совместимости существующих модулей.
type Entry struct {
	ActorUserID        string
	EventParticipantID string
	ContestID          string
	Action             string
	EntityType         string
	EntityID           string
	Metadata           map[string]any
}

func New(pool *pgxpool.Pool, log *slog.Logger) *Service {
	return &Service{pool: pool, log: log}
}

// Log пишет событие. Аудит не должен ронять основную операцию — ошибки только логируются.
func (s *Service) Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any) {
	s.LogEntry(ctx, Entry{
		ActorUserID: actorUserID, Action: action, EntityType: entityType,
		EntityID: entityID, Metadata: meta,
	})
}

// LogParticipant пишет событие от имени participant, не создавая фиктивного User.
func (s *Service) LogParticipant(
	ctx context.Context,
	participantID, contestID, action, entityType, entityID string,
	meta map[string]any,
) {
	s.LogEntry(ctx, Entry{
		EventParticipantID: participantID, ContestID: contestID,
		Action: action, EntityType: entityType, EntityID: entityID, Metadata: meta,
	})
}

// LogEntry пишет событие best-effort для обычных операций.
func (s *Service) LogEntry(ctx context.Context, entry Entry) {
	if err := writeEntry(ctx, s.pool, entry); err != nil {
		s.log.Error("audit write failed", "action", entry.Action, "error", err)
	}
}

// LogEntryTx позволяет финансовым операциям создавать audit в той же транзакции.
// Ошибка возвращается вызывающему коду и приводит к rollback бизнес-операции.
func (s *Service) LogEntryTx(ctx context.Context, tx pgx.Tx, entry Entry) error {
	return writeEntry(ctx, tx, entry)
}

type executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func writeEntry(ctx context.Context, exec executor, entry Entry) error {
	var metaJSON []byte
	if entry.Metadata != nil {
		metaJSON, _ = json.Marshal(entry.Metadata)
	} else {
		metaJSON = []byte("{}")
	}
	var actor, participant, contest, entity any
	if entry.ActorUserID != "" {
		actor = entry.ActorUserID
	}
	if entry.EventParticipantID != "" {
		participant = entry.EventParticipantID
	}
	if entry.ContestID != "" {
		contest = entry.ContestID
	}
	if entry.EntityID != "" {
		entity = entry.EntityID
	}
	_, err := exec.Exec(ctx, `
		INSERT INTO audit_logs
		  (actor_user_id, event_participant_id, contest_id, action, entity_type, entity_id, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, actor, participant, contest,
		entry.Action, entry.EntityType, entity, metaJSON)
	return err
}
