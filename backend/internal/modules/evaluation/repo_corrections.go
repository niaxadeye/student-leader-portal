package evaluation

import (
	"context"
	"errors"
)

func (r *Repo) InsertScoreCorrection(ctx context.Context, c ScoreCorrection, contestID, challengeID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO evaluation_score_corrections (
			contest_id, challenge_id, actor_user_id, contestant_user_id, jury_user_id,
			criterion_id, criterion_title, kind, old_score, new_score, reason
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		contestID, challengeID, c.ActorUserID, c.ContestantUserID, c.JuryUserID,
		c.CriterionID, c.CriterionTitle, c.Kind, c.OldScore, c.NewScore, c.Reason,
	)
	return err
}

func (r *Repo) ListScoreCorrections(ctx context.Context, challengeID string) ([]ScoreCorrection, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.kind, c.actor_user_id,
		       COALESCE(NULLIF(BTRIM(actor.full_name), ''), actor.login),
		       c.contestant_user_id,
		       COALESCE(NULLIF(BTRIM(contestant.full_name), ''), contestant.login),
		       c.jury_user_id,
		       CASE WHEN jury.id IS NULL THEN NULL
		            ELSE COALESCE(NULLIF(BTRIM(jury.full_name), ''), jury.login) END,
		       c.criterion_id, c.criterion_title, c.old_score, c.new_score, c.reason, c.created_at
		FROM evaluation_score_corrections c
		JOIN users actor ON actor.id = c.actor_user_id
		JOIN users contestant ON contestant.id = c.contestant_user_id
		LEFT JOIN users jury ON jury.id = c.jury_user_id
		WHERE c.challenge_id = $1
		ORDER BY c.created_at DESC
		LIMIT 200`, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []ScoreCorrection{}
	for rows.Next() {
		var row ScoreCorrection
		if err := rows.Scan(
			&row.ID, &row.Kind, &row.ActorUserID, &row.ActorName,
			&row.ContestantUserID, &row.ContestantName,
			&row.JuryUserID, &row.JuryName,
			&row.CriterionID, &row.CriterionTitle, &row.OldScore, &row.NewScore,
			&row.Reason, &row.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *Repo) sheetHasScore(ctx context.Context, sheetID, criterionID string) (*float64, error) {
	v, err := r.ScoreValue(ctx, sheetID, criterionID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	score := v.Score
	return &score, nil
}
