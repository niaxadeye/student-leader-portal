package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

var validRoles = map[string]bool{"SUPER_ADMIN": true, "ADMIN": true, "CONTESTANT": true}

// createUser: admin create-user <login> <password> <role> [full_name] [--must-change]
// Идемпотентна по login (повторный запуск обновляет пароль/роль). Роль — GLOBAL scope.
// Флаг --must-change ставит must_change_password=TRUE для проверки форс-смены пароля.
func createUser(ctx context.Context, pool *pgxpool.Pool, args []string) error {
	mustChange := false
	var pos []string
	for _, a := range args {
		if a == "--must-change" {
			mustChange = true
			continue
		}
		pos = append(pos, a)
	}
	if len(pos) < 3 {
		return fmt.Errorf("usage: admin create-user <login> <password> <role> [full_name] [--must-change]")
	}
	login, password, role := pos[0], pos[1], pos[2]
	fullName := role
	if len(pos) >= 4 {
		fullName = pos[3]
	}
	if !validRoles[role] {
		return fmt.Errorf("role must be one of SUPER_ADMIN|ADMIN|CONTESTANT")
	}
	if len(password) < 10 {
		return fmt.Errorf("password must be at least 10 characters")
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (login, password_hash, full_name, status, must_change_password)
		VALUES ($1, $2, $3, 'ACTIVE', $4)
		ON CONFLICT (login) DO UPDATE
		SET password_hash = EXCLUDED.password_hash, full_name = EXCLUDED.full_name,
		    status = 'ACTIVE', must_change_password = EXCLUDED.must_change_password,
		    failed_login_count = 0, locked_until = NULL, updated_at = now()
		RETURNING id`, login, hash, fullName, mustChange).Scan(&userID)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
		SELECT $1, r.id, 'GLOBAL', '00000000-0000-0000-0000-000000000000'
		FROM roles r WHERE r.code = $2
		ON CONFLICT DO NOTHING`, userID, role); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("user %q (%s) ready, must_change_password=%v\n", login, role, mustChange)
	return nil
}
