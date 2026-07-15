package submissions

import "context"

// AdminList — работы по испытанию для дирекции (SITE.md §7.6). Требует доступа к конкурсу.
func (s *Service) AdminList(ctx context.Context, a Actor, challengeID, status string, limit, offset int) ([]AdminRow, int, error) {
	if err := s.ensureAdmin(ctx, a, challengeID); err != nil {
		return nil, 0, err
	}
	return s.repo.AdminList(ctx, AdminListFilter{
		ChallengeID: challengeID, Status: status, Limit: limit, Offset: offset,
	})
}

// AdminGet — одна работа с файлами, ФИО и историей ревизий.
func (s *Service) AdminGet(ctx context.Context, a Actor, submissionID string) (*Submission, []Revision, error) {
	sub, err := s.repo.ByID(ctx, submissionID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.ensureAdmin(ctx, a, sub.ChallengeID); err != nil {
		return nil, nil, err
	}
	if err := s.repo.LoadContestant(ctx, sub); err != nil {
		return nil, nil, err
	}
	if _, err := s.withFiles(ctx, sub); err != nil {
		return nil, nil, err
	}
	revs, err := s.repo.Revisions(ctx, submissionID)
	if err != nil {
		return nil, nil, err
	}
	return sub, revs, nil
}

// ensureAdmin — доступ к конкурсу испытания (SUPER_ADMIN ∨ ADMIN scoped).
func (s *Service) ensureAdmin(ctx context.Context, a Actor, challengeID string) error {
	info, err := s.source.ChallengeInfo(ctx, challengeID)
	if err != nil {
		return err
	}
	ok, err := s.source.HasContestAccess(ctx, a.UserID, info.ContestID, a.IsSuper)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}
