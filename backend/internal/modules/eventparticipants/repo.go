package eventparticipants

import (
	"context"
	"errors"
	"strconv"
	"strings"
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

const participantSelect = `p.id, p.contest_id, p.full_name, p.full_name_normalized, p.birth_date,
		       p.union_card_number, p.sks_barcode, p.vk_url, p.telegram_url, p.vk_user_id, p.telegram_user_id,
		       p.status, p.created_at, p.updated_at, p.archived_at, p.direction_id, d.name`

const participantFrom = `event_participants p
		LEFT JOIN event_directions d ON d.id=p.direction_id AND d.contest_id=p.contest_id`

func (r *Repo) List(ctx context.Context, contestID string, f ListFilter) ([]Participant, int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+participantSelect+`, count(*) OVER()
		FROM `+participantFrom+`
		WHERE p.contest_id=$1
		  AND ($2='' OR p.full_name ILIKE '%'||$2||'%'
		       OR p.union_card_number::text ILIKE '%'||$2||'%'
		       OR p.sks_barcode::text ILIKE '%'||$2||'%'
		       OR p.vk_url ILIKE '%'||$2||'%'
		       OR p.telegram_url ILIKE '%'||$2||'%'
		       OR d.name ILIKE '%'||$2||'%')
		  AND ($3='' OR p.status=$3)
		  AND ($4='' OR p.direction_id::text=$4)
		ORDER BY p.created_at DESC
		LIMIT $5 OFFSET $6`, contestID, f.Search, f.Status, f.DirectionID, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]Participant, 0)
	total := 0
	for rows.Next() {
		var p Participant
		dest := append(participantScanDest(&p), &total)
		if err := rows.Scan(dest...); err != nil {
			return nil, 0, err
		}
		list = append(list, p)
	}
	return list, total, rows.Err()
}

func (r *Repo) All(ctx context.Context, contestID string) ([]Participant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+participantSelect+`
		FROM `+participantFrom+`
		WHERE p.contest_id=$1 ORDER BY p.full_name, p.birth_date, p.id`, contestID)
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
		SELECT `+participantSelect+`
		FROM `+participantFrom+`
		WHERE p.contest_id=$1 AND p.id=$2`, contestID, participantID))
}

func (r *Repo) Create(ctx context.Context, p *Participant) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO event_participants
		  (contest_id, full_name, full_name_normalized, birth_date,
		   union_card_number, sks_barcode, vk_url, telegram_url, status, direction_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'ACTIVE',$9) RETURNING id`,
		p.ContestID, p.FullName, p.FullNameNormalized, p.BirthDate,
		p.UnionCardNumber, p.SKSBarcode, p.VKURL, p.TelegramURL, p.DirectionID).Scan(&id)
	if isUniqueViolation(err) {
		return "", ErrIdentifierTaken
	}
	return id, err
}

func (r *Repo) Update(ctx context.Context, p *Participant) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE event_participants SET
		  full_name=$3, full_name_normalized=$4, birth_date=$5,
		  union_card_number=$6, sks_barcode=$7, vk_url=$8, telegram_url=$9,
		  direction_id=$10, updated_at=now()
		WHERE contest_id=$1 AND id=$2`,
		p.ContestID, p.ID, p.FullName, p.FullNameNormalized, p.BirthDate,
		p.UnionCardNumber, p.SKSBarcode, p.VKURL, p.TelegramURL, p.DirectionID)
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
		SELECT `+participantSelect+`
		FROM `+participantFrom+`
		WHERE p.contest_id=$1 AND p.full_name_normalized=$2 AND p.birth_date=$3`,
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

func (r *Repo) ListActiveEvents(ctx context.Context) ([]EventRef, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, slug, name, status, timezone
		FROM contests WHERE status='ACTIVE' ORDER BY name, slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]EventRef, 0)
	for rows.Next() {
		var event EventRef
		if err := rows.Scan(&event.ID, &event.Slug, &event.Name, &event.Status, &event.Timezone); err != nil {
			return nil, err
		}
		list = append(list, event)
	}
	return list, rows.Err()
}

func (r *Repo) ListActiveByTelegramUserID(ctx context.Context, userID int64) ([]ParticipantEventMatch, error) {
	return r.listActiveMatches(ctx, `
		SELECT `+participantSelect+`, `+eventSelect+`
		FROM `+activeMatchFrom+`
		WHERE `+activeMatchWhere+` AND p.telegram_user_id=$1
		ORDER BY c.name, c.slug`, userID)
}

// telegramProfileURL повторяет канонический вид, в котором ссылка лежит в базе.
func telegramProfileURL(username string) string {
	username = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
	if username == "" {
		return ""
	}
	return "https://t.me/" + username
}

func (r *Repo) ListActiveByTelegramUsername(ctx context.Context, username string) ([]ParticipantEventMatch, error) {
	profileURL := telegramProfileURL(username)
	if profileURL == "" {
		return nil, nil
	}
	return r.listActiveMatches(ctx, `
		SELECT `+participantSelect+`, `+eventSelect+`
		FROM `+activeMatchFrom+`
		WHERE `+activeMatchWhere+` AND p.telegram_url IS NOT NULL
		  AND lower(p.telegram_url) = $1
		ORDER BY c.name, c.slug`, profileURL)
}

func (r *Repo) ListActiveByVKUserID(ctx context.Context, userID int64) ([]ParticipantEventMatch, error) {
	return r.listActiveMatches(ctx, `
		SELECT `+participantSelect+`, `+eventSelect+`
		FROM `+activeMatchFrom+`
		WHERE `+activeMatchWhere+` AND p.vk_user_id=$1
		ORDER BY c.name, c.slug`, userID)
}

func (r *Repo) ListActiveByVKIdentity(ctx context.Context, userID int64, screenName string) ([]ParticipantEventMatch, error) {
	screenName = strings.ToLower(strings.TrimSpace(screenName))
	candidates := []string{"https://vk.com/id" + strconv.FormatInt(userID, 10), "https://vk.ru/id" + strconv.FormatInt(userID, 10)}
	if screenName != "" {
		candidates = append(candidates, "https://vk.com/"+screenName, "https://vk.ru/"+screenName)
	}
	return r.listActiveMatches(ctx, `
		SELECT `+participantSelect+`, `+eventSelect+`
		FROM `+activeMatchFrom+`
		WHERE `+activeMatchWhere+` AND p.vk_url IS NOT NULL AND lower(p.vk_url) = ANY($1::text[])
		ORDER BY c.name, c.slug`, candidates)
}

const eventSelect = `c.id, c.slug, c.name, c.status, c.timezone`

const activeMatchFrom = participantFrom + `
		JOIN contests c ON c.id=p.contest_id`

const activeMatchWhere = `p.status='ACTIVE' AND c.status='ACTIVE'`

func (r *Repo) listActiveMatches(ctx context.Context, query string, args ...any) ([]ParticipantEventMatch, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]ParticipantEventMatch, 0)
	for rows.Next() {
		var match ParticipantEventMatch
		dest := append(participantScanDest(&match.Participant),
			&match.Event.ID, &match.Event.Slug, &match.Event.Name, &match.Event.Status, &match.Event.Timezone)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		list = append(list, match)
	}
	return list, rows.Err()
}

func (r *Repo) FindByTelegramUserID(ctx context.Context, contestID string, userID int64) (*Participant, error) {
	return scanOneParticipant(r.pool.QueryRow(ctx, `
		SELECT `+participantSelect+` FROM `+participantFrom+`
		WHERE p.contest_id=$1 AND p.telegram_user_id=$2`, contestID, userID))
}

func (r *Repo) FindByVKUserID(ctx context.Context, contestID string, userID int64) (*Participant, error) {
	return scanOneParticipant(r.pool.QueryRow(ctx, `
		SELECT `+participantSelect+` FROM `+participantFrom+`
		WHERE p.contest_id=$1 AND p.vk_user_id=$2`, contestID, userID))
}

func (r *Repo) FindByTelegramUsername(ctx context.Context, contestID, username string) (*Participant, error) {
	profileURL := telegramProfileURL(username)
	if profileURL == "" {
		return nil, ErrNotFound
	}
	return scanOneParticipant(r.pool.QueryRow(ctx, `
		SELECT `+participantSelect+` FROM `+participantFrom+`
		WHERE p.contest_id=$1 AND p.telegram_url IS NOT NULL
		  AND lower(p.telegram_url) = $2`, contestID, profileURL))
}

func (r *Repo) FindByVKIdentity(ctx context.Context, contestID string, userID int64, screenName string) (*Participant, error) {
	screenName = strings.ToLower(strings.TrimSpace(screenName))
	candidates := []string{"https://vk.com/id" + strconv.FormatInt(userID, 10), "https://vk.ru/id" + strconv.FormatInt(userID, 10)}
	if screenName != "" {
		candidates = append(candidates, "https://vk.com/"+screenName, "https://vk.ru/"+screenName)
	}
	return scanOneParticipant(r.pool.QueryRow(ctx, `
		SELECT `+participantSelect+` FROM `+participantFrom+`
		WHERE p.contest_id=$1 AND p.vk_url IS NOT NULL AND lower(p.vk_url) = ANY($2::text[])`,
		contestID, candidates))
}

func (r *Repo) BindTelegram(ctx context.Context, contestID, participantID string, userID int64, profileURL *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE event_participants
		   SET telegram_user_id=$3,
		       telegram_url=COALESCE(telegram_url, $4),
		       updated_at=now()
		 WHERE contest_id=$1 AND id=$2`, contestID, participantID, userID, profileURL)
	return err
}

func (r *Repo) BindVK(ctx context.Context, contestID, participantID string, userID int64, profileURL *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE event_participants
		   SET vk_user_id=$3,
		       vk_url=COALESCE(vk_url, $4),
		       updated_at=now()
		 WHERE contest_id=$1 AND id=$2`, contestID, participantID, userID, profileURL)
	return err
}

func (r *Repo) findByIdentifier(ctx context.Context, contestID, column, value string) (*Participant, error) {
	// column выбирается только внутренними константными вызовами выше, пользовательские
	// данные передаются параметрами.
	query := `SELECT ` + participantSelect + `
	          FROM ` + participantFrom + `
	          WHERE p.contest_id=$1 AND p.` + column + `=$2`
	p, err := scanOneParticipant(r.pool.QueryRow(ctx, query, contestID, value))
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	return p, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func participantScanDest(p *Participant) []any {
	return []any{
		&p.ID, &p.ContestID, &p.FullName, &p.FullNameNormalized,
		&p.BirthDate, &p.UnionCardNumber, &p.SKSBarcode, &p.VKURL, &p.TelegramURL,
		&p.VKUserID, &p.TelegramUserID, &p.Status,
		&p.CreatedAt, &p.UpdatedAt, &p.ArchivedAt, &p.DirectionID, &p.DirectionName,
	}
}

func scanOneParticipant(row rowScanner) (*Participant, error) {
	var p Participant
	err := row.Scan(participantScanDest(&p)...)
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
