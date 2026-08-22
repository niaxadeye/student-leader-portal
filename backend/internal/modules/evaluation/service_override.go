package evaluation

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

func (s *Service) attachCorrections(ctx context.Context, challengeID string, board *Scoreboard) error {
	if board == nil {
		return nil
	}
	list, err := s.repo.ListScoreCorrections(ctx, challengeID)
	if err != nil {
		return err
	}
	board.Corrections = list
	return nil
}

func (s *Service) OverrideScore(ctx context.Context, a Actor, challengeID string, in ScoreOverrideInput) (*Scoreboard, error) {
	if !a.IsMega {
		return nil, ErrForbidden
	}
	reason, err := normalizeReason(in.Reason)
	if err != nil {
		return nil, err
	}
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	in.ContestantUserID = strings.TrimSpace(in.ContestantUserID)
	if in.ContestantUserID == "" {
		return nil, ErrValidation
	}
	ok, err := s.repo.ContestantInContest(ctx, ch.ContestID, in.ContestantUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrValidation
	}
	kind := strings.ToUpper(strings.TrimSpace(in.Kind))
	switch kind {
	case ScoreKindCriterion:
		if err := s.overrideCriterion(ctx, a, ch, in, reason); err != nil {
			return nil, err
		}
	case ScoreKindNumeric:
		if err := s.overrideNumeric(ctx, a, ch, in, reason); err != nil {
			return nil, err
		}
	default:
		return nil, ErrValidation
	}
	return s.AdminScoreboard(ctx, a, challengeID)
}

func (s *Service) overrideCriterion(ctx context.Context, a Actor, ch *challengeRef, in ScoreOverrideInput, reason string) error {
	scheme, err := s.requireScheme(ctx, ch.ID)
	if err != nil {
		return err
	}
	if !UsesCriteria(scheme.Type) {
		return ErrValidation
	}
	in.JuryUserID = strings.TrimSpace(in.JuryUserID)
	in.CriterionID = strings.TrimSpace(in.CriterionID)
	if in.JuryUserID == "" || in.CriterionID == "" {
		return ErrValidation
	}
	jury, err := s.repo.ListContestJury(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return err
	}
	if !juryContains(jury, in.JuryUserID) {
		return ErrValidation
	}
	if err := s.criterionBelongs(ctx, ch.ID, in.CriterionID); err != nil {
		return err
	}
	var criterion *Criterion
	for i := range scheme.Criteria {
		if scheme.Criteria[i].ID == in.CriterionID {
			criterion = &scheme.Criteria[i]
			break
		}
	}
	if criterion == nil {
		return ErrNotFound
	}
	if in.Score != nil && !scoreInRange(*in.Score, criterion.MinScore, criterion.MaxScore) {
		return ErrValidation
	}
	perf, err := s.repo.EnsurePerformance(ctx, ch.ID, in.ContestantUserID)
	if err != nil {
		return err
	}
	sheetID, err := s.repo.EnsureScoreSheet(ctx, perf.ID, in.JuryUserID)
	if err != nil {
		return err
	}
	old, err := s.repo.sheetHasScore(ctx, sheetID, in.CriterionID)
	if err != nil {
		return err
	}
	if scoresEqual(old, in.Score) {
		return ErrValidation
	}
	if in.Score == nil {
		if err := s.repo.DeleteScoreValue(ctx, sheetID, in.CriterionID); err != nil {
			return err
		}
	} else if _, err := s.repo.UpsertScoreValue(ctx, sheetID, in.CriterionID, *in.Score, nil, a.UserID); err != nil {
		return err
	}
	if _, err := s.repo.RefreshSheetTotal(ctx, sheetID); err != nil {
		return err
	}
	juryID := in.JuryUserID
	critID := in.CriterionID
	return s.recordCorrection(ctx, a, ch, ScoreCorrection{
		Kind:             ScoreKindCriterion,
		ActorUserID:      a.UserID,
		ContestantUserID: in.ContestantUserID,
		JuryUserID:       &juryID,
		CriterionID:      &critID,
		CriterionTitle:   criterion.Title,
		OldScore:         old,
		NewScore:         in.Score,
		Reason:           reason,
	})
}

func (s *Service) overrideNumeric(ctx context.Context, a Actor, ch *challengeRef, in ScoreOverrideInput, reason string) error {
	scheme, err := s.requireScheme(ctx, ch.ID)
	if err != nil {
		return err
	}
	if scheme.Type != TypeNumericResult {
		return ErrValidation
	}
	if in.Score != nil {
		min, max := numericBounds(scheme)
		if *in.Score < min-1e-9 || *in.Score > max+1e-9 {
			return ErrValidation
		}
	}
	old, err := s.repo.NumericScore(ctx, ch.ID, in.ContestantUserID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if errors.Is(err, ErrNotFound) {
		old = nil
	}
	if scoresEqual(old, in.Score) {
		return ErrValidation
	}
	if in.Score == nil {
		if err := s.repo.DeleteNumericResult(ctx, ch.ID, in.ContestantUserID); err != nil {
			return err
		}
	} else if err := s.repo.UpsertNumericResult(ctx, ch.ID, in.ContestantUserID, *in.Score, a.UserID); err != nil {
		return err
	}
	return s.recordCorrection(ctx, a, ch, ScoreCorrection{
		Kind:             ScoreKindNumeric,
		ActorUserID:      a.UserID,
		ContestantUserID: in.ContestantUserID,
		OldScore:         old,
		NewScore:         in.Score,
		Reason:           reason,
	})
}

func (s *Service) recordCorrection(ctx context.Context, a Actor, ch *challengeRef, c ScoreCorrection) error {
	if err := s.repo.InsertScoreCorrection(ctx, c, ch.ContestID, ch.ID); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_SCORE_OVERRIDE", "challenge", ch.ID, map[string]any{
		"contest_id":         ch.ContestID,
		"kind":               c.Kind,
		"contestant_user_id": c.ContestantUserID,
		"jury_user_id":       c.JuryUserID,
		"criterion_id":       c.CriterionID,
		"criterion_title":    c.CriterionTitle,
		"old_score":          c.OldScore,
		"new_score":          c.NewScore,
		"reason":             c.Reason,
	})
	return nil
}

func juryContains(list []JuryPerson, userID string) bool {
	for _, j := range list {
		if j.UserID == userID {
			return true
		}
	}
	return false
}

func normalizeReason(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	n := utf8.RuneCountInString(s)
	if n < MinOverrideReason || n > MaxOverrideReason {
		return "", ErrValidation
	}
	return s, nil
}

func scoresEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a-*b < 1e-9 && *b-*a < 1e-9
}
