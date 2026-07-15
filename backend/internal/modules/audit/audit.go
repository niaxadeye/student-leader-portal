// Package audit — append-only журнал действий (SITE.md §21.17).
package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func New(pool *pgxpool.Pool, log *slog.Logger) *Service {
	return &Service{pool: pool, log: log}
}

// Log пишет событие. Аудит не должен ронять основную операцию — ошибки только логируются.
func (s *Service) Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any) {
	var metaJSON []byte
	if meta != nil {
		metaJSON, _ = json.Marshal(meta)
	} else {
		metaJSON = []byte("{}")
	}
	var actor, entity any
	if actorUserID != "" {
		actor = actorUserID
	}
	if entityID != "" {
		entity = entityID
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, action, entity_type, entity_id, metadata)
		VALUES ($1,$2,$3,$4,$5)`, actor, action, entityType, entity, metaJSON)
	if err != nil {
		s.log.Error("audit write failed", "action", action, "error", err)
	}
}
