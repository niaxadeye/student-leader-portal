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
		       u.login, u.full_name, u.organization, u.status, p.joined_at, p.left_at
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
			&p.Login, &p.FullName, &p.Organization, &p.UserStatus,
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

// AddContestant в одной транзакции: создаёт пользователя (must_change),
// назначает роль CONTESTANT scope=CONTEST и добавляет participant-строку.
// Возвращает userID. Идемпотентен по login (повторный — обновит роль/связь).
func (r *Repo) AddContestant(ctx context.Context, contestID string, nc NewContestant) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (login, password_hash, full_name, organization, status, must_change_password)
		VALUES ($1,$2,$3,$4,'ACTIVE',TRUE)
		ON CONFLICT (login) DO UPDATE SET full_name=EXCLUDED.full_name,
		    organization=EXCLUDED.organization, updated_at=now()
		RETURNING id`, nc.Login, nc.PasswordHash, nc.FullName, nc.Organization).Scan(&userID)
	if err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
		SELECT $1, r.id, 'CONTEST', $2 FROM roles r WHERE r.code='CONTESTANT'
		ON CONFLICT DO NOTHING`, userID, contestID); err != nil {
		return "", err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO contest_participants (contest_id, user_id, participant_type)
		VALUES ($1,$2,'CONTESTANT')
		ON CONFLICT (contest_id, user_id) DO UPDATE SET left_at=NULL`,
		contestID, userID); err != nil {
		return "", err
	}
	return userID, tx.Commit(ctx)
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
