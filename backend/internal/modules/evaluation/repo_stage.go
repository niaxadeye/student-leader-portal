package evaluation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repo) StageLinkByMain(ctx context.Context, mainChallengeID string) (*StageLink, error) {
	return r.scanStageLink(r.pool.QueryRow(ctx, `
		SELECT l.id, l.contest_id, l.main_challenge_id, m.title, l.remote_challenge_id, rem.title,
		       l.main_weight, l.remote_weight, l.combine_mode
		FROM evaluation_stage_links l
		JOIN contest_challenges m ON m.id = l.main_challenge_id
		JOIN contest_challenges rem ON rem.id = l.remote_challenge_id
		WHERE l.main_challenge_id = $1`, mainChallengeID))
}

func (r *Repo) StageLinkByRemote(ctx context.Context, remoteChallengeID string) (*StageLink, error) {
	return r.scanStageLink(r.pool.QueryRow(ctx, `
		SELECT l.id, l.contest_id, l.main_challenge_id, m.title, l.remote_challenge_id, rem.title,
		       l.main_weight, l.remote_weight, l.combine_mode
		FROM evaluation_stage_links l
		JOIN contest_challenges m ON m.id = l.main_challenge_id
		JOIN contest_challenges rem ON rem.id = l.remote_challenge_id
		WHERE l.remote_challenge_id = $1`, remoteChallengeID))
}

func (r *Repo) scanStageLink(row pgx.Row) (*StageLink, error) {
	var l StageLink
	err := row.Scan(
		&l.ID, &l.ContestID, &l.MainChallengeID, &l.MainTitle, &l.RemoteChallengeID, &l.RemoteTitle,
		&l.MainWeight, &l.RemoteWeight, &l.CombineMode,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *Repo) ListRemoteStageOptions(ctx context.Context, contestID, mainChallengeID string) ([]ChallengeOption, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ch.id, ch.title
		FROM contest_challenges ch
		JOIN evaluation_schemes s ON s.challenge_id = ch.id AND s.active AND s.type = $3
		WHERE ch.contest_id = $1 AND ch.id <> $2
		  AND (
		    ch.status <> 'ARCHIVED'
		    OR EXISTS (
		      SELECT 1 FROM evaluation_stage_links l
		      WHERE l.remote_challenge_id = ch.id AND l.main_challenge_id = $2
		    )
		  )
		  AND (
		    NOT EXISTS (SELECT 1 FROM evaluation_stage_links l WHERE l.remote_challenge_id = ch.id)
		    OR EXISTS (
		      SELECT 1 FROM evaluation_stage_links l
		      WHERE l.remote_challenge_id = ch.id AND l.main_challenge_id = $2
		    )
		  )
		ORDER BY ch.sort_order, ch.created_at`, contestID, mainChallengeID, TypeRemoteCriteria)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []ChallengeOption{}
	for rows.Next() {
		var o ChallengeOption
		if err := rows.Scan(&o.ID, &o.Title); err != nil {
			return nil, err
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

func (r *Repo) UpsertStageLink(ctx context.Context, contestID, mainID, remoteID string, mainW, remoteW float64, mode string) (*StageLink, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO evaluation_stage_links (
			contest_id, main_challenge_id, remote_challenge_id, main_weight, remote_weight, combine_mode
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (main_challenge_id) DO UPDATE SET
			remote_challenge_id = EXCLUDED.remote_challenge_id,
			main_weight = EXCLUDED.main_weight,
			remote_weight = EXCLUDED.remote_weight,
			combine_mode = EXCLUDED.combine_mode,
			updated_at = now()`,
		contestID, mainID, remoteID, mainW, remoteW, mode)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrValidation
		}
		return nil, err
	}
	return r.StageLinkByMain(ctx, mainID)
}

func (r *Repo) DeleteStageLinkByMain(ctx context.Context, mainChallengeID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM evaluation_stage_links WHERE main_challenge_id = $1`, mainChallengeID)
	return err
}

func (r *Repo) DeleteStageLinkByRemote(ctx context.Context, remoteChallengeID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM evaluation_stage_links WHERE remote_challenge_id = $1`, remoteChallengeID)
	return err
}
