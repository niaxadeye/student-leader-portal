package submissions

import (
	"context"

	"github.com/eazytech/student-leader-cabinet/internal/modules/challenges"
	"github.com/eazytech/student-leader-cabinet/internal/modules/contests"
)

// challengeAdapter связывает submissions с модулями challenges и contests,
// не заводя обратных зависимостей в них.
type challengeAdapter struct {
	ch       *challenges.Repo
	contests *contests.Repo
}

// NewChallengeAdapter собирает ChallengeSource из репозиториев challenges/contests.
func NewChallengeAdapter(ch *challenges.Repo, ct *contests.Repo) ChallengeSource {
	return &challengeAdapter{ch: ch, contests: ct}
}

func (a *challengeAdapter) ChallengeInfo(ctx context.Context, challengeID string) (*ChallengeInfo, error) {
	c, err := a.ch.ByID(ctx, challengeID)
	if err != nil {
		if err == challenges.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ChallengeInfo{
		ID:                   c.ID,
		ContestID:            c.ContestID,
		Status:               c.Status,
		OpenAt:               c.OpenAt,
		DeadlineAt:           c.DeadlineAt,
		CloseAt:              c.CloseAt,
		CurrentSchemaVersion: c.CurrentSchemaVersion,
		Settings:             c.Settings,
	}, nil
}

func (a *challengeAdapter) SchemaFields(ctx context.Context, challengeID string) ([]FieldInfo, error) {
	fields, err := a.ch.Fields(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	out := make([]FieldInfo, 0, len(fields))
	for i := range fields {
		f := &fields[i]
		out = append(out, FieldInfo{
			ID: f.ID, Key: f.Key, Type: f.Type, Label: f.Label,
			Required: f.Required, Settings: f.Settings,
		})
	}
	return out, nil
}

func (a *challengeAdapter) IsParticipant(ctx context.Context, userID, contestID string) (bool, error) {
	return a.ch.IsParticipant(ctx, userID, contestID)
}

func (a *challengeAdapter) HasContestAccess(ctx context.Context, userID, contestID string, isSuper bool) (bool, error) {
	return a.contests.HasContestAccess(ctx, userID, contestID, isSuper)
}
