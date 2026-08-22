package contests

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// Participants возвращает активных участников конкурса с данными пользователя.
func (r *Repo) Participants(ctx context.Context, contestID string) ([]Participant, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.id, p.contest_id, p.user_id, p.participant_type,
		       u.login, u.full_name, u.organization, u.status, u.avatar_key, p.joined_at, p.left_at
		FROM contest_participants p JOIN users u ON u.id = p.user_id
		WHERE p.contest_id = $1 AND p.left_at IS NULL
		ORDER BY p.joined_at`, contestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]Participant, 0)
	for rows.Next() {
		var p Participant
		if err := rows.Scan(&p.ID, &p.ContestID, &p.UserID, &p.ParticipantType,
			&p.Login, &p.FullName, &p.Organization, &p.UserStatus, &p.AvatarKey,
			&p.JoinedAt, &p.LeftAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// NewContestant — данные для создания конкурсанта.
type NewContestant struct {
	Login        string
	FullName     string
	Organization string
	PasswordHash string
}

// LookupLogin возвращает существующий аккаунт по логину (без пароля).
func (r *Repo) LookupLogin(ctx context.Context, login string) (*existingAccount, error) {
	var acc existingAccount
	err := r.pool.QueryRow(ctx, `
		SELECT id, created_by FROM users WHERE login=$1 AND deleted_at IS NULL`, login).
		Scan(&acc.ID, &acc.CreatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT rl.code FROM user_roles ur JOIN roles rl ON rl.id=ur.role_id
		WHERE ur.user_id=$1`, acc.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	acc.Roles = []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		acc.Roles = append(acc.Roles, code)
	}
	return &acc, rows.Err()
}

// InsertContestantUser создаёт нового конкурсанта с created_by актора.
func (r *Repo) InsertContestantUser(ctx context.Context, createdBy string, nc NewContestant) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (login, password_hash, full_name, organization, status, must_change_password, created_by)
		VALUES ($1,$2,$3,$4,'ACTIVE',TRUE,$5)
		RETURNING id`, nc.Login, nc.PasswordHash, nc.FullName, nc.Organization, createdBy).Scan(&id)
	if isUniqueViolation(err) {
		return "", ErrLoginConflict
	}
	return id, err
}

// AttachContestant назначает роль CONTESTANT на конкурс и строку участия.
// Не меняет профиль и пароль существующего пользователя.
func (r *Repo) AttachContestant(ctx context.Context, contestID, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
		SELECT $1, r.id, 'CONTEST', $2 FROM roles r WHERE r.code='CONTESTANT'
		ON CONFLICT DO NOTHING`, userID, contestID); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO contest_participants (contest_id, user_id, participant_type)
		VALUES ($1,$2,'CONTESTANT')
		ON CONFLICT (contest_id, user_id) DO UPDATE SET left_at=NULL`,
		contestID, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RemoveParticipant помечает участие завершённым (soft, left_at).
func (r *Repo) RemoveParticipant(ctx context.Context, contestID, userID string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE contest_participants SET left_at=now()
		WHERE contest_id=$1 AND user_id=$2 AND left_at IS NULL`, contestID, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// LoginExists проверяет занятость логина (для дружелюбной ошибки до вставки).
func (r *Repo) LoginExists(ctx context.Context, login string) (bool, error) {
	var id string
	err := r.pool.QueryRow(ctx, `SELECT id FROM users WHERE login=$1 AND deleted_at IS NULL`, login).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *Repo) IsActiveContestant(ctx context.Context, contestID, userID string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM contest_participants
			WHERE contest_id=$1 AND user_id=$2 AND left_at IS NULL AND participant_type='CONTESTANT'
		)`, contestID, userID).Scan(&ok)
	return ok, err
}

func (r *Repo) SetUserAvatarKey(ctx context.Context, userID string, key *string) (*string, error) {
	var prev *string
	err := r.pool.QueryRow(ctx, `
		WITH old AS (SELECT avatar_key FROM users WHERE id=$1 AND deleted_at IS NULL)
		UPDATE users SET avatar_key=$2, updated_at=now()
		WHERE id=$1 AND deleted_at IS NULL
		RETURNING (SELECT avatar_key FROM old)`, userID, key).Scan(&prev)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return prev, nil
}
