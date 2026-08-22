//go:build integration

package auth

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const refreshIntegrationUserID = "30000000-0000-4000-8000-000000000001"

func TestRotateRefreshAtomicallyIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, login, password_hash, full_name, must_change_password)
		VALUES ($1, 'refresh-integration', 'unused', 'Refresh Integration', false)`, refreshIntegrationUserID); err != nil {
		t.Fatal(err)
	}

	repo := NewRepo(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	session := &Session{UserID: refreshIntegrationUserID, ExpiresAt: now.Add(time.Hour)}
	familyID := "30000000-0000-4000-8000-000000000002"
	oldHash := "old-refresh-token-hash"
	if err := repo.CreateSession(
		ctx,
		session,
		familyID,
		"30000000-0000-4000-8000-000000000003",
		oldHash,
		now.Add(time.Hour),
	); err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		row *refreshRow
		err error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, 2)
	var wg sync.WaitGroup
	for i, next := range []struct {
		jti  string
		hash string
	}{
		{"30000000-0000-4000-8000-000000000004", "new-refresh-token-hash-1"},
		{"30000000-0000-4000-8000-000000000005", "new-refresh-token-hash-2"},
	} {
		wg.Add(1)
		go func(i int, next struct{ jti, hash string }) {
			defer wg.Done()
			<-start
			row, err := repo.RotateRefreshAtomically(
				ctx, oldHash, next.jti, next.hash, now.Add(time.Hour), now.Add(time.Duration(i)*time.Millisecond),
			)
			outcomes <- outcome{row: row, err: err}
		}(i, next)
	}
	close(start)
	wg.Wait()
	close(outcomes)

	successes, reused := 0, 0
	for outcome := range outcomes {
		switch {
		case outcome.err == nil && outcome.row != nil:
			successes++
		case errors.Is(outcome.err, ErrRefreshReused) && outcome.row != nil:
			reused++
		default:
			t.Fatalf("unexpected refresh outcome: row=%+v err=%v", outcome.row, outcome.err)
		}
	}
	if successes != 1 || reused != 1 {
		t.Fatalf("successes=%d reused=%d", successes, reused)
	}

	var children int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM refresh_tokens
		WHERE rotated_from_id = (
			SELECT id FROM refresh_tokens WHERE token_hash = $1
		)`, oldHash).Scan(&children); err != nil {
		t.Fatal(err)
	}
	if children != 1 {
		t.Fatalf("rotated children=%d want=1", children)
	}

	var revokedReason *string
	if err := pool.QueryRow(ctx, `
		SELECT revoke_reason FROM auth_sessions WHERE id = $1`, session.ID).Scan(&revokedReason); err != nil {
		t.Fatal(err)
	}
	if revokedReason == nil || *revokedReason != "refresh_reuse_detected" {
		t.Fatalf("revoke reason=%v", revokedReason)
	}
	var activeTokens int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM refresh_tokens WHERE session_id = $1 AND revoked_at IS NULL`, session.ID).Scan(&activeTokens); err != nil {
		t.Fatal(err)
	}
	if activeTokens != 0 {
		t.Fatalf("active family tokens=%d want=0", activeTokens)
	}
}
