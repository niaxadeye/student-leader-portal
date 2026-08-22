package evaluation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *Repo) EnsureScoreSheet(ctx context.Context, performanceID, evaluatorUserID string) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO score_sheets (performance_id, evaluator_user_id)
		VALUES ($1, $2)
		ON CONFLICT (performance_id, evaluator_user_id) DO UPDATE SET
			updated_at = score_sheets.updated_at
		RETURNING id`, performanceID, evaluatorUserID).Scan(&id)
	return id, err
}

func (r *Repo) ScoreValueByMutation(ctx context.Context, mutationID string) (*ScoreValue, error) {
	if mutationID == "" {
		return nil, ErrNotFound
	}
	var v ScoreValue
	err := r.pool.QueryRow(ctx, `
		SELECT id, score_sheet_id, criterion_id, score, comment, revision, last_mutation_id
		FROM score_values WHERE last_mutation_id = $1`, mutationID).Scan(
		&v.ID, &v.ScoreSheetID, &v.CriterionID, &v.Score, &v.Comment, &v.Revision, &v.LastMutationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repo) ListScoreValues(ctx context.Context, performanceID, evaluatorUserID string) ([]ScoreValue, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT sv.id, sv.score_sheet_id, sv.criterion_id, sv.score, sv.comment, sv.revision, sv.last_mutation_id
		FROM score_values sv
		JOIN score_sheets sh ON sh.id = sv.score_sheet_id
		WHERE sh.performance_id = $1 AND sh.evaluator_user_id = $2`, performanceID, evaluatorUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []ScoreValue{}
	for rows.Next() {
		var v ScoreValue
		if err := rows.Scan(&v.ID, &v.ScoreSheetID, &v.CriterionID, &v.Score, &v.Comment, &v.Revision, &v.LastMutationID); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, rows.Err()
}

func (r *Repo) ScoreValue(ctx context.Context, sheetID, criterionID string) (*ScoreValue, error) {
	var v ScoreValue
	err := r.pool.QueryRow(ctx, `
		SELECT id, score_sheet_id, criterion_id, score, comment, revision, last_mutation_id
		FROM score_values WHERE score_sheet_id = $1 AND criterion_id = $2`, sheetID, criterionID).Scan(
		&v.ID, &v.ScoreSheetID, &v.CriterionID, &v.Score, &v.Comment, &v.Revision, &v.LastMutationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repo) UpsertScoreValue(ctx context.Context, sheetID, criterionID string, score float64, mutationID *string, actorUserID string) (*ScoreValue, error) {
	var v ScoreValue
	err := r.pool.QueryRow(ctx, `
		INSERT INTO score_values (score_sheet_id, criterion_id, score, revision, last_mutation_id)
		VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (score_sheet_id, criterion_id) DO UPDATE SET
			score = EXCLUDED.score,
			revision = score_values.revision + 1,
			last_mutation_id = EXCLUDED.last_mutation_id,
			updated_at = now()
		RETURNING id, score_sheet_id, criterion_id, score, comment, revision, last_mutation_id`,
		sheetID, criterionID, score, mutationID,
	).Scan(&v.ID, &v.ScoreSheetID, &v.CriterionID, &v.Score, &v.Comment, &v.Revision, &v.LastMutationID)
	if err != nil {
		return nil, err
	}
	if actorUserID == "" {
		_ = r.pool.QueryRow(ctx, `SELECT evaluator_user_id FROM score_sheets WHERE id = $1`, sheetID).Scan(&actorUserID)
	}
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO score_value_history (score_value_id, score, revision, mutation_id, actor_user_id)
		VALUES ($1, $2, $3, $4, $5)`,
		v.ID, v.Score, v.Revision, mutationID, actorUserID,
	); err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repo) DeleteScoreValue(ctx context.Context, sheetID, criterionID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM score_values WHERE score_sheet_id = $1 AND criterion_id = $2`, sheetID, criterionID)
	return err
}

func (r *Repo) RefreshSheetTotal(ctx context.Context, sheetID string) (float64, error) {
	var total float64
	err := r.pool.QueryRow(ctx, `
		UPDATE score_sheets SET
			total_score_cache = (
				SELECT COALESCE(SUM(sv.score * c.weight), 0)
				FROM score_values sv
				JOIN evaluation_criteria c ON c.id = sv.criterion_id
				WHERE sv.score_sheet_id = $1
			),
			updated_at = now()
		WHERE id = $1
		RETURNING COALESCE(total_score_cache, 0)`, sheetID).Scan(&total)
	return total, err
}

func (r *Repo) ListContestJury(ctx context.Context, contestID, challengeID string) ([]JuryPerson, error) {
	assigned, err := r.ListChallengeJury(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	scheme, err := r.SchemeByChallenge(ctx, challengeID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if scheme != nil && ExclusiveChallengeJury(scheme.Type) {
		return assigned, nil
	}
	if len(assigned) > 0 {
		return assigned, nil
	}
	return r.ListContestRoleJury(ctx, contestID)
}

func (r *Repo) ListChallengeScoreRows(ctx context.Context, challengeID string) ([]challengeScoreRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.contestant_user_id, sh.evaluator_user_id, sv.criterion_id, sv.score
		FROM performances p
		JOIN score_sheets sh ON sh.performance_id = p.id
		JOIN score_values sv ON sv.score_sheet_id = sh.id
		WHERE p.challenge_id = $1`, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []challengeScoreRow{}
	for rows.Next() {
		var row challengeScoreRow
		if err := rows.Scan(&row.ContestantUserID, &row.JuryUserID, &row.CriterionID, &row.Score); err != nil {
			return nil, err
		}
		list = append(list, row)
	}
	return list, rows.Err()
}
