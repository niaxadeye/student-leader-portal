package challenges

import (
	"context"
	"encoding/json"
)

// CreateField добавляет поле в конец списка испытания текущей версии схемы.
func (r *Repo) CreateField(ctx context.Context, challengeID string, f *Field, actorID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO challenge_fields
		  (challenge_id, field_key, field_type, label, description, help_text, placeholder,
		   required, sort_order, settings, validation, visibility, schema_version_from,
		   created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,
		  (SELECT coalesce(max(sort_order),0)+1 FROM challenge_fields
		     WHERE challenge_id=$1 AND deleted_at IS NULL AND schema_version_to IS NULL),
		  $9,$10,$11,
		  (SELECT current_schema_version FROM contest_challenges WHERE id=$1),
		  $12,$12)
		RETURNING id`,
		challengeID, f.Key, f.Type, f.Label, f.Description, f.HelpText, f.Placeholder,
		f.Required, jsonOr(f.Settings), jsonOr(f.Validation), jsonOr(f.Visibility), actorID).Scan(&id)
	if isUniqueViolation(err) {
		return "", ErrFieldKey
	}
	return id, err
}

// UpdateField меняет поле (ключ и тип тоже, валидация типа — в сервисе).
func (r *Repo) UpdateField(ctx context.Context, challengeID, fieldID string, f *Field, actorID string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE challenge_fields SET field_key=$3, field_type=$4, label=$5, description=$6,
		       help_text=$7, placeholder=$8, required=$9, settings=$10, validation=$11,
		       visibility=$12, updated_by=$13, updated_at=now()
		WHERE id=$2 AND challenge_id=$1 AND deleted_at IS NULL`,
		challengeID, fieldID, f.Key, f.Type, f.Label, f.Description, f.HelpText,
		f.Placeholder, f.Required, jsonOr(f.Settings), jsonOr(f.Validation),
		jsonOr(f.Visibility), actorID)
	if isUniqueViolation(err) {
		return ErrFieldKey
	}
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteField — soft delete (ответы прошлых ревизий ссылаются на поле, SITE.md §11.4).
func (r *Repo) DeleteField(ctx context.Context, challengeID, fieldID, actorID string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE challenge_fields SET deleted_at=now(), updated_by=$3, updated_at=now()
		WHERE id=$2 AND challenge_id=$1 AND deleted_at IS NULL`, challengeID, fieldID, actorID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReorderFields выставляет sort_order по позиции в переданном списке id (в транзакции).
func (r *Repo) ReorderFields(ctx context.Context, challengeID string, orderedIDs []string, actorID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, id := range orderedIDs {
		ct, err := tx.Exec(ctx, `
			UPDATE challenge_fields SET sort_order=$3, updated_by=$4, updated_at=now()
			WHERE id=$2 AND challenge_id=$1 AND deleted_at IS NULL`,
			challengeID, id, i+1, actorID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return ErrNotFound
		}
	}
	return tx.Commit(ctx)
}

// BumpSchemaVersion увеличивает версию схемы испытания и возвращает новую.
func (r *Repo) BumpSchemaVersion(ctx context.Context, challengeID, actorID string) (int, error) {
	var v int
	err := r.pool.QueryRow(ctx, `
		UPDATE contest_challenges SET current_schema_version=current_schema_version+1,
		       updated_by=$2, updated_at=now()
		WHERE id=$1 RETURNING current_schema_version`, challengeID, actorID).Scan(&v)
	return v, err
}

// SaveSnapshot фиксирует схему испытания как версию (идемпотентно по (challenge, version)).
func (r *Repo) SaveSnapshot(ctx context.Context, challengeID string, version int, schema []byte, summary, actorID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO challenge_schema_versions (challenge_id, version, schema_json, change_summary, created_by)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (challenge_id, version) DO NOTHING`,
		challengeID, version, schema, nilIfEmpty(summary), actorID)
	return err
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func jsonOr(m map[string]any) []byte {
	if m == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return b
}
