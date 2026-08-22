package evaluation

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

type challengeRef struct {
	ID        string
	ContestID string
	Title     string
	Status    string
}

func (r *Repo) ChallengeByID(ctx context.Context, challengeID string) (*challengeRef, error) {
	var c challengeRef
	err := r.pool.QueryRow(ctx, `
		SELECT id, contest_id, title, status FROM contest_challenges WHERE id = $1`, challengeID).Scan(&c.ID, &c.ContestID, &c.Title, &c.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrChallenge
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repo) SchemeByChallenge(ctx context.Context, challengeID string) (*Scheme, error) {
	s := Scheme{ChallengeID: challengeID}
	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.challenge_id, c.contest_id, s.name, s.type, s.scoring_unit,
		       s.min_score, s.max_score, s.corridor_mode, s.result_visibility, s.edit_policy,
		       s.settings_json, s.active, s.created_at, s.updated_at
		FROM evaluation_schemes s
		JOIN contest_challenges c ON c.id = s.challenge_id
		WHERE s.challenge_id = $1`, challengeID).Scan(
		&s.ID, &s.ChallengeID, &s.ContestID, &s.Name, &s.Type, &s.ScoringUnit,
		&s.MinScore, &s.MaxScore, &s.CorridorMode, &s.ResultVisibility, &s.EditPolicy,
		&s.SettingsJSON, &s.Active, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	criteria, err := r.listCriteria(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Criteria = criteria
	return &s, nil
}

func (r *Repo) UpsertScheme(ctx context.Context, challengeID, actorID string, in SchemeInput) (*Scheme, error) {
	settings := in.SettingsJSON
	if len(settings) == 0 {
		settings = []byte("{}")
	}
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO evaluation_schemes (
			challenge_id, name, type, scoring_unit, min_score, max_score,
			corridor_mode, result_visibility, edit_policy, settings_json, created_by, updated_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)
		ON CONFLICT (challenge_id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			scoring_unit = EXCLUDED.scoring_unit,
			min_score = EXCLUDED.min_score,
			max_score = EXCLUDED.max_score,
			corridor_mode = EXCLUDED.corridor_mode,
			result_visibility = EXCLUDED.result_visibility,
			edit_policy = EXCLUDED.edit_policy,
			settings_json = EXCLUDED.settings_json,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()
		RETURNING id`,
		challengeID, in.Name, in.Type, in.ScoringUnit, in.MinScore, in.MaxScore,
		in.CorridorMode, in.ResultVisibility, in.EditPolicy, settings, nullIfEmpty(actorID),
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	_ = id
	return r.SchemeByChallenge(ctx, challengeID)
}

func (r *Repo) listCriteria(ctx context.Context, schemeID string) ([]Criterion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, evaluation_scheme_id, group_id, title, description, min_score, max_score,
		       weight, is_required, sort_order, active, created_at, updated_at
		FROM evaluation_criteria
		WHERE evaluation_scheme_id = $1 AND active
		ORDER BY sort_order, created_at`, schemeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Criterion
	for rows.Next() {
		var c Criterion
		if err := rows.Scan(
			&c.ID, &c.SchemeID, &c.GroupID, &c.Title, &c.Description, &c.MinScore, &c.MaxScore,
			&c.Weight, &c.IsRequired, &c.SortOrder, &c.Active, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []Criterion{}, nil
	}
	ids := make([]string, len(list))
	idx := make(map[string]int, len(list))
	for i := range list {
		ids[i] = list[i].ID
		idx[list[i].ID] = i
		list[i].Bands = []ScaleBand{}
	}
	bandRows, err := r.pool.Query(ctx, `
		SELECT id, criterion_id, min_score, max_score, description, sort_order
		FROM criterion_scale_bands
		WHERE criterion_id = ANY($1::uuid[])
		ORDER BY sort_order, min_score`, ids)
	if err != nil {
		return nil, err
	}
	defer bandRows.Close()
	for bandRows.Next() {
		var b ScaleBand
		if err := bandRows.Scan(&b.ID, &b.CriterionID, &b.MinScore, &b.MaxScore, &b.Description, &b.SortOrder); err != nil {
			return nil, err
		}
		if i, ok := idx[b.CriterionID]; ok {
			list[i].Bands = append(list[i].Bands, b)
		}
	}
	return list, bandRows.Err()
}

func (r *Repo) InsertCriterion(ctx context.Context, schemeID string, in CriterionInput) (*Criterion, error) {
	var sort int
	if err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order), -1) + 1 FROM evaluation_criteria WHERE evaluation_scheme_id = $1`, schemeID).Scan(&sort); err != nil {
		return nil, err
	}
	required := true
	if in.IsRequired != nil {
		required = *in.IsRequired
	}
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO evaluation_criteria (
			evaluation_scheme_id, title, description, min_score, max_score, weight, is_required, sort_order
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id`,
		schemeID, in.Title, in.Description, in.MinScore, in.MaxScore, in.Weight, required, sort,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err := r.replaceBands(ctx, id, in.Bands); err != nil {
		return nil, err
	}
	return r.criterionByID(ctx, id)
}

func (r *Repo) UpdateCriterion(ctx context.Context, criterionID string, in CriterionInput) (*Criterion, error) {
	required := true
	if in.IsRequired != nil {
		required = *in.IsRequired
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE evaluation_criteria
		SET title=$2, description=$3, min_score=$4, max_score=$5, weight=$6, is_required=$7, updated_at=now()
		WHERE id=$1 AND active`,
		criterionID, in.Title, in.Description, in.MinScore, in.MaxScore, in.Weight, required,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	if err := r.replaceBands(ctx, criterionID, in.Bands); err != nil {
		return nil, err
	}
	return r.criterionByID(ctx, criterionID)
}

func (r *Repo) SoftDeleteCriterion(ctx context.Context, criterionID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE evaluation_criteria SET active=FALSE, updated_at=now() WHERE id=$1 AND active`, criterionID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ReorderCriteria(ctx context.Context, schemeID string, ids []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for i, id := range ids {
		tag, err := tx.Exec(ctx, `
			UPDATE evaluation_criteria SET sort_order=$3, updated_at=now()
			WHERE id=$1 AND evaluation_scheme_id=$2 AND active`, id, schemeID, i)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrValidation
		}
	}
	return tx.Commit(ctx)
}

func (r *Repo) CriterionSchemeID(ctx context.Context, criterionID string) (string, error) {
	var schemeID string
	err := r.pool.QueryRow(ctx, `
		SELECT evaluation_scheme_id FROM evaluation_criteria WHERE id=$1 AND active`, criterionID).Scan(&schemeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return schemeID, err
}

func (r *Repo) criterionByID(ctx context.Context, id string) (*Criterion, error) {
	var c Criterion
	err := r.pool.QueryRow(ctx, `
		SELECT id, evaluation_scheme_id, group_id, title, description, min_score, max_score,
		       weight, is_required, sort_order, active, created_at, updated_at
		FROM evaluation_criteria WHERE id=$1`, id).Scan(
		&c.ID, &c.SchemeID, &c.GroupID, &c.Title, &c.Description, &c.MinScore, &c.MaxScore,
		&c.Weight, &c.IsRequired, &c.SortOrder, &c.Active, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, criterion_id, min_score, max_score, description, sort_order
		FROM criterion_scale_bands WHERE criterion_id=$1 ORDER BY sort_order, min_score`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	c.Bands = []ScaleBand{}
	for rows.Next() {
		var b ScaleBand
		if err := rows.Scan(&b.ID, &b.CriterionID, &b.MinScore, &b.MaxScore, &b.Description, &b.SortOrder); err != nil {
			return nil, err
		}
		c.Bands = append(c.Bands, b)
	}
	return &c, rows.Err()
}

func (r *Repo) replaceBands(ctx context.Context, criterionID string, bands []ScaleBandInput) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM criterion_scale_bands WHERE criterion_id=$1`, criterionID); err != nil {
		return err
	}
	for i, b := range bands {
		if _, err := r.pool.Exec(ctx, `
			INSERT INTO criterion_scale_bands (criterion_id, min_score, max_score, description, sort_order)
			VALUES ($1,$2,$3,$4,$5)`, criterionID, b.MinScore, b.MaxScore, b.Description, i); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) SnapshotScheme(ctx context.Context, schemeID, actorID string, snapshot []byte) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO evaluation_scheme_versions (evaluation_scheme_id, version, configuration_snapshot, created_by)
		SELECT $1, COALESCE((SELECT MAX(version) FROM evaluation_scheme_versions WHERE evaluation_scheme_id = $1), 0) + 1, $2, $3`,
		schemeID, snapshot, nullIfEmpty(actorID))
	return err
}

func (r *Repo) ListJuryContests(ctx context.Context, userID string) ([]JuryContest, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.name, c.slug
		FROM contests c
		WHERE
		  EXISTS (
		    SELECT 1 FROM user_roles ur
		    JOIN roles rl ON rl.id = ur.role_id AND rl.code IN ('JURY', 'REMOTE_JURY')
		    WHERE ur.user_id = $1 AND ur.scope_type = 'CONTEST' AND ur.scope_id = c.id
		  )
		  OR EXISTS (
		    SELECT 1
		    FROM evaluation_staff_assignments a
		    JOIN evaluation_schemes s ON s.challenge_id = a.challenge_id AND s.active AND s.type = 'REMOTE_CRITERIA'
		    WHERE a.user_id = $1 AND a.contest_id = c.id AND a.role = 'JURY' AND a.active
		  )
		ORDER BY c.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []JuryContest
	idx := map[string]int{}
	var ids []string
	for rows.Next() {
		var c JuryContest
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug); err != nil {
			return nil, err
		}
		c.Challenges = []JuryChallenge{}
		idx[c.ID] = len(list)
		ids = append(ids, c.ID)
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []JuryContest{}, nil
	}
	chRows, err := r.pool.Query(ctx, `
		SELECT ch.contest_id, ch.id, ch.title, ch.slug, ch.status,
		       EXISTS (SELECT 1 FROM evaluation_schemes s WHERE s.challenge_id = ch.id AND s.active) AS has_scheme,
		       (SELECT s.type FROM evaluation_schemes s WHERE s.challenge_id = ch.id AND s.active LIMIT 1) AS scheme_type
		FROM contest_challenges ch
		WHERE ch.contest_id = ANY($1::uuid[]) AND ch.status <> 'ARCHIVED'
		  AND (
		    (
		      EXISTS (
		        SELECT 1 FROM evaluation_schemes s
		        WHERE s.challenge_id = ch.id AND s.active AND s.type = 'REMOTE_CRITERIA'
		      )
		      AND EXISTS (
		        SELECT 1 FROM evaluation_staff_assignments a
		        WHERE a.challenge_id = ch.id AND a.role = 'JURY' AND a.active AND a.user_id = $2
		      )
		    )
		    OR
		    (
		      NOT EXISTS (
		        SELECT 1 FROM evaluation_schemes s
		        WHERE s.challenge_id = ch.id AND s.active AND s.type = 'REMOTE_CRITERIA'
		      )
		      AND EXISTS (
		        SELECT 1 FROM user_roles ur
		        JOIN roles rl ON rl.id = ur.role_id AND rl.code = 'JURY'
		        WHERE ur.user_id = $2 AND ur.scope_type = 'CONTEST' AND ur.scope_id = ch.contest_id
		      )
		      AND NOT EXISTS (
		        SELECT 1 FROM user_roles ur
		        JOIN roles rl ON rl.id = ur.role_id AND rl.code = 'REMOTE_JURY'
		        WHERE ur.user_id = $2 AND ur.scope_type = 'CONTEST' AND ur.scope_id = ch.contest_id
		      )
		      AND NOT EXISTS (
		        SELECT 1
		        FROM evaluation_staff_assignments a
		        JOIN evaluation_schemes s ON s.challenge_id = a.challenge_id AND s.active AND s.type = 'REMOTE_CRITERIA'
		        WHERE a.user_id = $2 AND a.contest_id = ch.contest_id AND a.role = 'JURY' AND a.active
		      )
		      AND (
		        NOT EXISTS (
		          SELECT 1 FROM evaluation_staff_assignments a
		          WHERE a.challenge_id = ch.id AND a.role = 'JURY' AND a.active
		        )
		        OR EXISTS (
		          SELECT 1 FROM evaluation_staff_assignments a
		          WHERE a.challenge_id = ch.id AND a.role = 'JURY' AND a.active AND a.user_id = $2
		        )
		      )
		    )
		  )
		ORDER BY ch.sort_order, ch.created_at`, ids, userID)
	if err != nil {
		return nil, err
	}
	defer chRows.Close()
	for chRows.Next() {
		var contestID string
		var ch JuryChallenge
		var schemeType *string
		if err := chRows.Scan(&contestID, &ch.ID, &ch.Title, &ch.Slug, &ch.Status, &ch.HasScheme, &schemeType); err != nil {
			return nil, err
		}
		if schemeType != nil {
			ch.SchemeType = *schemeType
		}
		if i, ok := idx[contestID]; ok {
			list[i].Challenges = append(list[i].Challenges, ch)
		}
	}
	return list, chRows.Err()
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
