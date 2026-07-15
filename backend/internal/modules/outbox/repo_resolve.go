package outbox

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErrGone — агрегат события больше не существует (например, работа удалена).
// Такое событие не имеет смысла ретраить — помечаем DEAD.
var ErrGone = errors.New("aggregate gone")

// SubmissionView — данные для шаблона Telegram-уведомления об отправке (SITE.md §15).
type SubmissionView struct {
	SubmissionID string
	ContestName  string
	Challenge    string
	FullName     string
	Organization *string
	Revision     int
	SubmittedAt  *string
	FileCount    int
}

// ResolveSubmission собирает человекочитаемые поля для уведомления одним запросом.
func (r *Repo) ResolveSubmission(ctx context.Context, submissionID string) (*SubmissionView, error) {
	var v SubmissionView
	v.SubmissionID = submissionID
	err := r.pool.QueryRow(ctx, `
		SELECT ct.name, ch.title, u.full_name, u.organization,
		       s.current_revision_number,
		       to_char(s.submitted_at, 'DD.MM.YYYY HH24:MI'),
		       (SELECT count(*) FROM submission_files sf WHERE sf.submission_id = s.id)
		FROM submissions s
		JOIN contest_challenges ch ON ch.id = s.challenge_id
		JOIN contests ct ON ct.id = ch.contest_id
		JOIN users u ON u.id = s.contestant_user_id
		WHERE s.id = $1`, submissionID).
		Scan(&v.ContestName, &v.Challenge, &v.FullName, &v.Organization,
			&v.Revision, &v.SubmittedAt, &v.FileCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGone
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}
