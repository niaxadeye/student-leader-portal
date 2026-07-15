// Command admin — служебный CLI: миграции и bootstrap суперадмина.
//   admin migrate            — применить миграции
//   admin create-superadmin  — создать/обновить суперадмина из env
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eazytech/student-leader-cabinet/internal/config"
	"github.com/eazytech/student-leader-cabinet/internal/platform/db"
	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: admin <migrate|create-superadmin|create-user>")
		os.Exit(2)
	}
	cfg, err := config.Load()
	must(err)
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.Postgres.DSN())
	must(err)
	defer pool.Close()

	switch os.Args[1] {
	case "migrate":
		applied, err := db.Migrate(ctx, pool)
		must(err)
		if len(applied) == 0 {
			fmt.Println("no new migrations")
		} else {
			fmt.Printf("applied: %v\n", applied)
		}
	case "create-superadmin":
		must(createSuperadmin(ctx, pool))
		fmt.Println("superadmin ready")
	case "create-user":
		must(createUser(ctx, pool, os.Args[2:]))
	default:
		fmt.Println("unknown command:", os.Args[1])
		os.Exit(2)
	}
}

// createSuperadmin идемпотентно создаёт/обновляет суперадмина из env
// (BOOTSTRAP_SUPERADMIN_LOGIN / _PASSWORD). Логин уникален — при повторе обновляет пароль.
func createSuperadmin(ctx context.Context, pool *pgxpool.Pool) error {
	login := os.Getenv("BOOTSTRAP_SUPERADMIN_LOGIN")
	password := os.Getenv("BOOTSTRAP_SUPERADMIN_PASSWORD")
	if login == "" || password == "" {
		return fmt.Errorf("set BOOTSTRAP_SUPERADMIN_LOGIN and BOOTSTRAP_SUPERADMIN_PASSWORD")
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
		VALUES ($1, $2, 'Суперадминистратор', 'ACTIVE', FALSE)
		ON CONFLICT (login) DO UPDATE
		SET password_hash = EXCLUDED.password_hash, status = 'ACTIVE',
		    must_change_password = FALSE, updated_at = now()
		RETURNING id`, login, hash).Scan(&userID)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
		SELECT $1, r.id, 'GLOBAL', '00000000-0000-0000-0000-000000000000'
		FROM roles r WHERE r.code = 'SUPER_ADMIN'
		ON CONFLICT DO NOTHING`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
