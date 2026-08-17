package points

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

type Repository interface {
	CanManage(ctx context.Context, userID, contestID string) (bool, error)
	ParticipantExists(ctx context.Context, contestID, participantID string) (bool, error)
	LedgerBalance(ctx context.Context, contestID, participantID string) (int64, error)
	ActiveHolds(ctx context.Context, contestID, participantID string) (int64, error)
	List(ctx context.Context, contestID, participantID string, limit, offset int) ([]Entry, int, error)
	Append(ctx context.Context, input AppendInput) (*Entry, bool, error)
	AppendTx(ctx context.Context, tx pgx.Tx, input AppendInput) (*Entry, bool, error)
	AppendAdminAdjustment(ctx context.Context, actorUserID string, input AppendInput) (*Entry, bool, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) ensureManage(ctx context.Context, actor Actor, contestID string) error {
	if actor.IsMega {
		return nil
	}
	allowed, err := s.repo.CanManage(ctx, actor.UserID, contestID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ensureParticipant(ctx context.Context, contestID, participantID string) error {
	exists, err := s.repo.ParticipantExists(ctx, contestID, participantID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Balance(ctx context.Context, contestID, participantID string) (Balance, error) {
	ledger, err := s.repo.LedgerBalance(ctx, contestID, participantID)
	if err != nil {
		return Balance{}, err
	}
	reserved, err := s.repo.ActiveHolds(ctx, contestID, participantID)
	if err != nil {
		return Balance{}, err
	}
	return Balance{
		LedgerBalance: ledger, ReservedPoints: reserved, AvailablePoints: ledger - reserved,
	}, nil
}

func (s *Service) overview(
	ctx context.Context,
	contestID, participantID string,
	limit, offset int,
) (*Overview, error) {
	if err := s.ensureParticipant(ctx, contestID, participantID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	balance, err := s.Balance(ctx, contestID, participantID)
	if err != nil {
		return nil, err
	}
	entries, total, err := s.repo.List(ctx, contestID, participantID, limit, offset)
	if err != nil {
		return nil, err
	}
	if entries == nil {
		entries = []Entry{}
	}
	return &Overview{
		Balance: balance, Entries: entries, Total: total, Limit: limit, Offset: offset,
	}, nil
}

func (s *Service) ParticipantOverview(
	ctx context.Context,
	contestID, participantID string,
	limit, offset int,
) (*Overview, error) {
	return s.overview(ctx, contestID, participantID, limit, offset)
}

func (s *Service) AdminOverview(
	ctx context.Context,
	actor Actor,
	contestID, participantID string,
	limit, offset int,
) (*Overview, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	return s.overview(ctx, contestID, participantID, limit, offset)
}

// Append — внутренний idempotent writer для attendance/tasks/merch этапов.
func (s *Service) Append(ctx context.Context, input AppendInput) (*Entry, bool, error) {
	input, err := s.prepareAppend(ctx, input)
	if err != nil {
		return nil, false, err
	}
	return s.repo.Append(ctx, input)
}

// AppendTx сохраняет те же validation/idempotency правила внутри транзакции
// вызывающего модуля.
func (s *Service) AppendTx(
	ctx context.Context,
	tx pgx.Tx,
	input AppendInput,
) (*Entry, bool, error) {
	input, err := s.prepareAppend(ctx, input)
	if err != nil {
		return nil, false, err
	}
	return s.repo.AppendTx(ctx, tx, input)
}

func (s *Service) prepareAppend(ctx context.Context, input AppendInput) (AppendInput, error) {
	input.Description = strings.TrimSpace(input.Description)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.SourceType != nil {
		clean := strings.TrimSpace(*input.SourceType)
		input.SourceType = &clean
	}
	if input.SourceID != nil {
		clean := strings.TrimSpace(*input.SourceID)
		input.SourceID = &clean
	}
	if err := validateAppend(input); err != nil {
		return AppendInput{}, err
	}
	if err := s.ensureParticipant(ctx, input.ContestID, input.EventParticipantID); err != nil {
		return AppendInput{}, err
	}
	return input, nil
}

func (s *Service) Adjust(
	ctx context.Context,
	actor Actor,
	contestID, participantID string,
	input AdjustmentInput,
) (*AdjustmentResult, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	input.Reason = strings.TrimSpace(input.Reason)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.Amount == 0 || len(input.Reason) < 3 || len(input.Reason) > 1000 ||
		len(input.IdempotencyKey) < 8 || len(input.IdempotencyKey) > 200 {
		return nil, ErrValidation
	}
	if err := s.ensureParticipant(ctx, contestID, participantID); err != nil {
		return nil, err
	}
	actorID := actor.UserID
	entry, created, err := s.repo.AppendAdminAdjustment(ctx, actor.UserID, AppendInput{
		ContestID:          contestID,
		EventParticipantID: participantID,
		Amount:             input.Amount,
		Type:               TypeAdminAdjustment,
		Description:        input.Reason,
		CreatedByUserID:    &actorID,
		IdempotencyKey:     input.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	balance, err := s.Balance(ctx, contestID, participantID)
	if err != nil {
		return nil, err
	}
	return &AdjustmentResult{Entry: *entry, Balance: balance, Replayed: !created}, nil
}

func validateAppend(input AppendInput) error {
	if input.ContestID == "" || input.EventParticipantID == "" || input.Amount == 0 ||
		input.Description == "" || len(input.Description) > 1000 ||
		len(input.IdempotencyKey) < 8 || len(input.IdempotencyKey) > 200 {
		return ErrValidation
	}
	switch input.Type {
	case TypeLectureAttendance, TypeTaskReward, TypeRefund:
		if input.Amount <= 0 || !validSource(input) {
			return ErrValidation
		}
	case TypeMerchPurchase:
		if input.Amount >= 0 || !validSource(input) {
			return ErrValidation
		}
	case TypeAdminAdjustment:
		if input.CreatedByUserID == nil || *input.CreatedByUserID == "" ||
			input.SourceType != nil || input.SourceID != nil {
			return ErrValidation
		}
	default:
		return ErrValidation
	}
	return nil
}

func validSource(input AppendInput) bool {
	return input.SourceType != nil && *input.SourceType != "" &&
		input.SourceID != nil && *input.SourceID != ""
}
