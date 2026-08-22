//go:build integration

package evaluation

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	integrationUserID       = "10000000-0000-4000-8000-000000000001"
	integrationContestantID = "10000000-0000-4000-8000-000000000002"
	integrationContestID    = "10000000-0000-4000-8000-000000000003"
	integrationChallengeID  = "10000000-0000-4000-8000-000000000004"
	integrationSchemeID     = "10000000-0000-4000-8000-000000000005"
	integrationCriterionID  = "10000000-0000-4000-8000-000000000006"
	integrationPerformance  = "10000000-0000-4000-8000-000000000007"
	integrationCriterionTwo = "10000000-0000-4000-8000-000000000008"
)

func TestApplyScoreMutationIntegration(t *testing.T) {
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
	seedScoreIntegration(t, ctx, pool)

	repo := NewRepo(pool)
	sheetID, err := repo.EnsureScoreSheet(ctx, integrationPerformance, integrationUserID)
	if err != nil {
		t.Fatal(err)
	}
	apply := func(mutationID string, score float64, base int) (*ScoreWriteResult, error) {
		return repo.ApplyScoreMutation(ctx, scoreMutationParams{
			ChallengeID: integrationChallengeID, PerformanceID: integrationPerformance,
			EvaluatorUserID: integrationUserID, ScoreSheetID: sheetID,
			CriterionID: integrationCriterionID, MutationID: mutationID,
			Score: score, BaseRevision: &base,
		})
	}

	firstID := "20000000-0000-4000-8000-000000000001"
	first, err := apply(firstID, 7, 0)
	if err != nil || first.Revision != 1 || first.Score != 7 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	retry, err := apply(firstID, 7, 0)
	if err != nil || retry.Revision != 1 || retry.Score != 7 {
		t.Fatalf("idempotent retry=%+v err=%v", retry, err)
	}
	assertHistoryCount(t, ctx, pool, 1)

	second, err := apply("20000000-0000-4000-8000-000000000002", 8, 1)
	if err != nil || second.Revision != 2 || second.Score != 8 {
		t.Fatalf("second=%+v err=%v", second, err)
	}

	staleID := "20000000-0000-4000-8000-000000000003"
	_, err = apply(staleID, 9, 1)
	var conflict *ScoreRevisionConflict
	if !errors.As(err, &conflict) || conflict.Revision != 2 || conflict.Score == nil || *conflict.Score != 8 {
		t.Fatalf("conflict=%+v err=%v", conflict, err)
	}
	rebased, err := apply(staleID, 9, conflict.Revision)
	if err != nil || rebased.Revision != 3 || rebased.Score != 9 {
		t.Fatalf("rebased=%+v err=%v", rebased, err)
	}

	start := make(chan struct{})
	type outcome struct {
		result *ScoreWriteResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	var wg sync.WaitGroup
	for i, score := range []float64{5, 6} {
		wg.Add(1)
		go func(i int, score float64) {
			defer wg.Done()
			<-start
			result, err := apply([]string{
				"20000000-0000-4000-8000-000000000004",
				"20000000-0000-4000-8000-000000000005",
			}[i], score, 3)
			outcomes <- outcome{result: result, err: err}
		}(i, score)
	}
	close(start)
	wg.Wait()
	close(outcomes)

	successes, conflicts := 0, 0
	for outcome := range outcomes {
		if outcome.err == nil && outcome.result.Revision == 4 {
			successes++
			continue
		}
		if errors.Is(outcome.err, ErrRevision) {
			conflicts++
			continue
		}
		t.Fatalf("unexpected concurrent outcome: result=%+v err=%v", outcome.result, outcome.err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	assertHistoryCount(t, ctx, pool, 4)

	// Different criteria do not revision-conflict, but their writes must still
	// serialize at sheet level so total_score_cache observes both commits.
	start = make(chan struct{})
	outcomes = make(chan outcome, 2)
	for _, mutation := range []scoreMutationParams{
		{
			ChallengeID: integrationChallengeID, PerformanceID: integrationPerformance,
			EvaluatorUserID: integrationUserID, ScoreSheetID: sheetID,
			CriterionID: integrationCriterionID, MutationID: "20000000-0000-4000-8000-000000000006",
			Score: 7, BaseRevision: intPtr(4),
		},
		{
			ChallengeID: integrationChallengeID, PerformanceID: integrationPerformance,
			EvaluatorUserID: integrationUserID, ScoreSheetID: sheetID,
			CriterionID: integrationCriterionTwo, MutationID: "20000000-0000-4000-8000-000000000007",
			Score: 3, BaseRevision: intPtr(0),
		},
	} {
		wg.Add(1)
		go func(mutation scoreMutationParams) {
			defer wg.Done()
			<-start
			result, err := repo.ApplyScoreMutation(ctx, mutation)
			outcomes <- outcome{result: result, err: err}
		}(mutation)
	}
	close(start)
	wg.Wait()
	close(outcomes)
	for outcome := range outcomes {
		if outcome.err != nil {
			t.Fatalf("different-criterion write: result=%+v err=%v", outcome.result, outcome.err)
		}
	}
	var total float64
	if err := pool.QueryRow(ctx, `SELECT total_score_cache FROM score_sheets WHERE id=$1`, sheetID).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 10 {
		t.Fatalf("concurrent sheet total=%v want=10", total)
	}
	assertHistoryCount(t, ctx, pool, 6)
}

func seedScoreIntegration(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO users (id, login, password_hash, full_name, must_change_password) VALUES
			($1, 'integration-jury', 'unused', 'Integration Jury', false),
			($2, 'integration-contestant', 'unused', 'Integration Contestant', false)`, []any{integrationUserID, integrationContestantID}},
		{`INSERT INTO contests (id, name, slug, status, created_by) VALUES
			($1, 'Integration Contest', 'integration-contest', 'ACTIVE', $2)`, []any{integrationContestID, integrationUserID}},
		{`INSERT INTO contest_challenges (id, contest_id, title, slug, status) VALUES
			($1, $2, 'Integration Challenge', 'integration-challenge', 'PUBLISHED')`, []any{integrationChallengeID, integrationContestID}},
		{`INSERT INTO evaluation_schemes (id, challenge_id, name, type, scoring_unit) VALUES
			($1, $2, 'Integration Scheme', 'CRITERIA_SCORING', 'POINTS')`, []any{integrationSchemeID, integrationChallengeID}},
		{`INSERT INTO evaluation_criteria (id, evaluation_scheme_id, title, min_score, max_score, weight) VALUES
			($1, $3, 'Criterion', 1, 10, 1),
			($2, $3, 'Criterion Two', 1, 10, 1)`, []any{integrationCriterionID, integrationCriterionTwo, integrationSchemeID}},
		{`INSERT INTO performances (id, challenge_id, contestant_user_id, status) VALUES
			($1, $2, $3, 'ACTIVE')`, []any{integrationPerformance, integrationChallengeID, integrationContestantID}},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
}

func intPtr(value int) *int { return &value }

func assertHistoryCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM score_value_history`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("history count=%d want=%d", got, want)
	}
}
