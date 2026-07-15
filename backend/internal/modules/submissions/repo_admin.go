package submissions

import "context"

// AdminListFilter — фильтры таблицы дирекции (SITE.md §7.6).
type AdminListFilter struct {
	ChallengeID string
	Status      string // DRAFT|SUBMITTED|LOCKED|"" (все)
	Limit       int
	Offset      int
}

// AdminRow — строка таблицы работ по испытанию.
type AdminRow struct {
	Submission
	FileCount int
}

// AdminList возвращает работы по испытанию с ФИО/логином/организацией и числом файлов.
func (r *Repo) AdminList(ctx context.Context, f AdminListFilter) ([]AdminRow, int, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	// total
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM submissions s
		WHERE s.challenge_id=$1 AND ($2='' OR s.status=$2)`,
		f.ChallengeID, f.Status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+subCols+`,
		       u.full_name, u.login, u.organization,
		       (SELECT count(*) FROM submission_files sf WHERE sf.submission_id = s.id) AS file_count
		FROM submissions s
		JOIN users u ON u.id = s.contestant_user_id
		WHERE s.challenge_id=$1 AND ($2='' OR s.status=$2)
		ORDER BY s.submitted_at DESC NULLS LAST, s.updated_at DESC
		LIMIT $3 OFFSET $4`,
		f.ChallengeID, f.Status, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []AdminRow
	for rows.Next() {
		var a AdminRow
		s := &a.Submission
		if err := rows.Scan(&s.ID, &s.ChallengeID, &s.ContestantUserID, &s.Status, &s.Answers,
			&s.SchemaVersion, &s.Version, &s.CurrentRevisionNumber, &s.FirstOpenedAt,
			&s.LastSavedAt, &s.SubmittedAt, &s.LastResubmittedAt, &s.LockedAt, &s.LockReason,
			&s.CreatedAt, &s.UpdatedAt,
			&s.FullName, &s.Login, &s.Organization, &a.FileCount); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	return out, total, rows.Err()
}

// LoadContestant присоединяет ФИО/логин/организацию к одной работе (для карточки).
func (r *Repo) LoadContestant(ctx context.Context, s *Submission) error {
	return r.pool.QueryRow(ctx,
		`SELECT full_name, login, organization FROM users WHERE id=$1`,
		s.ContestantUserID).Scan(&s.FullName, &s.Login, &s.Organization)
}
