package evaluation

import (
	"context"
	"errors"
	"strings"
)

const scoringUINumeric = "NUMERIC"

func (s *Service) SetNumericResult(ctx context.Context, a Actor, challengeID, contestantUserID string, score *float64) (*Scoreboard, error) {
	if a.IsMega {
		return nil, ErrForbidden
	}
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	contestantUserID = strings.TrimSpace(contestantUserID)
	if contestantUserID == "" {
		return nil, ErrValidation
	}
	scheme, err := s.requireScheme(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if scheme.Type != TypeNumericResult {
		return nil, ErrValidation
	}
	ok, err := s.repo.ContestantInContest(ctx, ch.ContestID, contestantUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrValidation
	}
	if score == nil {
		if err := s.repo.DeleteNumericResult(ctx, challengeID, contestantUserID); err != nil {
			return nil, err
		}
	} else {
		min, max := numericBounds(scheme)
		if *score < min-1e-9 || *score > max+1e-9 {
			return nil, ErrValidation
		}
		if err := s.repo.UpsertNumericResult(ctx, challengeID, contestantUserID, *score, a.UserID); err != nil {
			return nil, err
		}
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_NUMERIC_SCORE", "challenge", challengeID, map[string]any{
		"contestant_user_id": contestantUserID, "score": score,
	})
	return s.AdminScoreboard(ctx, a, challengeID)
}

func (s *Service) numericScoreboard(ctx context.Context, ch *challengeRef, scheme *Scheme, board *Scoreboard) (*Scoreboard, error) {
	board.ScoringUI = scoringUINumeric
	min, max := numericBounds(scheme)
	board.MinScore = &min
	board.MaxScore = &max
	contestants, err := s.repo.ListLiveContestants(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return nil, err
	}
	s.decorateAvatars(ctx, contestants)
	rows, err := s.repo.ListNumericResults(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	byUser := map[string]float64{}
	for _, row := range rows {
		byUser[row.ContestantUserID] = row.Score
	}
	for _, c := range contestants {
		item := ScoreboardContestant{LiveContestant: c, Sheets: []ScoreboardSheet{}}
		if score, ok := byUser[c.UserID]; ok {
			v := score
			item.NumericScore = &v
			item.Sum = &v
		}
		board.Contestants = append(board.Contestants, item)
	}
	sums := make([]*float64, len(board.Contestants))
	for i := range board.Contestants {
		sums[i] = board.Contestants[i].Sum
	}
	ranks := competitionRanks(sums)
	for i := range board.Contestants {
		board.Contestants[i].Rank = ranks[i]
	}
	return board, nil
}

func numericBounds(scheme *Scheme) (min, max float64) {
	min = MinNumericScore
	max = 100
	if scheme.MinScore != nil {
		min = *scheme.MinScore
	}
	if scheme.MaxScore != nil {
		max = *scheme.MaxScore
	}
	return min, max
}

func (s *Service) rejectIfNoLive(ctx context.Context, challengeID string) error {
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if scheme != nil && !HasLiveSession(scheme.Type) {
		return ErrValidation
	}
	return nil
}
