package challenges

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repo) GetBriefing(ctx context.Context, challengeID string) (*Briefing, error) {
	b := Briefing{ChallengeID: challengeID, Files: []BriefingFile{}}
	err := r.pool.QueryRow(ctx, `
		SELECT body_text, publish_at, updated_at
		FROM challenge_briefings WHERE challenge_id=$1`, challengeID).Scan(&b.BodyText, &b.PublishAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return &b, nil
	}
	if err != nil {
		return nil, err
	}
	files, err := r.listBriefingFiles(ctx, challengeID, nil)
	if err != nil {
		return nil, err
	}
	b.Files = files
	return &b, nil
}

func (r *Repo) UpsertBriefing(ctx context.Context, challengeID, actorID, body string, publishAt *time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO challenge_briefings (challenge_id, body_text, publish_at, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$4)
		ON CONFLICT (challenge_id) DO UPDATE SET
			body_text = EXCLUDED.body_text,
			publish_at = EXCLUDED.publish_at,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()`,
		challengeID, body, publishAt, nullIfEmpty(actorID))
	return err
}

func (r *Repo) GetOverride(ctx context.Context, challengeID, userID string) (*BriefingOverride, error) {
	var o BriefingOverride
	err := r.pool.QueryRow(ctx, `
		SELECT id, challenge_id, contestant_user_id, custom_text, body_text,
		       custom_publish, publish_at, hidden, replace_files, updated_at
		FROM challenge_briefing_overrides
		WHERE challenge_id=$1 AND contestant_user_id=$2`, challengeID, userID).Scan(
		&o.ID, &o.ChallengeID, &o.ContestantUserID, &o.CustomText, &o.BodyText,
		&o.CustomPublish, &o.PublishAt, &o.Hidden, &o.ReplaceFiles, &o.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	files, err := r.listBriefingFiles(ctx, challengeID, &o.ID)
	if err != nil {
		return nil, err
	}
	o.Files = files
	return &o, nil
}

func (r *Repo) UpsertOverride(ctx context.Context, challengeID, userID, actorID string, in OverrideInput) (*BriefingOverride, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO challenge_briefing_overrides (
			challenge_id, contestant_user_id, custom_text, body_text,
			custom_publish, publish_at, hidden, replace_files, created_by, updated_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)
		ON CONFLICT (challenge_id, contestant_user_id) DO UPDATE SET
			custom_text = EXCLUDED.custom_text,
			body_text = EXCLUDED.body_text,
			custom_publish = EXCLUDED.custom_publish,
			publish_at = EXCLUDED.publish_at,
			hidden = EXCLUDED.hidden,
			replace_files = EXCLUDED.replace_files,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()
		RETURNING id`,
		challengeID, userID, in.CustomText, in.BodyText, in.CustomPublish, in.PublishAt,
		in.Hidden, in.ReplaceFiles, nullIfEmpty(actorID),
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetOverride(ctx, challengeID, userID)
}

func (r *Repo) DeleteOverride(ctx context.Context, challengeID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM challenge_briefing_overrides
		WHERE challenge_id=$1 AND contestant_user_id=$2`, challengeID, userID)
	return err
}

func (r *Repo) ListBriefingContestants(ctx context.Context, contestID, challengeID string) ([]BriefingContestant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.login, u.full_name, u.organization,
		       o.id, o.custom_text, o.body_text, o.custom_publish, o.publish_at,
		       o.hidden, o.replace_files, o.updated_at
		FROM contest_participants cp
		JOIN users u ON u.id = cp.user_id
		LEFT JOIN challenge_briefing_overrides o
		  ON o.challenge_id = $2 AND o.contestant_user_id = u.id
		WHERE cp.contest_id=$1 AND cp.left_at IS NULL AND cp.participant_type='CONTESTANT'
		ORDER BY u.full_name, u.login`, contestID, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []BriefingContestant{}
	for rows.Next() {
		var c BriefingContestant
		var oid *string
		var customText, customPublish, hidden, replaceFiles *bool
		var body *string
		var publishAt *time.Time
		var updated *time.Time
		if err := rows.Scan(
			&c.UserID, &c.Login, &c.FullName, &c.Organization,
			&oid, &customText, &body, &customPublish, &publishAt,
			&hidden, &replaceFiles, &updated,
		); err != nil {
			return nil, err
		}
		if oid != nil {
			o := BriefingOverride{
				ID: *oid, ChallengeID: challengeID, ContestantUserID: c.UserID,
				CustomText: derefBool(customText), BodyText: derefStr(body),
				CustomPublish: derefBool(customPublish), PublishAt: publishAt,
				Hidden: derefBool(hidden), ReplaceFiles: derefBool(replaceFiles),
			}
			if updated != nil {
				o.UpdatedAt = *updated
			}
			c.Override = &o
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (r *Repo) listBriefingFiles(ctx context.Context, challengeID string, overrideID *string) ([]BriefingFile, error) {
	q := `
		SELECT f.id, f.original_name, f.size_bytes, f.mime_type, f.object_key
		FROM challenge_briefing_files bf
		JOIN files f ON f.id = bf.file_id AND f.deleted_at IS NULL
		WHERE bf.challenge_id=$1`
	args := []any{challengeID}
	if overrideID == nil {
		q += ` AND bf.override_id IS NULL`
	} else {
		q += ` AND bf.override_id=$2`
		args = append(args, *overrideID)
	}
	q += ` ORDER BY bf.sort_order, bf.created_at`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []BriefingFile{}
	for rows.Next() {
		var f BriefingFile
		if err := rows.Scan(&f.FileID, &f.OriginalName, &f.SizeBytes, &f.MimeType, &f.ObjectKey); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (r *Repo) CountBriefingFiles(ctx context.Context, challengeID string, overrideID *string) (int, error) {
	var n int
	if overrideID == nil {
		err := r.pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM challenge_briefing_files
			WHERE challenge_id=$1 AND override_id IS NULL`, challengeID).Scan(&n)
		return n, err
	}
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM challenge_briefing_files
		WHERE challenge_id=$1 AND override_id=$2`, challengeID, *overrideID).Scan(&n)
	return n, err
}

func (r *Repo) InsertBriefingFile(ctx context.Context, challengeID string, overrideID *string, ownerID, contestID, objectKey, original, safe, ext, mime string, size int64) (string, error) {
	var fileID string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO files (
			owner_user_id, contest_id, challenge_id, bucket, object_key,
			original_name, safe_name, extension, mime_type, size_bytes, status, uploaded_at
		) VALUES ($1,$2,$3,'',$4,$5,$6,$7,$8,$9,'READY', now())
		RETURNING id`,
		ownerID, contestID, challengeID, objectKey, original, safe, nullIfEmpty(ext), nullIfEmpty(mime), size,
	).Scan(&fileID)
	if err != nil {
		return "", err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO challenge_briefing_files (challenge_id, override_id, file_id, sort_order)
		VALUES ($1,$2,$3, COALESCE((
			SELECT MAX(sort_order)+1 FROM challenge_briefing_files
			WHERE challenge_id=$1 AND override_id IS NOT DISTINCT FROM $2
		), 0))`, challengeID, overrideID, fileID)
	if err != nil {
		return "", err
	}
	return fileID, nil
}

func (r *Repo) BriefingFileMeta(ctx context.Context, challengeID, fileID string) (objectKey string, overrideID *string, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT f.object_key, bf.override_id
		FROM challenge_briefing_files bf
		JOIN files f ON f.id = bf.file_id
		WHERE bf.challenge_id=$1 AND bf.file_id=$2 AND f.deleted_at IS NULL`,
		challengeID, fileID).Scan(&objectKey, &overrideID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	return objectKey, overrideID, err
}

func (r *Repo) DeleteBriefingFile(ctx context.Context, challengeID, fileID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM challenge_briefing_files WHERE challenge_id=$1 AND file_id=$2`, challengeID, fileID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE files SET status='DELETED', deleted_at=now(), updated_at=now() WHERE id=$1`, fileID)
	return err
}

func (r *Repo) ContestantInContest(ctx context.Context, contestID, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM contest_participants
			WHERE contest_id=$1 AND user_id=$2 AND left_at IS NULL AND participant_type='CONTESTANT'
		)`, contestID, userID).Scan(&ok)
	return ok, err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func derefBool(v *bool) bool {
	return v != nil && *v
}

func derefStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
