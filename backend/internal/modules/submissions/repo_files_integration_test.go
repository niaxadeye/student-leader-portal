//go:build integration

package submissions

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	fileIntegrationUserID      = "40000000-0000-4000-8000-000000000001"
	fileIntegrationContestID   = "40000000-0000-4000-8000-000000000002"
	fileIntegrationChallengeA  = "40000000-0000-4000-8000-000000000003"
	fileIntegrationChallengeB  = "40000000-0000-4000-8000-000000000004"
	fileIntegrationSubmissionA = "40000000-0000-4000-8000-000000000005"
	fileIntegrationSubmissionB = "40000000-0000-4000-8000-000000000006"
	fileIntegrationFileA       = "40000000-0000-4000-8000-000000000007"
	fileIntegrationFileB       = "40000000-0000-4000-8000-000000000008"
)

func TestSubmissionFileBindingIntegration(t *testing.T) {
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
	seedFileIntegration(t, ctx, pool)

	repo := NewRepo(pool)
	ownerID, objectKey, err := repo.FileForSubmission(ctx, fileIntegrationSubmissionA, fileIntegrationFileA)
	if err != nil || ownerID != fileIntegrationUserID || objectKey != "integration/file-a.pdf" {
		t.Fatalf("bound file owner=%q key=%q err=%v", ownerID, objectKey, err)
	}

	if _, _, err := repo.FileForSubmission(ctx, fileIntegrationSubmissionA, fileIntegrationFileB); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("cross-submission lookup err=%v, want pgx.ErrNoRows", err)
	}

	service := NewService(repo, nil, nil)
	service.SetPresigner(func(_ context.Context, objectKey string) (string, error) {
		return "signed:" + objectKey, nil
	})
	if _, err := service.PresignFile(
		ctx,
		Actor{UserID: fileIntegrationUserID},
		fileIntegrationSubmissionA,
		fileIntegrationFileB,
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-submission presign err=%v, want ErrNotFound", err)
	}

	if err := repo.SoftDeleteFile(ctx, fileIntegrationSubmissionA, fileIntegrationFileB); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-submission delete err=%v, want ErrNotFound", err)
	}
	assertFileActive(t, ctx, pool, fileIntegrationFileB, true)

	if err := repo.SoftDeleteFile(ctx, fileIntegrationSubmissionA, fileIntegrationFileA); err != nil {
		t.Fatal(err)
	}
	assertFileActive(t, ctx, pool, fileIntegrationFileA, false)
}

func seedFileIntegration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO users (id, login, password_hash, full_name, must_change_password)
			VALUES ($1, 'file-integration', 'unused', 'File Integration', false)`, []any{fileIntegrationUserID}},
		{`INSERT INTO contests (id, name, slug, status, created_by)
			VALUES ($1, 'File Integration Contest', 'file-integration-contest', 'ACTIVE', $2)`, []any{fileIntegrationContestID, fileIntegrationUserID}},
		{`INSERT INTO contest_challenges (id, contest_id, title, slug, status) VALUES
			($1, $3, 'File Challenge A', 'file-challenge-a', 'PUBLISHED'),
			($2, $3, 'File Challenge B', 'file-challenge-b', 'PUBLISHED')`, []any{fileIntegrationChallengeA, fileIntegrationChallengeB, fileIntegrationContestID}},
		{`INSERT INTO submissions (id, challenge_id, contestant_user_id, status, schema_version) VALUES
			($1, $3, $4, 'DRAFT', 1),
			($2, $5, $4, 'DRAFT', 1)`, []any{
			fileIntegrationSubmissionA, fileIntegrationSubmissionB, fileIntegrationChallengeA,
			fileIntegrationUserID, fileIntegrationChallengeB,
		}},
		{`INSERT INTO files (
			id, owner_user_id, contest_id, challenge_id, submission_id, bucket,
			object_key, original_name, safe_name, status, uploaded_at
		) VALUES
			($1, $3, $4, $5, $6, '', 'integration/file-a.pdf', 'a.pdf', 'a.pdf', 'READY', now()),
			($2, $3, $4, $7, $8, '', 'integration/file-b.pdf', 'b.pdf', 'b.pdf', 'READY', now())`, []any{
			fileIntegrationFileA, fileIntegrationFileB, fileIntegrationUserID, fileIntegrationContestID,
			fileIntegrationChallengeA, fileIntegrationSubmissionA, fileIntegrationChallengeB, fileIntegrationSubmissionB,
		}},
		{`INSERT INTO submission_files (submission_id, file_id) VALUES ($1, $2), ($3, $4)`, []any{
			fileIntegrationSubmissionA, fileIntegrationFileA, fileIntegrationSubmissionB, fileIntegrationFileB,
		}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
}

func assertFileActive(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fileID string, want bool) {
	t.Helper()
	var active bool
	if err := pool.QueryRow(ctx, `SELECT deleted_at IS NULL FROM files WHERE id=$1`, fileID).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if active != want {
		t.Fatalf("file %s active=%t want=%t", fileID, active, want)
	}
}
