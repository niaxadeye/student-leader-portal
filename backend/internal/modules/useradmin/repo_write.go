package useradmin

import "context"

// NewUser — данные для создания пользователя.
type NewUser struct {
	Login        string
	FullName     string
	Email        *string
	Organization *string
	PasswordHash string
}

// Create вставляет пользователя (must_change=TRUE) и, если задан role, назначает его.
func (r *Repo) Create(ctx context.Context, nu NewUser, role, scopeType, scopeID string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (login, password_hash, full_name, email, organization, status, must_change_password)
		VALUES ($1,$2,$3,$4,$5,'ACTIVE',TRUE) RETURNING id`,
		nu.Login, nu.PasswordHash, nu.FullName, nu.Email, nu.Organization).Scan(&id)
	if isUniqueViolation(err) {
		return "", ErrLoginTaken
	}
	if err != nil {
		return "", err
	}
	if role != "" {
		if _, err = tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
			SELECT $1, rl.id, $2, $3 FROM roles rl WHERE rl.code=$4
			ON CONFLICT DO NOTHING`, id, scopeType, scopeID, role); err != nil {
			return "", err
		}
	}
	return id, tx.Commit(ctx)
}

// UpdateProfile меняет редактируемые поля профиля.
func (r *Repo) UpdateProfile(ctx context.Context, id, fullName string, email, org *string) error {
	ct, err := r.pool.Exec(ctx, `
		UPDATE users SET full_name=$2, email=$3, organization=$4, updated_at=now()
		WHERE id=$1 AND deleted_at IS NULL`, id, fullName, email, org)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// AssignRole идемпотентно назначает роль со scope. Возвращает ErrRoleNotFound для несуществующего кода.
func (r *Repo) AssignRole(ctx context.Context, userID, role, scopeType, scopeID string) error {
	ct, err := r.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
		SELECT $1, rl.id, $2, $3 FROM roles rl WHERE rl.code=$4
		ON CONFLICT DO NOTHING`, userID, scopeType, scopeID, role)
	if err != nil {
		return err
	}
	// 0 строк = роль-код не найден (при повторном назначении вернётся тоже 0 —
	// но это идемпотентно и безопасно; несуществующий код ловим отдельной проверкой в сервисе).
	if ct.RowsAffected() == 0 {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE code=$1)`, role).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrRoleNotFound
		}
	}
	return nil
}

// RemoveRole снимает роль пользователя со scope.
func (r *Repo) RemoveRole(ctx context.Context, userID, role, scopeType, scopeID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM user_roles ur USING roles rl
		WHERE ur.role_id=rl.id AND ur.user_id=$1 AND rl.code=$2
		  AND ur.scope_type=$3 AND ur.scope_id=$4`, userID, role, scopeType, scopeID)
	return err
}
