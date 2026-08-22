package evaluation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *Repo) NumericScore(ctx context.Context, challengeID, contestantUserID string) (*float64, error) {
	var score float64
	err := r.pool.QueryRow(ctx, `
		SELECT score FROM evaluation_numeric_results
		WHERE challenge_id = $1 AND contestant_user_id = $2`, challengeID, contestantUserID).Scan(&score)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &score, nil
}

type NumericResult struct {
	ContestantUserID string
	Score            float64
}

func (r *Repo) ListNumericResults(ctx context.Context, challengeID string) ([]NumericResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT contestant_user_id, score
		FROM evaluation_numeric_results
		WHERE challenge_id = $1`, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []NumericResult{}
	for rows.Next() {
		var row NumericResult
		if err := rows.Scan(&row.ContestantUserID, &row.Score); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

func (r *Repo) UpsertNumericResult(ctx context.Context, challengeID, contestantUserID string, score float64, actorID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO evaluation_numeric_results (challenge_id, contestant_user_id, score, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (challenge_id, contestant_user_id) DO UPDATE SET
			score = EXCLUDED.score,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()`,
		challengeID, contestantUserID, score, actorID)
	return err
}

func (r *Repo) DeleteNumericResult(ctx context.Context, challengeID, contestantUserID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM evaluation_numeric_results
		WHERE challenge_id = $1 AND contestant_user_id = $2`, challengeID, contestantUserID)
	return err
}
