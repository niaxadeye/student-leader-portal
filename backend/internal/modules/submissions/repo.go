package submissions

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const subCols = `s.id, s.challenge_id, s.contestant_user_id, s.status, s.answers_json,
	s.schema_version, s.version, s.current_revision_number, s.first_opened_at,
	s.last_saved_at, s.submitted_at, s.last_resubmitted_at, s.locked_at, s.lock_reason,
	s.created_at, s.updated_at`

func scanSub(row pgx.Row) (*Submission, error) {
	var s Submission
	err := row.Scan(&s.ID, &s.ChallengeID, &s.ContestantUserID, &s.Status, &s.Answers,
		&s.SchemaVersion, &s.Version, &s.CurrentRevisionNumber, &s.FirstOpenedAt,
		&s.LastSavedAt, &s.SubmittedAt, &s.LastResubmittedAt, &s.LockedAt, &s.LockReason,
		&s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ByChallengeAndUser находит работу конкурсанта по испытанию (без файлов).
func (r *Repo) ByChallengeAndUser(ctx context.Context, challengeID, userID string) (*Submission, error) {
	return scanSub(r.pool.QueryRow(ctx,
		`SELECT `+subCols+` FROM submissions s
		 WHERE s.challenge_id=$1 AND s.contestant_user_id=$2`, challengeID, userID))
}

// ByID находит работу по id (для админ-карточки).
func (r *Repo) ByID(ctx context.Context, id string) (*Submission, error) {
	return scanSub(r.pool.QueryRow(ctx,
		`SELECT `+subCols+` FROM submissions s WHERE s.id=$1`, id))
}

// EnsureDraft возвращает существующую работу или создаёт пустой черновик.
// Проставляет first_opened_at при первом открытии.
func (r *Repo) EnsureDraft(ctx context.Context, challengeID, userID string, schemaVersion int) (*Submission, error) {
	sub, err := r.ByChallengeAndUser(ctx, challengeID, userID)
	if err == nil {
		if sub.FirstOpenedAt == nil {
			_, _ = r.pool.Exec(ctx,
				`UPDATE submissions SET first_opened_at=now() WHERE id=$1`, sub.ID)
		}
		return sub, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	// Создаём черновик. ON CONFLICT — на случай гонки двух вкладок.
	_, err = r.pool.Exec(ctx, `
		INSERT INTO submissions (challenge_id, contestant_user_id, status, schema_version, first_opened_at)
		VALUES ($1, $2, 'DRAFT', $3, now())
		ON CONFLICT (challenge_id, contestant_user_id) DO NOTHING`,
		challengeID, userID, schemaVersion)
	if err != nil {
		return nil, err
	}
	return r.ByChallengeAndUser(ctx, challengeID, userID)
}

// LoadFiles присоединяет файлы работы (с ключом поля и метаданными).
func (r *Repo) LoadFiles(ctx context.Context, submissionID string) ([]SubmissionFile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sf.file_id, sf.field_id, sf.sort_order,
		       f.original_name, f.size_bytes, f.mime_type, f.object_key
		FROM submission_files sf
		JOIN files f ON f.id = sf.file_id AND f.deleted_at IS NULL
		WHERE sf.submission_id=$1
		ORDER BY sf.field_id, sf.sort_order`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubmissionFile
	for rows.Next() {
		var f SubmissionFile
		var objectKey string
		if err := rows.Scan(&f.FileID, &f.FieldID, &f.SortOrder,
			&f.OriginalName, &f.SizeBytes, &f.MimeType, &objectKey); err != nil {
			return nil, err
		}
		// objectKey держим в DownloadURL временно — сервис заменит на presigned.
		f.DownloadURL = objectKey
		out = append(out, f)
	}
	return out, rows.Err()
}
