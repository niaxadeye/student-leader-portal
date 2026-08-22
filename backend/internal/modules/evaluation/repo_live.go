package evaluation

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

func scanSession(row pgx.Row) (*Session, error) {
	var s Session
	err := row.Scan(
		&s.ID, &s.ChallengeID, &s.CurrentPerformanceID, &s.CurrentContestantUserID, &s.CurrentMatchID,
		&s.State, &s.CurrentPhaseID, &s.StartedAt, &s.StateChangedAt, &s.FinishedAt, &s.ControlledBy,
		&s.Revision, &s.PhaseStartedAt, &s.PhaseDurationSeconds, &s.PausedAt, &s.AccumulatedPauseSeconds,
		&s.CurrentQuestionNumber, &s.QuestionCount, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

const sessionCols = `
		id, challenge_id, current_performance_id, current_contestant_user_id, current_match_id,
		state, current_phase_id, started_at, state_changed_at, finished_at, controlled_by,
		revision, phase_started_at, phase_duration_seconds, paused_at, accumulated_pause_seconds,
		current_question_number, question_count, created_at, updated_at`

func questionNumberOf(s *Session) int {
	if s == nil || s.CurrentQuestionNumber < 1 {
		return 1
	}
	return s.CurrentQuestionNumber
}

func questionCountOf(s *Session) int {
	if s == nil || s.QuestionCount < 0 {
		return 0
	}
	return s.QuestionCount
}

func (r *Repo) EnsureSession(ctx context.Context, challengeID string) (*Session, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO evaluation_sessions (challenge_id, state, revision)
		VALUES ($1, 'NOT_STARTED', 0)
		ON CONFLICT (challenge_id) DO NOTHING`, challengeID)
	if err != nil {
		return nil, err
	}
	return r.SessionByChallenge(ctx, challengeID)
}

func (r *Repo) SessionByChallenge(ctx context.Context, challengeID string) (*Session, error) {
	return scanSession(r.pool.QueryRow(ctx, `SELECT `+sessionCols+` FROM evaluation_sessions WHERE challenge_id=$1`, challengeID))
}

func (r *Repo) SessionState(ctx context.Context, challengeID string) (string, error) {
	var state string
	err := r.pool.QueryRow(ctx, `SELECT state FROM evaluation_sessions WHERE challenge_id=$1`, challengeID).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return StateNotStarted, nil
	}
	return state, err
}

func (r *Repo) UserIsJury(ctx context.Context, userID, contestID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM user_roles ur
			JOIN roles rl ON rl.id = ur.role_id AND rl.code = 'JURY'
			WHERE ur.user_id = $1 AND ur.scope_type = 'CONTEST' AND ur.scope_id = $2
		)`, userID, contestID).Scan(&ok)
	return ok, err
}

func (r *Repo) ListLiveContestants(ctx context.Context, contestID, challengeID string) ([]LiveContestant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.login, u.full_name, u.organization, p.id, p.status, d.draw_number, p.speech_duration_seconds, u.avatar_key
		FROM contest_participants cp
		JOIN users u ON u.id = cp.user_id
		LEFT JOIN performances p ON p.challenge_id = $2 AND p.contestant_user_id = u.id
		LEFT JOIN evaluation_draw_entries d ON d.challenge_id = $2 AND d.contestant_user_id = u.id
		WHERE cp.contest_id = $1 AND cp.left_at IS NULL AND cp.participant_type = 'CONTESTANT'
		ORDER BY d.draw_number NULLS LAST, cp.joined_at, u.full_name`, contestID, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []LiveContestant
	for rows.Next() {
		var c LiveContestant
		if err := rows.Scan(&c.UserID, &c.Login, &c.FullName, &c.Organization, &c.PerformanceID, &c.PerformanceStatus, &c.DrawNumber, &c.SpeechDurationSeconds, &c.AvatarKey); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	if list == nil {
		list = []LiveContestant{}
	}
	return list, rows.Err()
}

func (r *Repo) SetSpeechDuration(ctx context.Context, challengeID, contestantUserID string, seconds float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE performances
		SET speech_duration_seconds=$3, updated_at=now()
		WHERE challenge_id=$1 AND contestant_user_id=$2`, challengeID, contestantUserID, seconds)
	return err
}

func (r *Repo) PerformanceByID(ctx context.Context, id string) (*Performance, error) {
	var p Performance
	err := r.pool.QueryRow(ctx, `
		SELECT id, challenge_id, contestant_user_id, sequence_number, status, started_at, finished_at, created_at, updated_at
		FROM performances WHERE id=$1`, id).Scan(
		&p.ID, &p.ChallengeID, &p.ContestantUserID, &p.SequenceNumber, &p.Status,
		&p.StartedAt, &p.FinishedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repo) ListPhaseTemplates(ctx context.Context, schemeID string) ([]PhaseTemplate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, evaluation_scheme_id, title, duration_seconds, scoring_allowed, maps_to_state, sort_order
		FROM performance_phase_templates WHERE evaluation_scheme_id=$1
		ORDER BY sort_order, title`, schemeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []PhaseTemplate
	for rows.Next() {
		var t PhaseTemplate
		if err := rows.Scan(&t.ID, &t.SchemeID, &t.Title, &t.DurationSeconds, &t.ScoringAllowed, &t.MapsToState, &t.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	if list == nil {
		list = []PhaseTemplate{}
	}
	return list, rows.Err()
}

func (r *Repo) SeedDefaultPhases(ctx context.Context, schemeID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO performance_phase_templates (evaluation_scheme_id, title, duration_seconds, scoring_allowed, maps_to_state, sort_order)
		SELECT $1, x.title, x.duration_seconds, x.scoring_allowed, x.maps_to_state, x.sort_order
		FROM (VALUES
			('Выступление', 480, TRUE, 'LIVE', 0),
			('Вопросы', 300, TRUE, 'QUESTIONS', 1)
		) AS x(title, duration_seconds, scoring_allowed, maps_to_state, sort_order)
		WHERE NOT EXISTS (SELECT 1 FROM performance_phase_templates t WHERE t.evaluation_scheme_id = $1)`, schemeID)
	return err
}

func (r *Repo) PhaseByID(ctx context.Context, id string) (*PhaseTemplate, error) {
	var t PhaseTemplate
	err := r.pool.QueryRow(ctx, `
		SELECT id, evaluation_scheme_id, title, duration_seconds, scoring_allowed, maps_to_state, sort_order
		FROM performance_phase_templates WHERE id=$1`, id).Scan(
		&t.ID, &t.SchemeID, &t.Title, &t.DurationSeconds, &t.ScoringAllowed, &t.MapsToState, &t.SortOrder,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &t, err
}

func (r *Repo) SaveSession(ctx context.Context, baseRev int, s *Session) (*Session, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var current int
	err = tx.QueryRow(ctx, `SELECT revision FROM evaluation_sessions WHERE id=$1 FOR UPDATE`, s.ID).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if current != baseRev {
		out, rerr := r.SessionByChallenge(ctx, s.ChallengeID)
		if rerr != nil {
			return nil, ErrRevision
		}
		return out, ErrRevision
	}
	row := tx.QueryRow(ctx, `
		UPDATE evaluation_sessions SET
			state=$2,
			current_performance_id=$3,
			current_contestant_user_id=$4,
			current_phase_id=$5,
			started_at=$6,
			finished_at=$7,
			controlled_by=$8,
			phase_started_at=$9,
			phase_duration_seconds=$10,
			paused_at=$11,
			accumulated_pause_seconds=$12,
			current_question_number=$13,
			question_count=$14,
			state_changed_at=now(),
			revision=revision+1,
			updated_at=now()
		WHERE id=$1
		RETURNING `+sessionCols,
		s.ID, s.State, s.CurrentPerformanceID, s.CurrentContestantUserID, s.CurrentPhaseID,
		s.StartedAt, s.FinishedAt, s.ControlledBy, s.PhaseStartedAt, s.PhaseDurationSeconds,
		s.PausedAt, s.AccumulatedPauseSeconds, questionNumberOf(s), questionCountOf(s),
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

func (r *Repo) UpsertPerformance(ctx context.Context, challengeID, contestantUserID, status string, start bool) (*Performance, error) {
	seq := r.nextSequence(ctx, challengeID, contestantUserID)
	var started any
	if start {
		started = time.Now()
	}
	var p Performance
	err := r.pool.QueryRow(ctx, `
		INSERT INTO performances (challenge_id, contestant_user_id, sequence_number, status, started_at)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (challenge_id, contestant_user_id) DO UPDATE SET
			status = EXCLUDED.status,
			started_at = COALESCE(performances.started_at, EXCLUDED.started_at),
			finished_at = CASE WHEN EXCLUDED.status = 'FINISHED' THEN now() ELSE NULL END,
			updated_at = now()
		RETURNING id, challenge_id, contestant_user_id, sequence_number, status, started_at, finished_at, created_at, updated_at`,
		challengeID, contestantUserID, seq, status, started,
	).Scan(&p.ID, &p.ChallengeID, &p.ContestantUserID, &p.SequenceNumber, &p.Status, &p.StartedAt, &p.FinishedAt, &p.CreatedAt, &p.UpdatedAt)
	return &p, err
}

func (r *Repo) nextSequence(ctx context.Context, challengeID, contestantUserID string) int {
	var seq int
	_ = r.pool.QueryRow(ctx, `
		SELECT COALESCE(
			(SELECT draw_number FROM evaluation_draw_entries WHERE challenge_id=$1 AND contestant_user_id=$2),
			(SELECT COALESCE(MAX(sequence_number), 0) + 1 FROM performances WHERE challenge_id=$1)
		)`, challengeID, contestantUserID).Scan(&seq)
	return seq
}

func (r *Repo) EnsurePerformance(ctx context.Context, challengeID, contestantUserID string) (*Performance, error) {
	seq := r.nextSequence(ctx, challengeID, contestantUserID)
	var p Performance
	err := r.pool.QueryRow(ctx, `
		INSERT INTO performances (challenge_id, contestant_user_id, sequence_number, status)
		VALUES ($1,$2,$3,'READY')
		ON CONFLICT (challenge_id, contestant_user_id) DO UPDATE SET
			updated_at = performances.updated_at
		RETURNING id, challenge_id, contestant_user_id, sequence_number, status, started_at, finished_at, created_at, updated_at`,
		challengeID, contestantUserID, seq,
	).Scan(&p.ID, &p.ChallengeID, &p.ContestantUserID, &p.SequenceNumber, &p.Status, &p.StartedAt, &p.FinishedAt, &p.CreatedAt, &p.UpdatedAt)
	return &p, err
}

func (r *Repo) FinishOpenPerformances(ctx context.Context, challengeID, exceptID string) error {
	if exceptID == "" {
		_, err := r.pool.Exec(ctx, `
			UPDATE performances SET status='FINISHED', finished_at=COALESCE(finished_at, now()), updated_at=now()
			WHERE challenge_id=$1 AND status NOT IN ('FINISHED','CANCELLED')`, challengeID)
		return err
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE performances SET status='FINISHED', finished_at=COALESCE(finished_at, now()), updated_at=now()
		WHERE challenge_id=$1 AND id <> $2 AND status NOT IN ('FINISHED','CANCELLED')`, challengeID, exceptID)
	return err
}

func (r *Repo) ResetPerformances(ctx context.Context, challengeID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE performances
		SET status='READY', started_at=NULL, finished_at=NULL, speech_duration_seconds=NULL, updated_at=now()
		WHERE challenge_id=$1 AND status <> 'CANCELLED'`, challengeID)
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

func (r *Repo) ReplaceDraw(ctx context.Context, challengeID string, userIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM evaluation_draw_entries WHERE challenge_id=$1`, challengeID); err != nil {
		return err
	}
	for i, uid := range userIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO evaluation_draw_entries (challenge_id, contestant_user_id, draw_number)
			VALUES ($1,$2,$3)`, challengeID, uid, i+1); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repo) BumpSessionRevision(ctx context.Context, challengeID, actorID string) (*Session, error) {
	if _, err := r.EnsureSession(ctx, challengeID); err != nil {
		return nil, err
	}
	return scanSession(r.pool.QueryRow(ctx, `
		UPDATE evaluation_sessions
		SET revision = revision + 1, controlled_by = $2, updated_at = now()
		WHERE challenge_id = $1
		RETURNING `+sessionCols, challengeID, actorID))
}

func (r *Repo) ListMyDraws(ctx context.Context, contestID, userID string) ([]MyDrawSummary, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT cc.id, d.draw_number,
		       (SELECT COUNT(*)::int FROM evaluation_draw_entries e WHERE e.challenge_id = cc.id)
		FROM contest_challenges cc
		LEFT JOIN evaluation_draw_entries d ON d.challenge_id = cc.id AND d.contestant_user_id = $2
		WHERE cc.contest_id = $1 AND cc.status = 'PUBLISHED'
		ORDER BY cc.sort_order, cc.created_at`, contestID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []MyDrawSummary
	for rows.Next() {
		var m MyDrawSummary
		if err := rows.Scan(&m.ChallengeID, &m.MyDrawNumber, &m.Total); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	if list == nil {
		list = []MyDrawSummary{}
	}
	return list, rows.Err()
}

func (r *Repo) SetCurrentContestant(ctx context.Context, challengeID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE evaluation_sessions
		SET current_contestant_user_id=$2, updated_at=now()
		WHERE challenge_id=$1`, challengeID, userID)
	return err
}

func (r *Repo) FillCurrentIfEmpty(ctx context.Context, challengeID, userID string) (*Session, error) {
	s, err := scanSession(r.pool.QueryRow(ctx, `
		UPDATE evaluation_sessions
		SET current_contestant_user_id=$2, updated_at=now()
		WHERE challenge_id=$1
		  AND current_contestant_user_id IS NULL
		  AND state IN ('NOT_STARTED','PREPARING')
		RETURNING `+sessionCols, challengeID, userID))
	if errors.Is(err, ErrNotFound) {
		return r.SessionByChallenge(ctx, challengeID)
	}
	return s, err
}

func (r *Repo) UpdatePhaseDuration(ctx context.Context, schemeID, mapsToState, title string, seconds, sortOrder int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE performance_phase_templates
		SET duration_seconds=$3, scoring_allowed=TRUE
		WHERE evaluation_scheme_id=$1 AND maps_to_state=$2`, schemeID, mapsToState, seconds)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO performance_phase_templates
			(evaluation_scheme_id, title, duration_seconds, scoring_allowed, maps_to_state, sort_order)
		VALUES ($1,$2,$3,TRUE,$4,$5)`, schemeID, title, seconds, mapsToState, sortOrder)
	return err
}
