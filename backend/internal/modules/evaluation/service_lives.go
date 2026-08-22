package evaluation

import (
	"context"
	"errors"
	"strings"
	"time"
)

func (s *Service) attachLives(ctx context.Context, snap *LiveSnapshot, viewerJuryID string) error {
	if snap == nil {
		return nil
	}
	scheme, err := s.repo.SchemeByChallenge(ctx, snap.ChallengeID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	snap.SchemeType = scheme.Type
	if scheme.Type != TypeEliminationLives {
		return nil
	}
	starting := DefaultStartingLives
	if v := startingLivesOf(scheme); v != nil {
		starting = *v
	}
	op, err := s.challengeOperator(ctx, snap.ChallengeID, scheme)
	if err != nil {
		return err
	}
	q := snap.Session.CurrentQuestionNumber
	if q < 1 {
		q = 1
	}
	snap.StartingLives = &starting
	snap.CurrentQuestionNumber = q
	snap.QuestionCount = questionCountOf(&snap.Session)
	snap.OperatorUserID = op

	var officialEvents []LifeEvent
	if op != nil {
		officialEvents, err = s.repo.ListLifeEvents(ctx, snap.ChallengeID, op)
		if err != nil {
			return err
		}
	}
	board := buildLivesBoard(starting, q, questionCountOf(&snap.Session), op, op, snap.Contestants, officialEvents)
	var viewer *string
	if viewerJuryID != "" {
		viewer = &viewerJuryID
	} else {
		viewer = op
	}
	official := op != nil && viewerJuryID != "" && *op == viewerJuryID
	board.Official = official || viewerJuryID == ""
	board.ViewerUserID = viewer

	keys, err := s.repo.ListQuestionKeys(ctx, snap.ChallengeID)
	if err != nil {
		return err
	}
	applyQuestionKeys(board, keys)

	markOwner := viewerJuryID
	if markOwner == "" && op != nil {
		markOwner = *op
	}
	if markOwner != "" {
		marks, err := s.repo.ListAnswerMarks(ctx, snap.ChallengeID, &markOwner, nil)
		if err != nil {
			return err
		}
		decorateViewerMarks(board, snap.Contestants, marks, q, official)
	}

	snap.ShowAnswerKey = viewerJuryID == ""
	if snap.ShowAnswerKey {
		snap.CorrectAnswer = board.CorrectAnswer
		snap.QuestionKeys = questionKeysList(keys, questionCountOf(&snap.Session))
	} else {
		board.CorrectAnswer = nil
		for i := range board.Questions {
			board.Questions[i].CorrectAnswer = nil
			if !official {
				for j := range board.Questions[i].Answers {
					board.Questions[i].Answers[j].Mismatch = false
				}
			}
		}
		if !official {
			for i := range board.Rows {
				board.Rows[i].Mismatch = false
				board.Rows[i].CanReveal = false
			}
		}
	}
	snap.Lives = board
	return nil
}

func (s *Service) StepQuestion(ctx context.Context, a Actor, challengeID string, delta, baseRev int) (*LiveSnapshot, error) {
	if delta != 1 && delta != -1 {
		return nil, ErrValidation
	}
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_QUESTION", func(_ time.Time, sess *Session, _ *challengeRef, _ []PhaseTemplate) error {
		if sess.State == StateFinished {
			return ErrValidation
		}
		scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
		if err != nil {
			return err
		}
		if scheme.Type != TypeEliminationLives {
			return ErrValidation
		}
		q := sess.CurrentQuestionNumber
		if q < 1 {
			q = 1
		}
		q += delta
		if q < 1 {
			q = 1
		}
		if planned := questionCountOf(sess); planned > 0 && q > planned {
			return ErrValidation
		}
		sess.CurrentQuestionNumber = q
		return nil
	})
}

func (s *Service) JuryWrongAnswer(ctx context.Context, a Actor, challengeID, contestantUserID string) (*LiveSnapshot, error) {
	ch, err := s.juryChallenge(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.requireLivesScheme(ctx, challengeID); err != nil {
		return nil, err
	}
	ok, err := s.repo.ContestantInContest(ctx, ch.ContestID, contestantUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrValidation
	}
	sess, err := s.repo.EnsureSession(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if sess.State == StateFinished {
		return nil, ErrScoringClosed
	}
	q := sess.CurrentQuestionNumber
	if q < 1 {
		q = 1
	}
	if _, err := s.repo.OpenWrongEvent(ctx, challengeID, contestantUserID, a.UserID, q); err == nil {
		return nil, ErrLifeDuplicate
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	lives, err := s.viewerLives(ctx, challengeID, contestantUserID, a.UserID)
	if err != nil {
		return nil, err
	}
	if lives <= 0 {
		return nil, ErrLifeGone
	}
	if _, err := s.repo.InsertLifeEvent(ctx, LifeEvent{
		ChallengeID:      challengeID,
		ContestantUserID: contestantUserID,
		QuestionNumber:   q,
		Delta:            -1,
		Reason:           ReasonWrongAnswer,
		CreatedByUserID:  a.UserID,
	}); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_LIFE_WRONG", "life_event", challengeID, map[string]any{
		"challenge_id": challengeID, "contestant_user_id": contestantUserID, "question_number": q,
	})
	s.hub.Publish(challengeID, LiveEvent{Type: "session.updated", Revision: sess.Revision, State: sess.State})
	return s.JuryLive(ctx, a, challengeID)
}

func (s *Service) JuryRestoreLife(ctx context.Context, a Actor, challengeID, contestantUserID string, question int) (*LiveSnapshot, error) {
	if question < 1 {
		return nil, ErrValidation
	}
	ch, err := s.juryChallenge(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.requireLivesScheme(ctx, challengeID); err != nil {
		return nil, err
	}
	ok, err := s.repo.ContestantInContest(ctx, ch.ContestID, contestantUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrValidation
	}
	sess, err := s.repo.EnsureSession(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if sess.State == StateFinished {
		return nil, ErrScoringClosed
	}
	op, err := s.challengeOperator(ctx, challengeID, nil)
	if err != nil {
		return nil, err
	}
	author := a.UserID
	if op != nil {
		author = *op
	}
	open, err := s.repo.OpenWrongEvent(ctx, challengeID, contestantUserID, author, question)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrValidation
	}
	if err != nil {
		return nil, err
	}
	rev := open.ID
	if _, err := s.repo.InsertLifeEvent(ctx, LifeEvent{
		ChallengeID:         challengeID,
		ContestantUserID:    contestantUserID,
		QuestionNumber:      question,
		Delta:               1,
		Reason:              ReasonRestore,
		CreatedByUserID:     author,
		ReversesLifeEventID: &rev,
	}); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_LIFE_RESTORE", "life_event", challengeID, map[string]any{
		"challenge_id": challengeID, "contestant_user_id": contestantUserID, "question_number": question,
	})
	s.hub.Publish(challengeID, LiveEvent{Type: "session.updated", Revision: sess.Revision, State: sess.State})
	return s.JuryLive(ctx, a, challengeID)
}

func (s *Service) challengeOperator(ctx context.Context, challengeID string, scheme *Scheme) (*string, error) {
	op, err := s.repo.ChallengeOperator(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if op != nil {
		return op, nil
	}
	if scheme == nil {
		scheme, err = s.repo.SchemeByChallenge(ctx, challengeID)
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}
	if id := operatorIDOf(scheme); id != "" {
		return &id, nil
	}
	return nil, nil
}

func (s *Service) requireLivesScheme(ctx context.Context, challengeID string) error {
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if errors.Is(err, ErrNotFound) {
		return ErrValidation
	}
	if err != nil {
		return err
	}
	if scheme.Type != TypeEliminationLives {
		return ErrValidation
	}
	return nil
}

func (s *Service) viewerLives(ctx context.Context, challengeID, contestantID, juryID string) (int, error) {
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if err != nil {
		return 0, err
	}
	starting := DefaultStartingLives
	if v := startingLivesOf(scheme); v != nil {
		starting = *v
	}
	events, err := s.repo.ListLifeEvents(ctx, challengeID, &juryID)
	if err != nil {
		return 0, err
	}
	sum := 0
	for _, e := range events {
		if e.ContestantUserID == contestantID {
			sum += e.Delta
		}
	}
	lives := starting + sum
	if lives < 0 {
		return 0, nil
	}
	return lives, nil
}

func (s *Service) SetCorrectAnswer(ctx context.Context, a Actor, challengeID, raw string) (*LiveSnapshot, error) {
	answer, err := normalizeAnswer(raw)
	if err != nil {
		return nil, err
	}
	if err := s.requireLivesScheme(ctx, challengeID); err != nil {
		return nil, err
	}
	_, err = s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	sess, err := s.repo.EnsureSession(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if sess.State == StateFinished {
		return nil, ErrScoringClosed
	}
	q := questionNumberOf(sess)
	if err := s.repo.UpsertQuestionKey(ctx, challengeID, q, answer, a.UserID); err != nil {
		return nil, err
	}
	if err := s.syncOperatorQuestion(ctx, challengeID, q); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_QUESTION_KEY", "evaluation_session", challengeID, map[string]any{
		"challenge_id": challengeID, "question_number": q, "correct_answer": answer,
	})
	s.hub.Publish(challengeID, LiveEvent{Type: "session.updated", Revision: sess.Revision, State: sess.State})
	return s.Live(ctx, a, challengeID, false)
}

type questionPlanItem struct {
	Number int
	Answer string
}

func (s *Service) SetQuestionPlan(ctx context.Context, a Actor, challengeID string, count int, items []questionPlanItem) (*LiveSnapshot, error) {
	if count < MinLiveQuestions || count > MaxLiveQuestions {
		return nil, ErrValidation
	}
	if err := s.requireLivesScheme(ctx, challengeID); err != nil {
		return nil, err
	}
	_, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	keys := map[int]string{}
	for _, item := range items {
		ans, err := normalizeAnswer(item.Answer)
		if err != nil || item.Number < 1 || item.Number > count {
			return nil, ErrValidation
		}
		if _, dup := keys[item.Number]; dup {
			return nil, ErrValidation
		}
		keys[item.Number] = ans
	}
	if len(keys) != count {
		return nil, ErrValidation
	}
	sess, err := s.repo.EnsureSession(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if sess.State == StateFinished {
		return nil, ErrScoringClosed
	}
	if err := s.repo.ReplaceQuestionKeys(ctx, challengeID, a.UserID, keys); err != nil {
		return nil, err
	}
	if err := s.repo.SetQuestionCount(ctx, challengeID, count); err != nil {
		return nil, err
	}
	for n := 1; n <= count; n++ {
		if err := s.syncOperatorQuestion(ctx, challengeID, n); err != nil {
			return nil, err
		}
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_QUESTION_PLAN", "evaluation_session", challengeID, map[string]any{
		"challenge_id": challengeID, "question_count": count,
	})
	updated, err := s.repo.SessionByChallenge(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	s.hub.Publish(challengeID, LiveEvent{Type: "session.updated", Revision: updated.Revision, State: updated.State})
	return s.Live(ctx, a, challengeID, false)
}

func (s *Service) JurySetAnswer(ctx context.Context, a Actor, challengeID, contestantUserID, raw string) (*LiveSnapshot, error) {
	answer, err := normalizeAnswer(raw)
	if err != nil {
		return nil, err
	}
	contestantUserID = strings.TrimSpace(contestantUserID)
	if contestantUserID == "" {
		return nil, ErrValidation
	}
	ch, err := s.juryChallenge(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.requireLivesScheme(ctx, challengeID); err != nil {
		return nil, err
	}
	ok, err := s.repo.ContestantInContest(ctx, ch.ContestID, contestantUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrValidation
	}
	sess, err := s.repo.EnsureSession(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if sess.State == StateFinished {
		return nil, ErrScoringClosed
	}
	q := questionNumberOf(sess)
	if err := s.repo.UpsertAnswerMark(ctx, challengeID, contestantUserID, a.UserID, q, answer); err != nil {
		return nil, err
	}
	op, err := s.challengeOperator(ctx, challengeID, nil)
	if err != nil {
		return nil, err
	}
	if op != nil && *op == a.UserID {
		if err := s.syncLifeForMark(ctx, challengeID, contestantUserID, a.UserID, q, answer); err != nil {
			return nil, err
		}
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_LIFE_ANSWER", "life_answer_mark", challengeID, map[string]any{
		"challenge_id": challengeID, "contestant_user_id": contestantUserID, "question_number": q, "answer": answer,
	})
	s.hub.Publish(challengeID, LiveEvent{Type: "session.updated", Revision: sess.Revision, State: sess.State})
	return s.JuryLive(ctx, a, challengeID)
}

func (s *Service) syncOperatorQuestion(ctx context.Context, challengeID string, question int) error {
	op, err := s.challengeOperator(ctx, challengeID, nil)
	if err != nil || op == nil {
		return err
	}
	marks, err := s.repo.ListAnswerMarks(ctx, challengeID, op, &question)
	if err != nil {
		return err
	}
	for _, m := range marks {
		if err := s.syncLifeForMark(ctx, challengeID, m.ContestantUserID, *op, question, m.Answer); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) syncLifeForMark(ctx context.Context, challengeID, contestantID, juryID string, question int, mark string) error {
	correct, err := s.repo.QuestionKey(ctx, challengeID, question)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	open, err := s.repo.OpenWrongEvent(ctx, challengeID, contestantID, juryID, question)
	hasOpen := true
	if errors.Is(err, ErrNotFound) {
		hasOpen = false
		err = nil
	}
	if err != nil {
		return err
	}
	lives, err := s.viewerLives(ctx, challengeID, contestantID, juryID)
	if err != nil {
		return err
	}
	switch lifeSyncAction(mark, correct, hasOpen, lives) {
	case "miss":
		_, err = s.repo.InsertLifeEvent(ctx, LifeEvent{
			ChallengeID:      challengeID,
			ContestantUserID: contestantID,
			QuestionNumber:   question,
			Delta:            -1,
			Reason:           ReasonWrongAnswer,
			CreatedByUserID:  juryID,
		})
	case "restore":
		rev := open.ID
		_, err = s.repo.InsertLifeEvent(ctx, LifeEvent{
			ChallengeID:         challengeID,
			ContestantUserID:    contestantID,
			QuestionNumber:      question,
			Delta:               1,
			Reason:              ReasonRestore,
			CreatedByUserID:     juryID,
			ReversesLifeEventID: &rev,
		})
	}
	return err
}
