package evaluation

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *Repo) InsertLifeEvent(ctx context.Context, e LifeEvent) (*LifeEvent, error) {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO life_events (
			challenge_id, contestant_user_id, question_number, delta, reason,
			created_by_user_id, reverses_life_event_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, challenge_id, contestant_user_id, question_number, delta, reason,
			created_by_user_id, reverses_life_event_id, created_at`,
		e.ChallengeID, e.ContestantUserID, e.QuestionNumber, e.Delta, e.Reason,
		e.CreatedByUserID, e.ReversesLifeEventID,
	).Scan(
		&e.ID, &e.ChallengeID, &e.ContestantUserID, &e.QuestionNumber, &e.Delta, &e.Reason,
		&e.CreatedByUserID, &e.ReversesLifeEventID, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repo) ListLifeEvents(ctx context.Context, challengeID string, createdBy *string) ([]LifeEvent, error) {
	q := `
		SELECT id, challenge_id, contestant_user_id, question_number, delta, reason,
			created_by_user_id, reverses_life_event_id, created_at
		FROM life_events
		WHERE challenge_id = $1`
	args := []any{challengeID}
	if createdBy != nil {
		q += ` AND created_by_user_id = $2`
		args = append(args, *createdBy)
	}
	q += ` ORDER BY created_at, id`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []LifeEvent{}
	for rows.Next() {
		var e LifeEvent
		if err := rows.Scan(
			&e.ID, &e.ChallengeID, &e.ContestantUserID, &e.QuestionNumber, &e.Delta, &e.Reason,
			&e.CreatedByUserID, &e.ReversesLifeEventID, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func (r *Repo) OpenWrongEvent(ctx context.Context, challengeID, contestantID, juryID string, question int) (*LifeEvent, error) {
	var e LifeEvent
	err := r.pool.QueryRow(ctx, `
		SELECT le.id, le.challenge_id, le.contestant_user_id, le.question_number, le.delta, le.reason,
			le.created_by_user_id, le.reverses_life_event_id, le.created_at
		FROM life_events le
		WHERE le.challenge_id = $1
		  AND le.contestant_user_id = $2
		  AND le.created_by_user_id = $3
		  AND le.question_number = $4
		  AND le.reason = $5
		  AND NOT EXISTS (
			SELECT 1 FROM life_events r WHERE r.reverses_life_event_id = le.id
		  )
		ORDER BY le.created_at DESC, le.id DESC
		LIMIT 1`,
		challengeID, contestantID, juryID, question, ReasonWrongAnswer,
	).Scan(
		&e.ID, &e.ChallengeID, &e.ContestantUserID, &e.QuestionNumber, &e.Delta, &e.Reason,
		&e.CreatedByUserID, &e.ReversesLifeEventID, &e.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *Repo) DeleteLifeEvents(ctx context.Context, challengeID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE life_events SET reverses_life_event_id = NULL WHERE challenge_id = $1`, challengeID)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM life_events WHERE challenge_id = $1`, challengeID)
	return err
}

func (r *Repo) ChallengeOperator(ctx context.Context, challengeID string) (*string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		SELECT user_id FROM evaluation_staff_assignments
		WHERE challenge_id = $1 AND role = $2 AND active
		ORDER BY updated_at DESC
		LIMIT 1`, challengeID, RoleTrialOperator).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *Repo) SetChallengeOperator(ctx context.Context, contestID, challengeID, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE evaluation_staff_assignments
		SET active = FALSE, updated_at = now()
		WHERE challenge_id = $1 AND role = $2 AND active`,
		challengeID, RoleTrialOperator); err != nil {
		return err
	}
	if userID != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO evaluation_staff_assignments (contest_id, challenge_id, user_id, role, active)
			VALUES ($1, $2, $3, $4, TRUE)`,
			contestID, challengeID, userID, RoleTrialOperator); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repo) UpsertQuestionKey(ctx context.Context, challengeID string, question int, answer, actorID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO evaluation_question_keys (challenge_id, question_number, correct_answer, set_by_user_id)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (challenge_id, question_number) DO UPDATE SET
			correct_answer = EXCLUDED.correct_answer,
			set_by_user_id = EXCLUDED.set_by_user_id,
			updated_at = now()`,
		challengeID, question, answer, actorID)
	return err
}

func (r *Repo) QuestionKey(ctx context.Context, challengeID string, question int) (string, error) {
	var answer string
	err := r.pool.QueryRow(ctx, `
		SELECT correct_answer FROM evaluation_question_keys
		WHERE challenge_id = $1 AND question_number = $2`, challengeID, question).Scan(&answer)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return answer, err
}

func (r *Repo) ListQuestionKeys(ctx context.Context, challengeID string) (map[int]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT question_number, correct_answer
		FROM evaluation_question_keys WHERE challenge_id = $1`, challengeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var n int
		var a string
		if err := rows.Scan(&n, &a); err != nil {
			return nil, err
		}
		out[n] = a
	}
	return out, rows.Err()
}

func (r *Repo) DeleteQuestionKeys(ctx context.Context, challengeID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM evaluation_question_keys WHERE challenge_id = $1`, challengeID)
	return err
}

func (r *Repo) ReplaceQuestionKeys(ctx context.Context, challengeID, actorID string, keys map[int]string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM evaluation_question_keys WHERE challenge_id = $1`, challengeID); err != nil {
		return err
	}
	for n, answer := range keys {
		if _, err := tx.Exec(ctx, `
			INSERT INTO evaluation_question_keys (challenge_id, question_number, correct_answer, set_by_user_id)
			VALUES ($1,$2,$3,$4)`, challengeID, n, answer, actorID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repo) SetQuestionCount(ctx context.Context, challengeID string, count int) error {
	if count < MinLiveQuestions {
		return ErrValidation
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE evaluation_sessions
		SET question_count = $2,
		    current_question_number = CASE
		        WHEN current_question_number > $2 THEN $2
		        ELSE current_question_number
		    END,
		    updated_at = now()
		WHERE challenge_id = $1`, challengeID, count)
	return err
}

func (r *Repo) UpsertAnswerMark(ctx context.Context, challengeID, contestantID, juryID string, question int, answer string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO life_answer_marks (
			challenge_id, contestant_user_id, jury_user_id, question_number, answer
		) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (challenge_id, contestant_user_id, jury_user_id, question_number) DO UPDATE SET
			answer = EXCLUDED.answer,
			updated_at = now()`,
		challengeID, contestantID, juryID, question, answer)
	return err
}

func (r *Repo) ListAnswerMarks(ctx context.Context, challengeID string, juryID *string, question *int) ([]AnswerMark, error) {
	q := `
		SELECT contestant_user_id, jury_user_id, question_number, answer
		FROM life_answer_marks
		WHERE challenge_id = $1`
	args := []any{challengeID}
	n := 2
	if juryID != nil {
		q += fmt.Sprintf(` AND jury_user_id = $%d`, n)
		args = append(args, *juryID)
		n++
	}
	if question != nil {
		q += fmt.Sprintf(` AND question_number = $%d`, n)
		args = append(args, *question)
	}
	q += ` ORDER BY question_number, contestant_user_id`
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []AnswerMark{}
	for rows.Next() {
		var m AnswerMark
		if err := rows.Scan(&m.ContestantUserID, &m.JuryUserID, &m.QuestionNumber, &m.Answer); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *Repo) DeleteAnswerMarks(ctx context.Context, challengeID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM life_answer_marks WHERE challenge_id = $1`, challengeID)
	return err
}
