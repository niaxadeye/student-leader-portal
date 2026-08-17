package eventtasks

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/modules/audit"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventpermissions"
	"github.com/eazytech/student-leader-cabinet/internal/modules/points"
)

type pointAppender interface {
	AppendTx(context.Context, pgx.Tx, points.AppendInput) (*points.Entry, bool, error)
}

type txAuditor interface {
	LogEntryTx(context.Context, pgx.Tx, audit.Entry) error
}

type Repo struct {
	pool   *pgxpool.Pool
	points pointAppender
	audit  txAuditor
}

func NewRepo(pool *pgxpool.Pool, pointRepo pointAppender, auditor txAuditor) *Repo {
	return &Repo{pool: pool, points: pointRepo, audit: auditor}
}

func (r *Repo) Can(ctx context.Context, userID, contestID, permission string) (bool, error) {
	var allowed bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM contests c WHERE c.id=$2 AND (
				c.owner_user_id=$1 OR EXISTS (
					SELECT 1 FROM event_staff_permissions ep
					WHERE ep.user_id=$1 AND ep.contest_id=c.id
					  AND ep.permission=ANY($3::varchar[]))))`,
		userID, contestID, eventpermissions.GrantsFor(permission)).Scan(&allowed)
	return allowed, err
}

const taskCols = `id, contest_id, title, description, image_key, icon, points,
	starts_at, ends_at, status, sort_order, allowed_submission_types, created_at, updated_at`

func (r *Repo) List(ctx context.Context, contestID string) ([]Task, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+taskCols+` FROM event_tasks
		WHERE contest_id=$1 ORDER BY sort_order, created_at`, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (r *Repo) ParticipantList(ctx context.Context, contestID, participantID string) ([]Task, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+taskCols+` FROM event_tasks
		WHERE contest_id=$1 AND status='ACTIVE' ORDER BY sort_order, created_at`, contestID)
	if err != nil {
		return nil, err
	}
	list, err := scanTasks(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range list {
		submission, subErr := r.ParticipantSubmission(ctx, contestID, list[i].ID, participantID)
		if subErr == nil {
			list[i].Submission = submission
		} else if !errors.Is(subErr, ErrNotFound) {
			return nil, subErr
		}
	}
	return list, nil
}

func (r *Repo) Get(ctx context.Context, contestID, taskID string) (*Task, error) {
	return scanTask(r.pool.QueryRow(ctx, `SELECT `+taskCols+` FROM event_tasks
		WHERE contest_id=$1 AND id=$2`, contestID, taskID))
}

func (r *Repo) Create(ctx context.Context, contestID string, input TaskInput) (*Task, error) {
	return scanTask(r.pool.QueryRow(ctx, `
		INSERT INTO event_tasks
		  (contest_id,title,description,icon,points,starts_at,ends_at,sort_order,allowed_submission_types)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING `+taskCols, contestID, input.Title, input.Description, input.Icon,
		input.Points, input.StartsAt, input.EndsAt, input.SortOrder, input.AllowedSubmissionTypes))
}

func (r *Repo) Update(ctx context.Context, contestID, taskID string, input TaskInput) (*Task, error) {
	task, err := scanTask(r.pool.QueryRow(ctx, `
		UPDATE event_tasks SET title=$3, description=$4, icon=$5, points=$6,
		       starts_at=$7, ends_at=$8, sort_order=$9, allowed_submission_types=$10,
		       updated_at=now()
		WHERE contest_id=$1 AND id=$2 AND status<>'ARCHIVED'
		RETURNING `+taskCols, contestID, taskID, input.Title, input.Description, input.Icon,
		input.Points, input.StartsAt, input.EndsAt, input.SortOrder, input.AllowedSubmissionTypes))
	if errors.Is(err, ErrNotFound) {
		return nil, r.notFoundOrTransition(ctx, contestID, taskID)
	}
	return task, err
}

func (r *Repo) Transition(ctx context.Context, contestID, taskID string, allowedFrom []string, to string) (*Task, error) {
	task, err := scanTask(r.pool.QueryRow(ctx, `
		UPDATE event_tasks SET status=$4, updated_at=now()
		WHERE contest_id=$1 AND id=$2 AND status=ANY($3::text[])
		RETURNING `+taskCols, contestID, taskID, allowedFrom, to))
	if errors.Is(err, ErrNotFound) {
		return nil, r.notFoundOrTransition(ctx, contestID, taskID)
	}
	return task, err
}

func (r *Repo) Delete(ctx context.Context, contestID, taskID string) error {
	ct, err := r.pool.Exec(ctx, `DELETE FROM event_tasks t
		WHERE t.contest_id=$1 AND t.id=$2 AND t.status='DRAFT'
		  AND NOT EXISTS(SELECT 1 FROM event_task_submissions s WHERE s.task_id=t.id)`,
		contestID, taskID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() > 0 {
		return nil
	}
	return r.notFoundOrTransition(ctx, contestID, taskID)
}

func (r *Repo) SetImage(ctx context.Context, contestID, taskID string, imageKey *string) (*Task, *string, error) {
	var task Task
	var previous *string
	err := r.pool.QueryRow(ctx, `
		WITH old AS (SELECT image_key FROM event_tasks WHERE contest_id=$1 AND id=$2)
		UPDATE event_tasks SET image_key=$3, updated_at=now()
		WHERE contest_id=$1 AND id=$2 AND status<>'ARCHIVED'
		RETURNING id, contest_id, title, description, image_key, icon, points,
		  starts_at, ends_at, status, sort_order, allowed_submission_types, created_at, updated_at,
		  (SELECT image_key FROM old)`, contestID, taskID, imageKey).
		Scan(&task.ID, &task.ContestID, &task.Title, &task.Description, &task.ImageKey,
			&task.Icon, &task.Points, &task.StartsAt, &task.EndsAt, &task.Status,
			&task.SortOrder, &task.AllowedSubmissionTypes, &task.CreatedAt, &task.UpdatedAt,
			&previous)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, r.notFoundOrTransition(ctx, contestID, taskID)
	}
	return &task, previous, err
}

func (r *Repo) notFoundOrTransition(ctx context.Context, contestID, taskID string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM event_tasks
		WHERE contest_id=$1 AND id=$2)`, contestID, taskID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrInvalidTransition
}

func (r *Repo) ParticipantSubmission(ctx context.Context, contestID, taskID, participantID string) (*Submission, error) {
	submission, err := scanSubmission(r.pool.QueryRow(ctx, submissionSelect+`
		WHERE s.contest_id=$1 AND s.task_id=$2 AND s.event_participant_id=$3`,
		contestID, taskID, participantID))
	if err != nil {
		return nil, err
	}
	return r.withAttempts(ctx, submission)
}

func (r *Repo) SubmitAttempt(ctx context.Context, params SubmitParams) (submission *Submission, err error) {
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var taskStatus, taskTitle, participantStatus, contestStatus string
		var startsAt, endsAt *time.Time
		var allowed []string
		if err := tx.QueryRow(ctx, `
			SELECT t.status, t.title, t.starts_at, t.ends_at, t.allowed_submission_types,
			       p.status, c.status
			FROM event_tasks t
			JOIN event_participants p ON p.contest_id=t.contest_id AND p.id=$3
			JOIN contests c ON c.id=t.contest_id
			WHERE t.contest_id=$1 AND t.id=$2 FOR SHARE OF t`,
			params.ContestID, params.TaskID, params.EventParticipantID).
			Scan(&taskStatus, &taskTitle, &startsAt, &endsAt, &allowed,
				&participantStatus, &contestStatus); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if taskStatus != StatusActive || participantStatus != "ACTIVE" || contestStatus != "ACTIVE" ||
			(startsAt != nil && params.Now.Before(*startsAt)) ||
			(endsAt != nil && params.Now.After(*endsAt)) {
			return ErrSubmissionClosed
		}
		for _, asset := range params.Assets {
			if !slices.Contains(allowed, asset.Type) {
				return ErrValidation
			}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO event_task_submissions
			  (contest_id,task_id,event_participant_id,status,current_attempt)
			VALUES ($1,$2,$3,'PENDING',0)
			ON CONFLICT (task_id,event_participant_id) DO NOTHING`,
			params.ContestID, params.TaskID, params.EventParticipantID); err != nil {
			return err
		}
		var submissionID, status string
		var currentAttempt int
		if err := tx.QueryRow(ctx, `SELECT id,status,current_attempt FROM event_task_submissions
			WHERE task_id=$1 AND event_participant_id=$2 FOR UPDATE`,
			params.TaskID, params.EventParticipantID).Scan(&submissionID, &status, &currentAttempt); err != nil {
			return err
		}
		if currentAttempt > 0 && status != SubmissionRejected {
			return ErrInvalidTransition
		}
		nextAttempt := currentAttempt + 1
		var attemptID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO event_task_submission_attempts
			  (contest_id,submission_id,attempt_number,status,participant_comment)
			VALUES ($1,$2,$3,'PENDING',$4) RETURNING id`,
			params.ContestID, submissionID, nextAttempt, params.ParticipantComment).Scan(&attemptID); err != nil {
			return err
		}
		for _, asset := range params.Assets {
			if _, err := tx.Exec(ctx, `
				INSERT INTO event_task_submission_assets
				  (contest_id,attempt_id,type,object_key,external_url,original_name,mime_type,size_bytes,sort_order)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				params.ContestID, attemptID, asset.Type, asset.ObjectKey, asset.ExternalURL,
				asset.OriginalName, asset.MimeType, asset.SizeBytes, asset.SortOrder); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE event_task_submissions SET status='PENDING',current_attempt=$2,
			  participant_comment=$3,moderator_comment=NULL,reviewed_by_user_id=NULL,
			  submitted_at=now(),reviewed_at=NULL,updated_at=now()
			WHERE id=$1`, submissionID, nextAttempt, params.ParticipantComment); err != nil {
			return err
		}
		return r.audit.LogEntryTx(ctx, tx, audit.Entry{
			EventParticipantID: params.EventParticipantID, ContestID: params.ContestID,
			Action: "TASK_SUBMISSION_SUBMITTED", EntityType: "event_task_submission",
			EntityID: submissionID, Metadata: map[string]any{
				"task_id": params.TaskID, "task_title": taskTitle, "attempt": nextAttempt,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	return r.ParticipantSubmission(ctx, params.ContestID, params.TaskID, params.EventParticipantID)
}

func (r *Repo) ModerationList(ctx context.Context, contestID, status string) ([]Submission, error) {
	rows, err := r.pool.Query(ctx, submissionSelect+`
		WHERE s.contest_id=$1 AND s.status=$2 ORDER BY s.submitted_at ASC`, contestID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Submission, 0)
	for rows.Next() {
		submission, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *submission)
	}
	return list, rows.Err()
}

func (r *Repo) SubmissionByID(ctx context.Context, contestID, submissionID string) (*Submission, error) {
	submission, err := scanSubmission(r.pool.QueryRow(ctx, submissionSelect+`
		WHERE s.contest_id=$1 AND s.id=$2`, contestID, submissionID))
	if err != nil {
		return nil, err
	}
	return r.withAttempts(ctx, submission)
}

func (r *Repo) Approve(ctx context.Context, actor Actor, contestID, submissionID, comment string) (result *ModerationResult, err error) {
	replayed := false
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var taskID, participantID, status, taskTitle string
		var currentAttempt int
		var rewardGrantedAt *time.Time
		var taskPoints int64
		err := tx.QueryRow(ctx, `
			SELECT s.task_id,s.event_participant_id,s.status,s.current_attempt,s.reward_granted_at,
			       t.points,t.title
			FROM event_task_submissions s JOIN event_tasks t ON t.id=s.task_id
			WHERE s.contest_id=$1 AND s.id=$2 FOR UPDATE OF s`, contestID, submissionID).
			Scan(&taskID, &participantID, &status, &currentAttempt, &rewardGrantedAt, &taskPoints, &taskTitle)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == SubmissionApproved && rewardGrantedAt != nil {
			replayed = true
			return nil
		}
		if status != SubmissionPending {
			return ErrInvalidTransition
		}
		commentValue := nilIfEmpty(comment)
		if _, err := tx.Exec(ctx, `UPDATE event_task_submission_attempts
			SET status='APPROVED',moderator_comment=$3,reviewed_by_user_id=$4,reviewed_at=now()
			WHERE submission_id=$1 AND attempt_number=$2 AND status='PENDING'`,
			submissionID, currentAttempt, commentValue, actor.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE event_task_submissions
			SET status='APPROVED',moderator_comment=$2,reviewed_by_user_id=$3,
			    reviewed_at=now(),reward_granted_at=now(),updated_at=now()
			WHERE id=$1`, submissionID, commentValue, actor.UserID); err != nil {
			return err
		}
		sourceType, sourceID, actorID := "event_task_submission", submissionID, actor.UserID
		_, created, err := r.points.AppendTx(ctx, tx, points.AppendInput{
			ContestID: contestID, EventParticipantID: participantID, Amount: taskPoints,
			Type: points.TypeTaskReward, SourceType: &sourceType, SourceID: &sourceID,
			Description:     fmt.Sprintf("Выполнение задания «%s»", taskTitle),
			CreatedByUserID: &actorID, IdempotencyKey: "task-reward:" + submissionID,
		})
		if err != nil {
			return err
		}
		if !created {
			return points.ErrIdempotencyConflict
		}
		return r.audit.LogEntryTx(ctx, tx, audit.Entry{
			ActorUserID: actor.UserID, ContestID: contestID,
			Action: "TASK_SUBMISSION_APPROVED", EntityType: "event_task_submission",
			EntityID: submissionID, Metadata: map[string]any{
				"task_id": taskID, "participant_id": participantID,
				"attempt": currentAttempt, "points": taskPoints,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	submission, err := r.SubmissionByID(ctx, contestID, submissionID)
	if err != nil {
		return nil, err
	}
	return &ModerationResult{Submission: *submission, Replayed: replayed}, nil
}

func (r *Repo) Reject(ctx context.Context, actor Actor, contestID, submissionID, comment string) (result *ModerationResult, err error) {
	replayed := false
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var taskID, participantID, status string
		var currentAttempt int
		err := tx.QueryRow(ctx, `SELECT task_id,event_participant_id,status,current_attempt
			FROM event_task_submissions WHERE contest_id=$1 AND id=$2 FOR UPDATE`,
			contestID, submissionID).Scan(&taskID, &participantID, &status, &currentAttempt)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == SubmissionRejected {
			replayed = true
			return nil
		}
		if status != SubmissionPending {
			return ErrInvalidTransition
		}
		if _, err := tx.Exec(ctx, `UPDATE event_task_submission_attempts
			SET status='REJECTED',moderator_comment=$3,reviewed_by_user_id=$4,reviewed_at=now()
			WHERE submission_id=$1 AND attempt_number=$2 AND status='PENDING'`,
			submissionID, currentAttempt, comment, actor.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE event_task_submissions
			SET status='REJECTED',moderator_comment=$2,reviewed_by_user_id=$3,
			    reviewed_at=now(),updated_at=now() WHERE id=$1`,
			submissionID, comment, actor.UserID); err != nil {
			return err
		}
		return r.audit.LogEntryTx(ctx, tx, audit.Entry{
			ActorUserID: actor.UserID, ContestID: contestID,
			Action: "TASK_SUBMISSION_REJECTED", EntityType: "event_task_submission",
			EntityID: submissionID, Metadata: map[string]any{
				"task_id": taskID, "participant_id": participantID,
				"attempt": currentAttempt, "reason": comment,
			},
		})
	})
	if err != nil {
		return nil, err
	}
	submission, err := r.SubmissionByID(ctx, contestID, submissionID)
	if err != nil {
		return nil, err
	}
	return &ModerationResult{Submission: *submission, Replayed: replayed}, nil
}

func (r *Repo) ParticipantAsset(ctx context.Context, participantID, assetID string) (*Asset, error) {
	return scanAsset(r.pool.QueryRow(ctx, assetSelect+`
		JOIN event_task_submission_attempts a ON a.id=x.attempt_id
		JOIN event_task_submissions s ON s.id=a.submission_id
		WHERE x.id=$1 AND s.event_participant_id=$2`, assetID, participantID))
}

func (r *Repo) AdminAsset(ctx context.Context, contestID, submissionID, assetID string) (*Asset, error) {
	return scanAsset(r.pool.QueryRow(ctx, assetSelect+`
		JOIN event_task_submission_attempts a ON a.id=x.attempt_id
		WHERE x.id=$1 AND x.contest_id=$2 AND a.submission_id=$3`,
		assetID, contestID, submissionID))
}

const submissionSelect = `
	SELECT s.id,s.contest_id,s.task_id,s.event_participant_id,p.full_name,t.title,
	       s.status,s.current_attempt,s.participant_comment,s.moderator_comment,
	       s.reviewed_by_user_id,s.submitted_at,s.reviewed_at,s.reward_granted_at,
	       s.created_at,s.updated_at,t.points,t.allowed_submission_types
	FROM event_task_submissions s
	JOIN event_participants p ON p.id=s.event_participant_id
	JOIN event_tasks t ON t.id=s.task_id `

func (r *Repo) withAttempts(ctx context.Context, submission *Submission) (*Submission, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id,attempt_number,status,participant_comment,moderator_comment,
		       reviewed_by_user_id,submitted_at,reviewed_at
		FROM event_task_submission_attempts WHERE submission_id=$1
		ORDER BY attempt_number DESC`, submission.ID)
	if err != nil {
		return nil, err
	}
	attempts := make([]Attempt, 0)
	for rows.Next() {
		var attempt Attempt
		if err := rows.Scan(&attempt.ID, &attempt.AttemptNumber, &attempt.Status,
			&attempt.ParticipantComment, &attempt.ModeratorComment,
			&attempt.ReviewedByUserID, &attempt.SubmittedAt, &attempt.ReviewedAt); err != nil {
			rows.Close()
			return nil, err
		}
		attempt.Assets = []Asset{}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range attempts {
		assets, err := r.assetsForAttempt(ctx, attempts[i].ID)
		if err != nil {
			return nil, err
		}
		attempts[i].Assets = assets
	}
	submission.Attempts = attempts
	return submission, nil
}

func (r *Repo) assetsForAttempt(ctx context.Context, attemptID string) ([]Asset, error) {
	rows, err := r.pool.Query(ctx, assetSelect+`WHERE x.attempt_id=$1 ORDER BY x.sort_order,x.created_at`, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	assets := make([]Asset, 0)
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, *asset)
	}
	return assets, rows.Err()
}

const assetSelect = `SELECT x.id,x.attempt_id,x.type,x.object_key,x.external_url,
	x.original_name,x.mime_type,x.size_bytes,x.sort_order,x.created_at
	FROM event_task_submission_assets x `

type rowScanner interface{ Scan(...any) error }

func scanTask(row rowScanner) (*Task, error) {
	var task Task
	err := row.Scan(&task.ID, &task.ContestID, &task.Title, &task.Description,
		&task.ImageKey, &task.Icon, &task.Points, &task.StartsAt, &task.EndsAt,
		&task.Status, &task.SortOrder, &task.AllowedSubmissionTypes,
		&task.CreatedAt, &task.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func scanTasks(rows pgx.Rows) ([]Task, error) {
	list := make([]Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *task)
	}
	return list, rows.Err()
}

func scanSubmission(row rowScanner) (*Submission, error) {
	var submission Submission
	err := row.Scan(&submission.ID, &submission.ContestID, &submission.TaskID,
		&submission.EventParticipantID, &submission.ParticipantName, &submission.TaskTitle,
		&submission.Status, &submission.CurrentAttempt, &submission.ParticipantComment,
		&submission.ModeratorComment, &submission.ReviewedByUserID,
		&submission.SubmittedAt, &submission.ReviewedAt, &submission.RewardGrantedAt,
		&submission.CreatedAt, &submission.UpdatedAt, &submission.Points,
		&submission.AllowedSubmissionTypes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

func scanAsset(row rowScanner) (*Asset, error) {
	var asset Asset
	err := row.Scan(&asset.ID, &asset.AttemptID, &asset.Type, &asset.ObjectKey,
		&asset.ExternalURL, &asset.OriginalName, &asset.MimeType, &asset.SizeBytes,
		&asset.SortOrder, &asset.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
