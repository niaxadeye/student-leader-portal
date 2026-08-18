package lectures

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/modules/audit"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventpermissions"
	"github.com/eazytech/student-leader-cabinet/internal/modules/points"
)

type pointAppender interface {
	AppendTx(ctx context.Context, tx pgx.Tx, input points.AppendInput) (*points.Entry, bool, error)
}

type txAuditor interface {
	LogEntryTx(ctx context.Context, tx pgx.Tx, entry audit.Entry) error
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
			SELECT 1 FROM contests c
			WHERE c.id=$2 AND (
				c.owner_user_id=$1 OR EXISTS (
					SELECT 1 FROM event_staff_permissions ep
					WHERE ep.user_id=$1 AND ep.contest_id=c.id
					  AND ep.permission=ANY($3::varchar[]))))`,
		userID, contestID, eventpermissions.GrantsFor(permission)).Scan(&allowed)
	return allowed, err
}

func (r *Repo) List(ctx context.Context, contestID string) ([]Lecture, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contest_id, title, description, points, starts_at, ends_at,
		       attendance_starts_at, attendance_ends_at, status, created_at, updated_at
		FROM lectures WHERE contest_id=$1
		ORDER BY starts_at NULLS LAST, created_at DESC`, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Lecture, 0)
	for rows.Next() {
		lecture, err := scanLecture(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *lecture)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return r.withDirections(ctx, r.pool, list)
}

func (r *Repo) Get(ctx context.Context, contestID, lectureID string) (*Lecture, error) {
	lecture, err := scanLecture(r.pool.QueryRow(ctx, `
		SELECT id, contest_id, title, description, points, starts_at, ends_at,
		       attendance_starts_at, attendance_ends_at, status, created_at, updated_at
		FROM lectures WHERE contest_id=$1 AND id=$2`, contestID, lectureID))
	if err != nil {
		return nil, err
	}
	list, err := r.withDirections(ctx, r.pool, []Lecture{*lecture})
	if err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (r *Repo) Create(ctx context.Context, contestID string, input LectureInput) (*Lecture, error) {
	var lecture *Lecture
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		created, err := scanLecture(tx.QueryRow(ctx, `
			INSERT INTO lectures
			  (contest_id, title, description, points, starts_at, ends_at,
			   attendance_starts_at, attendance_ends_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			RETURNING id, contest_id, title, description, points, starts_at, ends_at,
			          attendance_starts_at, attendance_ends_at, status, created_at, updated_at`,
			contestID, input.Title, input.Description, input.Points, input.StartsAt, input.EndsAt,
			input.AttendanceStartsAt, input.AttendanceEndsAt))
		if err != nil {
			return err
		}
		if err := replaceLectureDirections(ctx, tx, contestID, created.ID, input.DirectionIDs); err != nil {
			return err
		}
		lecture = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	list, err := r.withDirections(ctx, r.pool, []Lecture{*lecture})
	if err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (r *Repo) Update(ctx context.Context, contestID, lectureID string, input LectureInput) (*Lecture, error) {
	var lecture *Lecture
	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		updated, err := scanLecture(tx.QueryRow(ctx, `
			UPDATE lectures SET title=$3, description=$4, points=$5, starts_at=$6, ends_at=$7,
			       attendance_starts_at=$8, attendance_ends_at=$9, updated_at=now()
			WHERE contest_id=$1 AND id=$2 AND status <> 'FINISHED'
			RETURNING id, contest_id, title, description, points, starts_at, ends_at,
			          attendance_starts_at, attendance_ends_at, status, created_at, updated_at`,
			contestID, lectureID, input.Title, input.Description, input.Points,
			input.StartsAt, input.EndsAt, input.AttendanceStartsAt, input.AttendanceEndsAt))
		if errors.Is(err, ErrNotFound) {
			return r.notFoundOrTransition(ctx, contestID, lectureID)
		}
		if err != nil {
			return err
		}
		if err := replaceLectureDirections(ctx, tx, contestID, updated.ID, input.DirectionIDs); err != nil {
			return err
		}
		lecture = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	list, err := r.withDirections(ctx, r.pool, []Lecture{*lecture})
	if err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (r *Repo) Transition(ctx context.Context, contestID, lectureID, from, to string) (*Lecture, error) {
	lecture, err := scanLecture(r.pool.QueryRow(ctx, `
		UPDATE lectures SET status=$4, updated_at=now()
		WHERE contest_id=$1 AND id=$2 AND status=$3
		RETURNING id, contest_id, title, description, points, starts_at, ends_at,
		          attendance_starts_at, attendance_ends_at, status, created_at, updated_at`,
		contestID, lectureID, from, to))
	if errors.Is(err, ErrNotFound) {
		return nil, r.notFoundOrTransition(ctx, contestID, lectureID)
	}
	if err != nil {
		return nil, err
	}
	list, err := r.withDirections(ctx, r.pool, []Lecture{*lecture})
	if err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (r *Repo) Delete(ctx context.Context, contestID, lectureID string) error {
	ct, err := r.pool.Exec(ctx, `
		DELETE FROM lectures l
		WHERE l.contest_id=$1 AND l.id=$2 AND l.status='DRAFT'
		  AND NOT EXISTS (SELECT 1 FROM lecture_attendance a WHERE a.lecture_id=l.id)`,
		contestID, lectureID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() > 0 {
		return nil
	}
	var status string
	var hasAttendance bool
	err = r.pool.QueryRow(ctx, `
		SELECT l.status, EXISTS(SELECT 1 FROM lecture_attendance a WHERE a.lecture_id=l.id)
		FROM lectures l WHERE l.contest_id=$1 AND l.id=$2`, contestID, lectureID).
		Scan(&status, &hasAttendance)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if hasAttendance {
		return ErrLectureHasAttendance
	}
	return ErrInvalidTransition
}

func (r *Repo) notFoundOrTransition(ctx context.Context, contestID, lectureID string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM lectures WHERE contest_id=$1 AND id=$2)`, contestID, lectureID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return ErrInvalidTransition
}

func (r *Repo) CreateCode(
	ctx context.Context,
	contestID, participantID, nonceHash string,
	expiresAt time.Time,
) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		// Keep the table bounded without a separate worker. Used codes are historical
		// transport data, not attendance truth, and can be discarded after one day.
		if _, err := tx.Exec(ctx, `DELETE FROM participant_qr_codes
			WHERE expires_at < now() - interval '24 hours'`); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO participant_qr_codes
			  (contest_id, event_participant_id, nonce_hash, expires_at)
			SELECT $1,$2,$3,$4 FROM event_participants p JOIN contests c ON c.id=p.contest_id
			WHERE p.contest_id=$1 AND p.id=$2 AND p.status='ACTIVE' AND c.status='ACTIVE'`,
			contestID, participantID, nonceHash, expiresAt)
		if err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM participant_qr_codes WHERE nonce_hash=$1)`, nonceHash).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrParticipantInactive
		}
		return nil
	})
}

func (r *Repo) ScanAttendance(ctx context.Context, params ScanParams) (result *ScanResult, err error) {
	err = pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		lecture, err := scanLecture(tx.QueryRow(ctx, `
			SELECT id, contest_id, title, description, points, starts_at, ends_at,
			       attendance_starts_at, attendance_ends_at, status, created_at, updated_at
			FROM lectures WHERE contest_id=$1 AND id=$2 FOR SHARE`,
			params.ContestID, params.LectureID))
		if err != nil {
			return err
		}
		if lecture.Status != StatusActive ||
			(lecture.AttendanceStartsAt != nil && params.Now.Before(*lecture.AttendanceStartsAt)) ||
			(lecture.AttendanceEndsAt != nil && !params.Now.Before(*lecture.AttendanceEndsAt)) {
			return ErrAttendanceClosed
		}

		var participantID, participantName, participantStatus, contestStatus string
		var participantDirectionID *string
		var expiresAt time.Time
		var usedAt *time.Time
		var usedLectureID *string
		err = tx.QueryRow(ctx, `
			SELECT q.event_participant_id, p.full_name, p.status, p.direction_id, c.status,
			       q.expires_at, q.used_at, q.used_for_lecture_id
			FROM participant_qr_codes q
			JOIN event_participants p ON p.contest_id=q.contest_id AND p.id=q.event_participant_id
			JOIN contests c ON c.id=q.contest_id
			WHERE q.contest_id=$1 AND q.nonce_hash=$2
			FOR UPDATE OF q`, params.ContestID, params.Code.NonceHash).
			Scan(&participantID, &participantName, &participantStatus, &participantDirectionID,
				&contestStatus, &expiresAt, &usedAt, &usedLectureID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInvalidCode
		}
		if err != nil {
			return err
		}
		if !expiresAt.After(params.Now) || !params.Code.ExpiresAt.After(params.Now) {
			return ErrExpiredCode
		}
		if participantStatus != "ACTIVE" || contestStatus != "ACTIVE" {
			return ErrParticipantInactive
		}

		if existing, findErr := attendanceByParticipant(ctx, tx, params.LectureID, participantID); findErr == nil {
			result = &ScanResult{Attendance: *existing, AlreadyChecked: true}
			return nil
		} else if !errors.Is(findErr, ErrNotFound) {
			return findErr
		}
		restrictedIDs, err := lectureDirectionIDs(ctx, tx, params.LectureID)
		if err != nil {
			return err
		}
		if !lectureAllowsParticipant(restrictedIDs, participantDirectionID) {
			return ErrWrongDirection
		}
		if usedAt != nil || usedLectureID != nil {
			return ErrReplayedCode
		}

		attendance, err := scanAttendance(tx.QueryRow(ctx, `
			INSERT INTO lecture_attendance
			  (contest_id, lecture_id, event_participant_id, scanned_by_user_id,
			   scanner_type, points_awarded, qr_nonce_hash)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (lecture_id, event_participant_id) DO NOTHING
			RETURNING id, contest_id, lecture_id, event_participant_id, $8::text,
			          scanned_by_user_id, scanner_type, points_awarded, created_at`,
			params.ContestID, params.LectureID, participantID, params.Actor.UserID,
			params.ScannerType, lecture.Points, params.Code.NonceHash, participantName))
		if errors.Is(err, ErrNotFound) {
			existing, findErr := attendanceByParticipant(ctx, tx, params.LectureID, participantID)
			if findErr != nil {
				return findErr
			}
			if _, updateErr := tx.Exec(ctx, `UPDATE participant_qr_codes
				SET used_at=COALESCE(used_at, now()), used_for_lecture_id=COALESCE(used_for_lecture_id, $2)
				WHERE nonce_hash=$1`, params.Code.NonceHash, params.LectureID); updateErr != nil {
				return updateErr
			}
			result = &ScanResult{Attendance: *existing, AlreadyChecked: true}
			return nil
		}
		if isUniqueViolation(err) {
			return ErrReplayedCode
		}
		if err != nil {
			return err
		}

		if _, err = tx.Exec(ctx, `UPDATE participant_qr_codes
			SET used_at=now(), used_for_lecture_id=$2 WHERE nonce_hash=$1`,
			params.Code.NonceHash, params.LectureID); err != nil {
			return err
		}
		sourceType := "lecture_attendance"
		sourceID := attendance.ID
		actorID := params.Actor.UserID
		_, created, err := r.points.AppendTx(ctx, tx, points.AppendInput{
			ContestID: params.ContestID, EventParticipantID: participantID,
			Amount: lecture.Points, Type: points.TypeLectureAttendance,
			SourceType: &sourceType, SourceID: &sourceID,
			Description:     fmt.Sprintf("Посещение лекции «%s»", lecture.Title),
			CreatedByUserID: &actorID, IdempotencyKey: "lecture-attendance:" + attendance.ID,
		})
		if err != nil {
			return err
		}
		if !created {
			return points.ErrIdempotencyConflict
		}
		if err := r.audit.LogEntryTx(ctx, tx, audit.Entry{
			ActorUserID: params.Actor.UserID, ContestID: params.ContestID,
			Action: "LECTURE_ATTENDANCE_RECORDED", EntityType: "lecture_attendance",
			EntityID: attendance.ID, Metadata: map[string]any{
				"lecture_id": params.LectureID, "participant_id": participantID,
				"points": lecture.Points, "scanner_type": params.ScannerType,
			},
		}); err != nil {
			return err
		}
		result = &ScanResult{Attendance: *attendance, AlreadyChecked: false}
		return nil
	})
	return result, err
}

func (r *Repo) ListAttendance(ctx context.Context, contestID, lectureID string) ([]Attendance, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM lectures
		WHERE contest_id=$1 AND id=$2)`, contestID, lectureID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	rows, err := r.pool.Query(ctx, `
		SELECT a.id, a.contest_id, a.lecture_id, a.event_participant_id, p.full_name,
		       a.scanned_by_user_id, a.scanner_type, a.points_awarded, a.created_at
		FROM lecture_attendance a
		JOIN event_participants p ON p.id=a.event_participant_id
		WHERE a.contest_id=$1 AND a.lecture_id=$2 ORDER BY a.created_at DESC`,
		contestID, lectureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Attendance, 0)
	for rows.Next() {
		attendance, err := scanAttendance(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *attendance)
	}
	return list, rows.Err()
}

func (r *Repo) ParticipantLectures(ctx context.Context, contestID, participantID string) ([]ParticipantLecture, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.id, l.contest_id, l.title, l.description, l.points, l.starts_at, l.ends_at,
		       l.attendance_starts_at, l.attendance_ends_at, l.status, l.created_at, l.updated_at,
		       a.id, a.scanned_by_user_id, a.scanner_type, a.points_awarded, a.created_at
		FROM lectures l
		LEFT JOIN lecture_attendance a
		  ON a.lecture_id=l.id AND a.event_participant_id=$2
		WHERE l.contest_id=$1 AND l.status IN ('ACTIVE','FINISHED')
		  AND (
		    NOT EXISTS (SELECT 1 FROM lecture_directions ld WHERE ld.lecture_id=l.id)
		    OR EXISTS (
		      SELECT 1 FROM lecture_directions ld
		      JOIN event_participants p ON p.id=$2 AND p.contest_id=$1
		       AND p.direction_id=ld.direction_id
		      WHERE ld.lecture_id=l.id
		    )
		  )
		ORDER BY l.starts_at DESC NULLS LAST, l.created_at DESC`, contestID, participantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]ParticipantLecture, 0)
	for rows.Next() {
		var item ParticipantLecture
		var attendanceID, scannedBy, scannerType *string
		var pointsAwarded *int64
		var attendanceAt *time.Time
		if err := rows.Scan(&item.Lecture.ID, &item.Lecture.ContestID, &item.Lecture.Title,
			&item.Lecture.Description, &item.Lecture.Points, &item.Lecture.StartsAt,
			&item.Lecture.EndsAt, &item.Lecture.AttendanceStartsAt,
			&item.Lecture.AttendanceEndsAt, &item.Lecture.Status,
			&item.Lecture.CreatedAt, &item.Lecture.UpdatedAt, &attendanceID,
			&scannedBy, &scannerType, &pointsAwarded, &attendanceAt); err != nil {
			return nil, err
		}
		if attendanceID != nil {
			item.Attendance = &Attendance{
				ID: *attendanceID, ContestID: contestID, LectureID: item.Lecture.ID,
				EventParticipantID: participantID, ScannedByUserID: *scannedBy,
				ScannerType: *scannerType, PointsAwarded: *pointsAwarded, CreatedAt: *attendanceAt,
			}
		}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	lectures := make([]Lecture, len(list))
	for i := range list {
		lectures[i] = list[i].Lecture
	}
	lectures, err = r.withDirections(ctx, r.pool, lectures)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Lecture = lectures[i]
	}
	return list, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanLecture(row rowScanner) (*Lecture, error) {
	var lecture Lecture
	err := row.Scan(&lecture.ID, &lecture.ContestID, &lecture.Title, &lecture.Description,
		&lecture.Points, &lecture.StartsAt, &lecture.EndsAt, &lecture.AttendanceStartsAt,
		&lecture.AttendanceEndsAt, &lecture.Status, &lecture.CreatedAt, &lecture.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	lecture.DirectionIDs = []string{}
	lecture.Directions = []DirectionRef{}
	return &lecture, nil
}

func scanAttendance(row rowScanner) (*Attendance, error) {
	var attendance Attendance
	err := row.Scan(&attendance.ID, &attendance.ContestID, &attendance.LectureID,
		&attendance.EventParticipantID, &attendance.ParticipantName,
		&attendance.ScannedByUserID, &attendance.ScannerType,
		&attendance.PointsAwarded, &attendance.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &attendance, nil
}

func attendanceByParticipant(ctx context.Context, query interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, lectureID, participantID string) (*Attendance, error) {
	return scanAttendance(query.QueryRow(ctx, `
		SELECT a.id, a.contest_id, a.lecture_id, a.event_participant_id, p.full_name,
		       a.scanned_by_user_id, a.scanner_type, a.points_awarded, a.created_at
		FROM lecture_attendance a
		JOIN event_participants p ON p.id=a.event_participant_id
		WHERE a.lecture_id=$1 AND a.event_participant_id=$2`, lectureID, participantID))
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type directionQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func replaceLectureDirections(ctx context.Context, q directionQuerier, contestID, lectureID string, ids []string) error {
	if _, err := q.Exec(ctx, `DELETE FROM lecture_directions WHERE lecture_id=$1`, lectureID); err != nil {
		return err
	}
	for _, id := range ids {
		tag, err := q.Exec(ctx, `
			INSERT INTO lecture_directions (contest_id, lecture_id, direction_id)
			SELECT $1, $2, d.id FROM event_directions d
			WHERE d.contest_id=$1 AND d.id=$3`, contestID, lectureID, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrValidation
		}
	}
	return nil
}

func lectureDirectionIDs(ctx context.Context, q directionQuerier, lectureID string) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT direction_id FROM lecture_directions WHERE lecture_id=$1`, lectureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repo) withDirections(ctx context.Context, q directionQuerier, lectures []Lecture) ([]Lecture, error) {
	for i := range lectures {
		lectures[i].DirectionIDs = []string{}
		lectures[i].Directions = []DirectionRef{}
	}
	if len(lectures) == 0 {
		return lectures, nil
	}
	ids := make([]string, len(lectures))
	index := make(map[string]int, len(lectures))
	for i := range lectures {
		ids[i] = lectures[i].ID
		index[lectures[i].ID] = i
	}
	rows, err := q.Query(ctx, `
		SELECT ld.lecture_id, d.id, d.name
		FROM lecture_directions ld
		JOIN event_directions d ON d.id=ld.direction_id AND d.contest_id=ld.contest_id
		WHERE ld.lecture_id=ANY($1::uuid[])
		ORDER BY d.sort_order, d.name, d.id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var lectureID, directionID, name string
		if err := rows.Scan(&lectureID, &directionID, &name); err != nil {
			return nil, err
		}
		i, ok := index[lectureID]
		if !ok {
			continue
		}
		lectures[i].DirectionIDs = append(lectures[i].DirectionIDs, directionID)
		lectures[i].Directions = append(lectures[i].Directions, DirectionRef{ID: directionID, Name: name})
	}
	return lectures, rows.Err()
}
