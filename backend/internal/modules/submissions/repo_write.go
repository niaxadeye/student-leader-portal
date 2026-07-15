package submissions

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

// SaveAnswers обновляет черновик ответов (не трогает статус/ревизии).
func (r *Repo) SaveAnswers(ctx context.Context, id string, answers map[string]any) error {
	raw, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE submissions SET answers_json=$2, last_saved_at=now(), updated_at=now()
		WHERE id=$1`, id, raw)
	return err
}

// SubmitParams — данные для транзакции отправки.
type SubmitParams struct {
	SubmissionID   string
	Answers        map[string]any
	SchemaVersion  int
	SchemaSnapshot []byte
	FilesSnapshot  []byte
	Checksum       string
	ActionType     string
	RevisionNumber int
	ActorID        string
	ChallengeID    string
	// Outbox-событие пишется в той же транзакции, что и ревизия (SITE.md §15).
	OutboxEventType string
	OutboxPayload   []byte
}

// Submit в одной транзакции: пишет ревизию, обновляет статус/счётчики работы.
func (r *Repo) Submit(ctx context.Context, p SubmitParams) error {
	answers, err := json.Marshal(p.Answers)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO submission_revisions
			  (submission_id, revision_number, action_type, schema_version,
			   schema_snapshot, answers_snapshot, files_snapshot, checksum, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			p.SubmissionID, p.RevisionNumber, p.ActionType, p.SchemaVersion,
			p.SchemaSnapshot, answers, p.FilesSnapshot, p.Checksum, p.ActorID); err != nil {
			return err
		}
		// first submit → submitted_at; resubmit → last_resubmitted_at + version++.
		_, err := tx.Exec(ctx, `
			UPDATE submissions SET
			  status='SUBMITTED',
			  answers_json=$2,
			  current_revision_number=$3,
			  version = CASE WHEN $4='RESUBMIT' THEN version+1 ELSE version END,
			  submitted_at = COALESCE(submitted_at, now()),
			  last_resubmitted_at = CASE WHEN $4='RESUBMIT' THEN now() ELSE last_resubmitted_at END,
			  last_saved_at = now(),
			  updated_at = now()
			WHERE id=$1`,
			p.SubmissionID, answers, p.RevisionNumber, p.ActionType)
		if err != nil {
			return err
		}
		// Транзакционный outbox: событие уведомления пишется атомарно с ревизией.
		// Сбой Telegram не откатывает submit — доставка асинхронна (SITE.md §15).
		if p.OutboxEventType != "" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO outbox_events
				  (event_type, aggregate_type, aggregate_id, payload, status, available_at)
				VALUES ($1, 'submission', $2, $3, 'PENDING', now())`,
				p.OutboxEventType, p.SubmissionID, p.OutboxPayload); err != nil {
				return err
			}
		}
		return nil
	})
}

// InsertFile создаёт строку files и привязку submission_files в транзакции.
func (r *Repo) InsertFile(ctx context.Context, f *FileRow, submissionID string, fieldID *string) (string, error) {
	var id string
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			INSERT INTO files (owner_user_id, contest_id, challenge_id, submission_id, field_id,
			  bucket, object_key, original_name, safe_name, extension, mime_type, size_bytes, status, uploaded_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'READY',now())
			RETURNING id`,
			f.OwnerUserID, f.ContestID, f.ChallengeID, submissionID, fieldID,
			f.Bucket, f.ObjectKey, f.OriginalName, f.SafeName, f.Extension, f.MimeType, f.SizeBytes,
		).Scan(&id); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO submission_files (submission_id, file_id, field_id, sort_order)
			VALUES ($1,$2,$3, COALESCE((SELECT max(sort_order)+1 FROM submission_files
			         WHERE submission_id=$1 AND field_id IS NOT DISTINCT FROM $3), 0))`,
			submissionID, id, fieldID)
		return err
	})
	return id, err
}

// SoftDeleteFile помечает файл удалённым и снимает привязку (только владелец, только черновик — проверяет сервис).
func (r *Repo) SoftDeleteFile(ctx context.Context, submissionID, fileID string) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`DELETE FROM submission_files WHERE submission_id=$1 AND file_id=$2`,
			submissionID, fileID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`UPDATE files SET status='DELETED', deleted_at=now(), updated_at=now() WHERE id=$1`, fileID)
		return err
	})
}

// FileRow — метаданные загружаемого файла.
type FileRow struct {
	OwnerUserID  string
	ContestID    string
	ChallengeID  string
	Bucket       string
	ObjectKey    string
	OriginalName string
	SafeName     string
	Extension    *string
	MimeType     *string
	SizeBytes    int64
}

// FileByID возвращает object_key и владельца файла (для скачивания/удаления).
func (r *Repo) FileByID(ctx context.Context, fileID string) (ownerID, objectKey string, err error) {
	err = r.pool.QueryRow(ctx,
		`SELECT owner_user_id, object_key FROM files WHERE id=$1 AND deleted_at IS NULL`,
		fileID).Scan(&ownerID, &objectKey)
	return
}

// Revisions возвращает историю ревизий работы (для админ-карточки), новейшие сверху.
func (r *Repo) Revisions(ctx context.Context, submissionID string) ([]Revision, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, revision_number, action_type, schema_version, checksum, created_at,
		       answers_snapshot, files_snapshot
		FROM submission_revisions WHERE submission_id=$1
		ORDER BY revision_number DESC`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Revision
	for rows.Next() {
		var rev Revision
		var files []map[string]any
		if err := rows.Scan(&rev.ID, &rev.RevisionNumber, &rev.ActionType, &rev.SchemaVersion,
			&rev.Checksum, &rev.CreatedAt, &rev.AnswersSnapshot, &files); err != nil {
			return nil, err
		}
		rev.FilesSnapshot = files
		out = append(out, rev)
	}
	return out, rows.Err()
}
