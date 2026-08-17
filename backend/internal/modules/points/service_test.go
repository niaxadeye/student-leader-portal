package points

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type fakeRepository struct {
	allowed       bool
	exists        bool
	balance       int64
	holds         int64
	entries       []Entry
	adjustments   map[string]*Entry
	createdWrites int
	lastActor     string
}

func (f *fakeRepository) CanManage(context.Context, string, string) (bool, error) {
	return f.allowed, nil
}
func (f *fakeRepository) ParticipantExists(context.Context, string, string) (bool, error) {
	return f.exists, nil
}
func (f *fakeRepository) LedgerBalance(context.Context, string, string) (int64, error) {
	return f.balance, nil
}
func (f *fakeRepository) ActiveHolds(context.Context, string, string) (int64, error) {
	return f.holds, nil
}
func (f *fakeRepository) List(context.Context, string, string, int, int) ([]Entry, int, error) {
	return f.entries, len(f.entries), nil
}
func (f *fakeRepository) Append(_ context.Context, input AppendInput) (*Entry, bool, error) {
	return f.append(input)
}
func (f *fakeRepository) AppendTx(
	_ context.Context,
	_ pgx.Tx,
	input AppendInput,
) (*Entry, bool, error) {
	return f.append(input)
}
func (f *fakeRepository) AppendAdminAdjustment(
	_ context.Context,
	actorUserID string,
	input AppendInput,
) (*Entry, bool, error) {
	f.lastActor = actorUserID
	return f.append(input)
}
func (f *fakeRepository) append(input AppendInput) (*Entry, bool, error) {
	if f.adjustments == nil {
		f.adjustments = make(map[string]*Entry)
	}
	if existing := f.adjustments[input.IdempotencyKey]; existing != nil {
		if !sameOperation(existing, input) {
			return nil, false, ErrIdempotencyConflict
		}
		return existing, false, nil
	}
	entry := &Entry{
		ID: "entry-1", ContestID: input.ContestID, EventParticipantID: input.EventParticipantID,
		Amount: input.Amount, Type: input.Type, SourceType: input.SourceType,
		SourceID: input.SourceID, Description: input.Description,
		CreatedByUserID: input.CreatedByUserID, IdempotencyKey: input.IdempotencyKey,
		CreatedAt: time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC),
	}
	f.adjustments[input.IdempotencyKey] = entry
	f.createdWrites++
	f.balance += input.Amount
	return entry, true, nil
}

func readyRepository() *fakeRepository {
	return &fakeRepository{allowed: true, exists: true}
}

func TestAdjustIsIdempotent(t *testing.T) {
	t.Parallel()
	repo := readyRepository()
	service := NewService(repo)
	actor := Actor{UserID: "staff-1"}
	input := AdjustmentInput{Amount: 100, Reason: "Компенсация участнику", IdempotencyKey: "adjustment-001"}

	first, err := service.Adjust(context.Background(), actor, "contest-1", "participant-1", input)
	if err != nil {
		t.Fatalf("first Adjust: %v", err)
	}
	second, err := service.Adjust(context.Background(), actor, "contest-1", "participant-1", input)
	if err != nil {
		t.Fatalf("second Adjust: %v", err)
	}
	if first.Replayed || !second.Replayed {
		t.Fatalf("replayed flags: first=%v second=%v", first.Replayed, second.Replayed)
	}
	if first.Entry.ID != second.Entry.ID || repo.createdWrites != 1 || repo.balance != 100 {
		t.Fatalf("idempotency failed: first=%q second=%q writes=%d balance=%d",
			first.Entry.ID, second.Entry.ID, repo.createdWrites, repo.balance)
	}
	if repo.lastActor != actor.UserID || first.Entry.CreatedByUserID == nil ||
		*first.Entry.CreatedByUserID != actor.UserID {
		t.Fatalf("actor was not preserved: repo=%q entry=%v", repo.lastActor, first.Entry.CreatedByUserID)
	}
}

func TestAdjustRejectsIdempotencyKeyReuseWithDifferentPayload(t *testing.T) {
	t.Parallel()
	repo := readyRepository()
	service := NewService(repo)
	actor := Actor{UserID: "staff-1"}
	key := "adjustment-conflict"
	if _, err := service.Adjust(context.Background(), actor, "contest-1", "participant-1",
		AdjustmentInput{Amount: 100, Reason: "Первая причина", IdempotencyKey: key}); err != nil {
		t.Fatalf("first Adjust: %v", err)
	}
	_, err := service.Adjust(context.Background(), actor, "contest-1", "participant-1",
		AdjustmentInput{Amount: 200, Reason: "Другая причина", IdempotencyKey: key})
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
	if repo.createdWrites != 1 || repo.balance != 100 {
		t.Fatalf("conflicting retry changed ledger: writes=%d balance=%d", repo.createdWrites, repo.balance)
	}
}

func TestAdjustValidatesPermissionAmountReasonAndParticipant(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		repo  *fakeRepository
		input AdjustmentInput
		want  error
	}{
		{
			name: "forbidden", repo: &fakeRepository{exists: true},
			input: AdjustmentInput{Amount: 1, Reason: "Причина", IdempotencyKey: "valid-key"},
			want:  ErrForbidden,
		},
		{
			name: "zero amount", repo: readyRepository(),
			input: AdjustmentInput{Reason: "Причина", IdempotencyKey: "valid-key"},
			want:  ErrValidation,
		},
		{
			name: "short reason", repo: readyRepository(),
			input: AdjustmentInput{Amount: 1, Reason: "x", IdempotencyKey: "valid-key"},
			want:  ErrValidation,
		},
		{
			name: "participant missing", repo: &fakeRepository{allowed: true},
			input: AdjustmentInput{Amount: 1, Reason: "Причина", IdempotencyKey: "valid-key"},
			want:  ErrNotFound,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewService(test.repo).Adjust(context.Background(), Actor{UserID: "staff"},
				"contest-1", "participant-1", test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestOverviewCalculatesAvailableBalanceAndEmptyHistory(t *testing.T) {
	t.Parallel()
	repo := readyRepository()
	repo.balance = 450
	repo.holds = 125
	overview, err := NewService(repo).ParticipantOverview(
		context.Background(), "contest-1", "participant-1", 0, -1,
	)
	if err != nil {
		t.Fatalf("ParticipantOverview: %v", err)
	}
	if overview.Balance.LedgerBalance != 450 || overview.Balance.ReservedPoints != 125 ||
		overview.Balance.AvailablePoints != 325 {
		t.Fatalf("balance = %#v", overview.Balance)
	}
	if overview.Entries == nil || overview.Limit != defaultLimit || overview.Offset != 0 {
		t.Fatalf("overview defaults = %#v", overview)
	}
}

func TestValidateAppendEnforcesTypeSignAndSource(t *testing.T) {
	t.Parallel()
	sourceType, sourceID := "lecture", "source-1"
	base := AppendInput{
		ContestID: "contest-1", EventParticipantID: "participant-1",
		Amount: 100, Type: TypeLectureAttendance, SourceType: &sourceType, SourceID: &sourceID,
		Description: "Посещение лекции", IdempotencyKey: "lecture:source-1:participant-1",
	}
	if err := validateAppend(base); err != nil {
		t.Fatalf("valid append: %v", err)
	}
	base.Amount = -100
	if !errors.Is(validateAppend(base), ErrValidation) {
		t.Fatal("negative lecture reward must be rejected")
	}
	base.Amount = 100
	base.SourceID = nil
	if !errors.Is(validateAppend(base), ErrValidation) {
		t.Fatal("unscoped source must be rejected")
	}
}
