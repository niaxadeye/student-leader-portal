package evaluation

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type runtimeReset struct {
	ReplaceJury bool
	JuryUserIDs []string
	ContestID   string
}

func (r *Repo) ChallengeJuryRestricted(ctx context.Context, challengeID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM evaluation_staff_assignments
			WHERE challenge_id = $1 AND role = $2 AND active
		)`, challengeID, RoleJury).Scan(&ok)
	return ok, err
}

func (r *Repo) UserIsChallengeJury(ctx context.Context, userID, challengeID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM evaluation_staff_assignments
			WHERE challenge_id = $1 AND user_id = $2 AND role = $3 AND active
		)`, challengeID, userID, RoleJury).Scan(&ok)
	return ok, err
}

func (r *Repo) ListContestRoleJury(ctx context.Context, contestID string) ([]JuryPerson, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.login, u.full_name
		FROM user_roles ur
		JOIN roles rl ON rl.id = ur.role_id AND rl.code = 'JURY'
		JOIN users u ON u.id = ur.user_id
		WHERE ur.scope_type = 'CONTEST' AND ur.scope_id = $1
		  AND NOT EXISTS (
		    SELECT 1 FROM user_roles rur
		    JOIN roles rrl ON rrl.id = rur.role_id AND rrl.code = 'REMOTE_JURY'
		    WHERE rur.user_id = u.id AND rur.scope_type = 'CONTEST' AND rur.scope_id = $1
		  )
		  AND NOT EXISTS (
		    SELECT 1
		    FROM evaluation_staff_assignments a
		    JOIN evaluation_schemes s ON s.challenge_id = a.challenge_id AND s.active AND s.type = 'REMOTE_CRITERIA'
		    WHERE a.user_id = u.id AND a.contest_id = $1 AND a.role = $2 AND a.active
		  )
		ORDER BY u.full_name, u.login`, contestID, RoleJury)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []JuryPerson{}
	for rows.Next() {
		var p JuryPerson
		if err := rows.Scan(&p.UserID, &p.Login, &p.FullName); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *Repo) ListChallengeJury(ctx context.Context, challengeID string) ([]JuryPerson, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.login, u.full_name
		FROM evaluation_staff_assignments a
		JOIN users u ON u.id = a.user_id
		WHERE a.challenge_id = $1 AND a.role = $2 AND a.active
		ORDER BY u.full_name, u.login`, challengeID, RoleJury)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []JuryPerson{}
	for rows.Next() {
		var p JuryPerson
		if err := rows.Scan(&p.UserID, &p.Login, &p.FullName); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *Repo) ResetChallengeRuntime(ctx context.Context, challengeID, actorID string, baseRev int, sess *Session, extra *runtimeReset) (*Session, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var current int
	err = tx.QueryRow(ctx, `SELECT revision FROM evaluation_sessions WHERE id=$1 FOR UPDATE`, sess.ID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if current != baseRev {
		return nil, ErrRevision
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM score_value_history
		WHERE score_value_id IN (
			SELECT sv.id
			FROM score_values sv
			JOIN score_sheets sh ON sh.id = sv.score_sheet_id
			JOIN performances p ON p.id = sh.performance_id
			WHERE p.challenge_id = $1
		)`, challengeID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM score_values
		WHERE score_sheet_id IN (
			SELECT sh.id
			FROM score_sheets sh
			JOIN performances p ON p.id = sh.performance_id
			WHERE p.challenge_id = $1
		)`, challengeID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM score_sheets
		WHERE performance_id IN (SELECT id FROM performances WHERE challenge_id = $1)`, challengeID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM evaluation_numeric_results WHERE challenge_id = $1`, challengeID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE life_events SET reverses_life_event_id = NULL WHERE challenge_id = $1`, challengeID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM life_events WHERE challenge_id = $1`, challengeID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM life_answer_marks WHERE challenge_id = $1`, challengeID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE performances
		SET status='READY', started_at=NULL, finished_at=NULL, speech_duration_seconds=NULL, updated_at=now()
		WHERE challenge_id=$1 AND status <> 'CANCELLED'`, challengeID); err != nil {
		return nil, err
	}

	if extra != nil && extra.ReplaceJury {
		if _, err := tx.Exec(ctx, `
			UPDATE evaluation_staff_assignments
			SET active = FALSE, updated_at = now()
			WHERE challenge_id = $1 AND role = $2 AND active`,
			challengeID, RoleJury); err != nil {
			return nil, err
		}
		for _, userID := range extra.JuryUserIDs {
			if _, err := tx.Exec(ctx, `
				INSERT INTO evaluation_staff_assignments (contest_id, challenge_id, user_id, role, active)
				VALUES ($1, $2, $3, $4, TRUE)`,
				extra.ContestID, challengeID, userID, RoleJury); err != nil {
				return nil, err
			}
		}
	}

	row := tx.QueryRow(ctx, `
		UPDATE evaluation_sessions SET
			state=$2,
			current_performance_id=$3,
			current_contestant_user_id=$4,
			current_match_id=$5,
			current_phase_id=$6,
			started_at=$7,
			finished_at=$8,
			controlled_by=$9,
			phase_started_at=$10,
			phase_duration_seconds=$11,
			paused_at=$12,
			accumulated_pause_seconds=$13,
			current_question_number=$14,
			question_count=$15,
			state_changed_at=now(),
			revision=revision+1,
			updated_at=now()
		WHERE id=$1
		RETURNING `+sessionCols,
		sess.ID, sess.State, sess.CurrentPerformanceID, sess.CurrentContestantUserID, sess.CurrentMatchID,
		sess.CurrentPhaseID, sess.StartedAt, sess.FinishedAt, actorID, sess.PhaseStartedAt,
		sess.PhaseDurationSeconds, sess.PausedAt, sess.AccumulatedPauseSeconds,
		questionNumberOf(sess), questionCountOf(sess),
	)
	out, err := scanSession(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repo) SetChallengeJury(ctx context.Context, contestID, challengeID string, userIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE evaluation_staff_assignments
		SET active = FALSE, updated_at = now()
		WHERE challenge_id = $1 AND role = $2 AND active`,
		challengeID, RoleJury); err != nil {
		return err
	}
	for _, userID := range userIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO evaluation_staff_assignments (contest_id, challenge_id, user_id, role, active)
			VALUES ($1, $2, $3, $4, TRUE)`,
			contestID, challengeID, userID, RoleJury); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repo) UserHasRemoteJury(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_roles ur
			JOIN roles rl ON rl.id = ur.role_id AND rl.code = $2
			WHERE ur.user_id = $1
		) OR EXISTS (
			SELECT 1
			FROM evaluation_staff_assignments a
			JOIN evaluation_schemes s ON s.challenge_id = a.challenge_id AND s.active AND s.type = 'REMOTE_CRITERIA'
			WHERE a.user_id = $1 AND a.role = $3 AND a.active
		)`, userID, RoleRemoteJury, RoleJury).Scan(&ok)
	return ok, err
}

func (r *Repo) UserIsRemoteJuryInContest(ctx context.Context, userID, contestID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_roles ur
			JOIN roles rl ON rl.id = ur.role_id AND rl.code = $3
			WHERE ur.user_id = $1 AND ur.scope_type = 'CONTEST' AND ur.scope_id = $2
		) OR EXISTS (
			SELECT 1
			FROM evaluation_staff_assignments a
			JOIN evaluation_schemes s ON s.challenge_id = a.challenge_id AND s.active AND s.type = 'REMOTE_CRITERIA'
			WHERE a.user_id = $1 AND a.contest_id = $2 AND a.role = $4 AND a.active
		)`, userID, contestID, RoleRemoteJury, RoleJury).Scan(&ok)
	return ok, err
}

func (r *Repo) UserHasRemoteJuryRole(ctx context.Context, userID, contestID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_roles ur
			JOIN roles rl ON rl.id = ur.role_id AND rl.code = $3
			WHERE ur.user_id = $1 AND ur.scope_type = 'CONTEST' AND ur.scope_id = $2
		)`, userID, contestID, RoleRemoteJury).Scan(&ok)
	return ok, err
}

func (r *Repo) ListContestRemoteJury(ctx context.Context, contestID string) ([]JuryPerson, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.login, u.full_name
		FROM user_roles ur
		JOIN roles rl ON rl.id = ur.role_id AND rl.code = $2
		JOIN users u ON u.id = ur.user_id
		WHERE ur.scope_type = 'CONTEST' AND ur.scope_id = $1
		  AND u.deleted_at IS NULL AND u.status = 'ACTIVE'
		ORDER BY u.full_name, u.login`, contestID, RoleRemoteJury)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []JuryPerson{}
	for rows.Next() {
		var p JuryPerson
		if err := rows.Scan(&p.UserID, &p.Login, &p.FullName); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}
