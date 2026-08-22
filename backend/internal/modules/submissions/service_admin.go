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

// ensureAdmin — доступ к конкурсу испытания на чтение. Просмотр и проверка работ
// доступны и на уровне VIEW (§1.3): владелец, назначенный ADMIN (EDIT|VIEW), мега.
func (s *Service) ensureAdmin(ctx context.Context, a Actor, challengeID string) error {
	info, err := s.source.ChallengeInfo(ctx, challengeID)
	if err != nil {
		return err
	}
	ok, err := s.source.ContestViewable(ctx, a.UserID, info.ContestID, a.IsMega)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) ensureJuryReview(ctx context.Context, a Actor, challengeID string) error {
	if s.juryReview == nil {
		return ErrForbidden
	}
	ok, err := s.juryReview(ctx, a.UserID, challengeID, a.IsMega)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func submittedStatus(status string) bool {
	return status == StatusSubmitted || status == StatusLocked
}

func (s *Service) JuryList(ctx context.Context, a Actor, challengeID string) ([]AdminRow, error) {
	if err := s.ensureJuryReview(ctx, a, challengeID); err != nil {
		return nil, err
	}
	rows, _, err := s.repo.AdminList(ctx, AdminListFilter{
		ChallengeID: challengeID, Status: "", Limit: 200, Offset: 0,
	})
	if err != nil {
		return nil, err
	}
	out := make([]AdminRow, 0, len(rows))
	for _, row := range rows {
		if submittedStatus(row.Status) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (s *Service) JuryGet(ctx context.Context, a Actor, submissionID string) (*Submission, error) {
	sub, err := s.repo.ByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureJuryReview(ctx, a, sub.ChallengeID); err != nil {
		return nil, err
	}
	if !submittedStatus(sub.Status) {
		return nil, ErrNotFound
	}
	if err := s.repo.LoadContestant(ctx, sub); err != nil {
		return nil, err
	}
	if _, err := s.withFiles(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}
