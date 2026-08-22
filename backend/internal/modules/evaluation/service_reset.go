package evaluation

import (
	"context"
	"errors"
	"strings"
)

func (s *Service) confirmDestructive(ctx context.Context, a Actor, password string) error {
	if strings.TrimSpace(password) == "" {
		return ErrWrongPassword
	}
	if s.passwords == nil {
		return ErrForbidden
	}
	if err := s.passwords.VerifyUserPassword(ctx, a.UserID, password); err != nil {
		return ErrWrongPassword
	}
	return nil
}

func (s *Service) ResetResults(ctx context.Context, a Actor, challengeID, password string) error {
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return err
	}
	if err := s.confirmDestructive(ctx, a, password); err != nil {
		return err
	}
	if err := s.resetRuntime(ctx, a, ch, nil); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_RESET_RESULTS", "challenge", challengeID, map[string]any{
		"contest_id": ch.ContestID,
	})
	return nil
}

func (s *Service) ReplaceJury(ctx context.Context, a Actor, challengeID, password string, juryUserIDs []string) error {
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return err
	}
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if scheme != nil && ExclusiveChallengeJury(scheme.Type) {
		return ErrValidation
	}
	if err := s.confirmDestructive(ctx, a, password); err != nil {
		return err
	}
	ids := uniqueIDs(juryUserIDs)
	for _, id := range ids {
		remote, err := s.repo.UserIsRemoteJuryInContest(ctx, id, ch.ContestID)
		if err != nil {
			return err
		}
		if remote {
			return ErrValidation
		}
		ok, err := s.repo.UserIsJury(ctx, id, ch.ContestID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrValidation
		}
	}
	if err := s.resetRuntime(ctx, a, ch, &runtimeReset{
		ReplaceJury: true,
		JuryUserIDs: ids,
		ContestID:   ch.ContestID,
	}); err != nil {
		return err
	}
	if err := s.syncOperatorAfterJuryChange(ctx, ch, ids); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_REPLACE_JURY", "challenge", challengeID, map[string]any{
		"contest_id":      ch.ContestID,
		"jury_user_ids":   ids,
		"jury_count":      len(ids),
		"contest_default": len(ids) == 0,
	})
	return nil
}

func (s *Service) SetRemoteJury(ctx context.Context, a Actor, challengeID string, juryUserIDs []string) error {
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return err
	}
	scheme, err := s.requireScheme(ctx, challengeID)
	if err != nil {
		return err
	}
	if !ExclusiveChallengeJury(scheme.Type) {
		return ErrValidation
	}
	ids := uniqueIDs(juryUserIDs)
	for _, id := range ids {
		ok, err := s.repo.UserHasRemoteJuryRole(ctx, id, ch.ContestID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrValidation
		}
	}
	if err := s.repo.SetChallengeJury(ctx, ch.ContestID, ch.ID, ids); err != nil {
		return err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_REMOTE_JURY", "challenge", challengeID, map[string]any{
		"contest_id":    ch.ContestID,
		"jury_user_ids": ids,
		"jury_count":    len(ids),
	})
	return nil
}

func (s *Service) resetRuntime(ctx context.Context, a Actor, ch *challengeRef, extra *runtimeReset) error {
	sess, err := s.repo.EnsureSession(ctx, ch.ID)
	if err != nil {
		return err
	}
	next := *sess
	ctrl := a.UserID
	next.ControlledBy = &ctrl
	resetSessionIdle(&next)
	saved, err := s.repo.ResetChallengeRuntime(ctx, ch.ID, a.UserID, sess.Revision, &next, extra)
	if err != nil {
		return err
	}
	s.hub.Publish(ch.ID, LiveEvent{Type: "session.updated", Revision: saved.Revision, State: saved.State})
	s.hub.Touch(ch.ID, a.UserID)
	return nil
}

func (s *Service) syncOperatorAfterJuryChange(ctx context.Context, ch *challengeRef, assigned []string) error {
	if len(assigned) == 0 {
		return nil
	}
	op, err := s.repo.ChallengeOperator(ctx, ch.ID)
	if err != nil {
		return err
	}
	if op == nil || *op == "" {
		return nil
	}
	for _, id := range assigned {
		if id == *op {
			return nil
		}
	}
	return s.repo.SetChallengeOperator(ctx, ch.ContestID, ch.ID, "")
}

func uniqueIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
