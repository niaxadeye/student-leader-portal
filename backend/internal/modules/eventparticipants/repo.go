package eventparticipants

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/modules/eventpermissions"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// CanManage проверяет явный scoped permission или владение конкурсом.
// MEGA_ADMIN обрабатывается сервисом до обращения к репозиторию.
func (r *Repo) CanManage(ctx context.Context, userID, contestID string) (bool, error) {
	var allowed bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM contests c
			WHERE c.id=$2 AND (
				c.owner_user_id=$1 OR EXISTS (
					SELECT 1 FROM event_staff_permissions ep
					WHERE ep.user_id=$1 AND ep.contest_id=c.id
					  AND ep.permission=ANY($3::varchar[]))))`,
		userID, contestID, eventpermissions.GrantsFor(PermissionManageParticipants)).Scan(&allowed)
	return allowed, err
}

func (r *Repo) List(ctx context.Context, contestID string, f ListFilter) ([]Participant, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contest_id, full_name, full_name_normalized, birth_date,
		       union_card_number, sks_barcode, status, created_at, updated_at,
		       archived_at, count(*) OVER()
		FROM event_participants
		WHERE contest_id=$1
		  AND ($2='' OR full_name ILIKE '%'||$2||'%'
		       OR union_card_number::text ILIKE '%'||$2||'%'
		       OR sks_barcode::text ILIKE '%'||$2||'%')
		  AND ($3='' OR status=$3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5`, contestID, f.Search, f.Status, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]Participant, 0)
	total := 0
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ID, &p.ContestID, &p.FullName, &p.FullNameNormalized,
			&p.BirthDate, &p.UnionCardNumber, &p.SKSBarcode, &p.Status,
			&p.CreatedAt, &p.UpdatedAt, &p.ArchivedAt, &total); err != nil {
			return nil, 0, err
		}
		list = append(list, p)
	}
	return list, total, rows.Err()
}

func (r *Repo) All(ctx context.Context, contestID string) ([]Participant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contest_id, full_name, full_name_normalized, birth_date,
		       union_card_number, sks_barcode, status, created_at, updated_at, archived_at
		FROM event_participants WHERE contest_id=$1 ORDER BY full_name, birth_date, id`, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Participant, 0)
	for rows.Next() {
		participant, err := scanOneParticipant(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *participant)
	}
	return list, rows.Err()
}

func (r *Repo) ByID(ctx context.Context, contestID, participantID string) (*Participant, error) {
	return scanOneParticipant(r.pool.QueryRow(ctx, `
		SELECT id, contest_id, full_name, full_name_normalized, birth_date,
		       union_card_number, sks_barcode, status, created_at, updated_at, archived_at
		FROM event_participants WHERE contest_id=$1 AND id=$2`, contestID, participantID))
}

func (r *Repo) Create(ctx context.Context, p *Participant) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO event_participants
		  (contest_id, full_name, full_name_normalized, birth_date,
		   union_card_number, sks_barcode, status)
		VALUES ($1,$2,$3,$4,$5,$6,'ACTIVE') RETURNING id`,
		p.ContestID, p.FullName, p.FullNameNormalized, p.BirthDate,
		p.UnionCardNumber, p.SKSBarcode).Scan(&id)
	if isUniqueViolation(err) {
		return "", ErrIdentifierTaken
	}
	return id, err
}

func (r *Repo) Update(ctx context.Context, p *Participant) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE event_participants SET
		  full_name=$3, full_name_normalized=$4, birth_date=$5,
		  union_card_number=$6, sks_barcode=$7, updated_at=now()
		WHERE contest_id=$1 AND id=$2`,
		p.ContestID, p.ID, p.FullName, p.FullNameNormalized, p.BirthDate,
		p.UnionCardNumber, p.SKSBarcode)
	if isUniqueViolation(err) {
		return ErrIdentifierTaken
	}
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetStatus меняет статус и отзывает все сессии при BLOCKED/ARCHIVED.
func (r *Repo) SetStatus(ctx context.Context, contestID, participantID, status string) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `
			UPDATE event_participants SET status=$3::varchar(16), updated_at=now(),
			  archived_at=CASE WHEN $3::varchar(16)='ARCHIVED'
			    THEN COALESCE(archived_at, now()) ELSE archived_at END
			WHERE contest_id=$1 AND id=$2`, contestID, participantID, status)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return ErrNotFound
		}
		if status != StatusActive {
			_, err = tx.Exec(ctx, `
				UPDATE participant_sessions SET revoked_at=COALESCE(revoked_at, now()),
				  revoke_reason=$3
				WHERE contest_id=$1 AND event_participant_id=$2 AND revoked_at IS NULL`,
				contestID, participantID, "participant_"+status)
		}
		return err
	})
}

func (r *Repo) EventBySlug(ctx context.Context, slug string) (*EventRef, error) {
	var event EventRef
	err := r.pool.QueryRow(ctx, `
		SELECT id, slug, name, status, timezone FROM contests WHERE slug=$1`, slug).
		Scan(&event.ID, &event.Slug, &event.Name, &event.Status, &event.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrEventUnavailable
	}
	return &event, err
}

func (r *Repo) FindByNameBirth(ctx context.Context, contestID, normalizedName string, birthDate time.Time) ([]Participant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, contest_id, full_name, full_name_normalized, birth_date,
		       union_card_number, sks_barcode, status, created_at, updated_at, archived_at
		FROM event_participants
		WHERE contest_id=$1 AND full_name_normalized=$2 AND birth_date=$3`,
		contestID, normalizedName, birthDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Participant, 0)
	for rows.Next() {
		p, err := scanOneParticipant(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

func (r *Repo) FindByUnionCard(ctx context.Context, contestID, number string) (*Participant, error) {
	return r.findByIdentifier(ctx, contestID, "union_card_number", number)
}

func (r *Repo) FindBySKSBarcode(ctx context.Context, contestID, barcode string) (*Participant, error) {
	return r.findByIdentifier(ctx, contestID, "sks_barcode", barcode)
}

func (r *Repo) findByIdentifier(ctx context.Context, contestID, column, value string) (*Participant, error) {
	// column выбирается только внутренними константными вызовами выше, пользовательские
	// данные передаются параметрами.
	query := `SELECT id, contest_id, full_name, full_name_normalized, birth_date,
	                 union_card_number, sks_barcode, status, created_at, updated_at, archived_at
	          FROM event_participants WHERE contest_id=$1 AND ` + column + `=$2`
	p, err := scanOneParticipant(r.pool.QueryRow(ctx, query, contestID, value))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	return p, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOneParticipant(row rowScanner) (*Participant, error) {
	var p Participant
	err := row.Scan(&p.ID, &p.ContestID, &p.FullName, &p.FullNameNormalized,
		&p.BirthDate, &p.UnionCardNumber, &p.SKSBarcode, &p.Status,
		&p.CreatedAt, &p.UpdatedAt, &p.ArchivedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
