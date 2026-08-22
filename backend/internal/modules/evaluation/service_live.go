package evaluation

import (
	"context"
	"errors"
	"strings"
	"time"
)

func (s *Service) Live(ctx context.Context, a Actor, challengeID string, edit bool) (*LiveSnapshot, error) {
	ch, err := s.liveChallenge(ctx, a, challengeID, edit)
	if err != nil {
		return nil, err
	}
	s.hub.Touch(challengeID, a.UserID)
	return s.buildSnapshot(ctx, ch)
}

func (s *Service) JuryLive(ctx context.Context, a Actor, challengeID string) (*LiveSnapshot, error) {
	ch, err := s.juryChallenge(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	s.hub.Touch(challengeID, a.UserID)
	snap, err := s.buildSnapshot(ctx, ch)
	if err != nil {
		return nil, err
	}
	if err := s.attachLives(ctx, snap, a.UserID); err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Service) liveChallenge(ctx context.Context, a Actor, challengeID string, edit bool) (*challengeRef, error) {
	if edit {
		return s.challengeForEdit(ctx, a, challengeID)
	}
	return s.challengeForView(ctx, a, challengeID)
}

func (s *Service) Start(ctx context.Context, a Actor, challengeID string, baseRev int) (*LiveSnapshot, error) {
	if err := s.rejectIfNoLive(ctx, challengeID); err != nil {
		return nil, err
	}
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_START", func(now time.Time, sess *Session, ch *challengeRef, phases []PhaseTemplate) error {
		switch sess.State {
		case StateNotStarted, StatePreparing, StatePaused:
		default:
			return ErrValidation
		}
		scheme, err := s.repo.SchemeByChallenge(ctx, ch.ID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if scheme != nil && scheme.Type == TypeEliminationLives {
			if sess.State == StatePaused && sess.PausedAt != nil {
				sess.PausedAt = nil
			}
			sess.State = StateLive
			if sess.StartedAt == nil {
				sess.StartedAt = &now
			}
			clearPhaseTimer(sess)
			return nil
		}
		switch sess.State {
		case StatePaused:
			if sess.PausedAt != nil {
				sess.AccumulatedPauseSeconds += now.Sub(*sess.PausedAt).Seconds()
				sess.PausedAt = nil
			}
			restorePhaseOrLive(sess, phases)
		default:
			sess.State = StateLive
			if sess.StartedAt == nil {
				sess.StartedAt = &now
			}
			if sess.CurrentPhaseID == nil {
				if speech := firstSpeechPhase(phases); speech != nil {
					applyPhase(sess, speech, now)
				} else if sess.PhaseStartedAt == nil {
					sess.PhaseStartedAt = &now
				}
			} else if sess.PhaseStartedAt == nil {
				sess.PhaseStartedAt = &now
			}
		}
		if sess.CurrentContestantUserID == nil {
			list, err := s.repo.ListLiveContestants(ctx, ch.ContestID, ch.ID)
			if err != nil {
				return err
			}
			if len(list) > 0 {
				id := list[0].UserID
				sess.CurrentContestantUserID = &id
			}
		}
		if sess.CurrentContestantUserID != nil {
			perf, err := s.repo.UpsertPerformance(ctx, ch.ID, *sess.CurrentContestantUserID, PerfLive, true)
			if err != nil {
				return err
			}
			sess.CurrentPerformanceID = &perf.ID
		}
		return nil
	})
}

func restorePhaseOrLive(sess *Session, phases []PhaseTemplate) {
	sess.State = StateLive
	if sess.CurrentPhaseID == nil {
		return
	}
	for i := range phases {
		if phases[i].ID == *sess.CurrentPhaseID {
			sess.State = phases[i].MapsToState
			return
		}
	}
}

func (s *Service) Pause(ctx context.Context, a Actor, challengeID string, baseRev int) (*LiveSnapshot, error) {
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_PAUSE", func(now time.Time, sess *Session, _ *challengeRef, _ []PhaseTemplate) error {
		switch sess.State {
		case StateLive, StateQuestions, StateDiscussion, StateScoring, StatePreparing, StatePostScoring:
		default:
			return ErrValidation
		}
		if sess.PausedAt == nil {
			t := now
			sess.PausedAt = &t
		}
		sess.State = StatePaused
		return nil
	})
}

func (s *Service) Finish(ctx context.Context, a Actor, challengeID string, baseRev int) (*LiveSnapshot, error) {
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_FINISH", func(now time.Time, sess *Session, ch *challengeRef, _ []PhaseTemplate) error {
		if sess.State == StateFinished {
			return ErrValidation
		}
		except := ""
		if sess.CurrentPerformanceID != nil {
			except = *sess.CurrentPerformanceID
		}
		if err := s.repo.FinishOpenPerformances(ctx, ch.ID, except); err != nil {
			return err
		}
		if sess.CurrentContestantUserID != nil {
			if _, err := s.repo.UpsertPerformance(ctx, ch.ID, *sess.CurrentContestantUserID, PerfFinished, false); err != nil {
				return err
			}
		}
		sess.State = StateFinished
		sess.FinishedAt = &now
		clearPhaseTimer(sess)
		return nil
	})
}

func (s *Service) Restart(ctx context.Context, a Actor, challengeID string, baseRev int) (*LiveSnapshot, error) {
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_RESTART", func(_ time.Time, sess *Session, ch *challengeRef, _ []PhaseTemplate) error {
		if sess.State != StateFinished {
			return ErrValidation
		}
		if err := s.repo.ResetPerformances(ctx, ch.ID); err != nil {
			return err
		}
		if err := s.repo.DeleteLifeEvents(ctx, ch.ID); err != nil {
			return err
		}
		if err := s.repo.DeleteAnswerMarks(ctx, ch.ID); err != nil {
			return err
		}
		resetSessionIdle(sess)
		return nil
	})
}

func resetSessionIdle(sess *Session) {
	sess.State = StateNotStarted
	sess.FinishedAt = nil
	sess.PausedAt = nil
	sess.StartedAt = nil
	sess.CurrentPhaseID = nil
	sess.PhaseStartedAt = nil
	sess.PhaseDurationSeconds = nil
	sess.AccumulatedPauseSeconds = 0
	sess.CurrentPerformanceID = nil
	sess.CurrentContestantUserID = nil
	sess.CurrentMatchID = nil
	sess.CurrentQuestionNumber = 1
}

func (s *Service) SetContestant(ctx context.Context, a Actor, challengeID, contestantUserID string, baseRev int) (*LiveSnapshot, error) {
	contestantUserID = strings.TrimSpace(contestantUserID)
	if contestantUserID == "" {
		return nil, ErrValidation
	}
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_CONTESTANT", func(_ time.Time, sess *Session, ch *challengeRef, _ []PhaseTemplate) error {
		if sess.State == StateFinished {
			return ErrValidation
		}
		ok, err := s.repo.ContestantInContest(ctx, ch.ContestID, contestantUserID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrValidation
		}
		perf, err := s.repo.EnsurePerformance(ctx, ch.ID, contestantUserID)
		if err != nil {
			return err
		}
		sess.CurrentContestantUserID = &contestantUserID
		sess.CurrentPerformanceID = &perf.ID
		if drawLocked(sess.State) && sess.State != StatePaused {
			holdBetweenContestants(sess)
		}
		return nil
	})
}

func (s *Service) SetPhase(ctx context.Context, a Actor, challengeID, phaseID string, baseRev int) (*LiveSnapshot, error) {
	phaseID = strings.TrimSpace(phaseID)
	if phaseID == "" {
		return nil, ErrValidation
	}
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_PHASE", func(now time.Time, sess *Session, ch *challengeRef, phases []PhaseTemplate) error {
		if sess.State == StateFinished || sess.State == StateNotStarted {
			return ErrValidation
		}
		var phase *PhaseTemplate
		for i := range phases {
			if phases[i].ID == phaseID {
				phase = &phases[i]
				break
			}
		}
		if phase == nil {
			return ErrNotFound
		}
		if phase.MapsToState != StateLive {
			if err := s.recordSpeechIfActive(ctx, now, sess, ch, phases); err != nil {
				return err
			}
		}
		applyPhase(sess, phase, now)
		return nil
	})
}

func (s *Service) RestartTimer(ctx context.Context, a Actor, challengeID string, baseRev int) (*LiveSnapshot, error) {
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_TIMER_RESTART", func(now time.Time, sess *Session, _ *challengeRef, phases []PhaseTemplate) error {
		if sess.State != StatePaused {
			return ErrValidation
		}
		var phase *PhaseTemplate
		if sess.CurrentPhaseID != nil {
			for i := range phases {
				if phases[i].ID == *sess.CurrentPhaseID {
					phase = &phases[i]
					break
				}
			}
		}
		if phase == nil {
			phase = firstSpeechPhase(phases)
		}
		if phase == nil {
			return ErrValidation
		}
		applyPhase(sess, phase, now)
		return nil
	})
}

func (s *Service) CompleteContestant(ctx context.Context, a Actor, challengeID string, baseRev int) (*LiveSnapshot, error) {
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_COMPLETE_CONTESTANT", func(now time.Time, sess *Session, ch *challengeRef, phases []PhaseTemplate) error {
		if sess.State == StateNotStarted || sess.State == StateFinished {
			return ErrValidation
		}
		if sess.CurrentContestantUserID == nil {
			return ErrValidation
		}
		if err := s.recordSpeechIfActive(ctx, now, sess, ch, phases); err != nil {
			return err
		}
		currentID := *sess.CurrentContestantUserID
		if _, err := s.repo.UpsertPerformance(ctx, ch.ID, currentID, PerfFinished, false); err != nil {
			return err
		}
		list, err := s.repo.ListLiveContestants(ctx, ch.ContestID, ch.ID)
		if err != nil {
			return err
		}
		if nextID := nextContestantID(list, currentID); nextID != "" {
			perf, err := s.repo.EnsurePerformance(ctx, ch.ID, nextID)
			if err != nil {
				return err
			}
			sess.CurrentContestantUserID = &nextID
			sess.CurrentPerformanceID = &perf.ID
		}
		holdBetweenContestants(sess)
		return nil
	})
}

func (s *Service) EndSpeech(ctx context.Context, a Actor, challengeID string, baseRev int) (*LiveSnapshot, error) {
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_END_SPEECH", func(now time.Time, sess *Session, ch *challengeRef, phases []PhaseTemplate) error {
		if sess.State == StateNotStarted || sess.State == StateFinished || sess.State == StateApplause {
			return ErrValidation
		}
		if sess.CurrentContestantUserID == nil {
			return ErrValidation
		}
		if !speechPhaseActive(sess, phases) {
			return ErrValidation
		}
		if err := s.recordSpeechIfActive(ctx, now, sess, ch, phases); err != nil {
			return err
		}
		clearPhaseTimer(sess)
		sess.State = StateApplause
		return nil
	})
}

func (s *Service) recordSpeechIfActive(ctx context.Context, now time.Time, sess *Session, ch *challengeRef, phases []PhaseTemplate) error {
	if sess.CurrentContestantUserID == nil || !speechPhaseActive(sess, phases) {
		return nil
	}
	elapsed := phaseElapsed(now, sess.PhaseStartedAt, sess.PausedAt, sess.AccumulatedPauseSeconds)
	if elapsed == nil {
		return nil
	}
	if _, err := s.repo.EnsurePerformance(ctx, ch.ID, *sess.CurrentContestantUserID); err != nil {
		return err
	}
	return s.repo.SetSpeechDuration(ctx, ch.ID, *sess.CurrentContestantUserID, *elapsed)
}

func speechPhaseActive(sess *Session, phases []PhaseTemplate) bool {
	if sess.State == StateLive {
		return true
	}
	if sess.CurrentPhaseID == nil {
		return false
	}
	for i := range phases {
		if phases[i].ID == *sess.CurrentPhaseID {
			return phases[i].MapsToState == StateLive
		}
	}
	return false
}

type liveMutate func(now time.Time, sess *Session, ch *challengeRef, phases []PhaseTemplate) error

func (s *Service) command(ctx context.Context, a Actor, challengeID string, baseRev int, action string, fn liveMutate) (*LiveSnapshot, error) {
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	sess, err := s.repo.EnsureSession(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	phases, err := s.phasesForChallenge(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	next := *sess
	now := time.Now().UTC()
	ctrl := a.UserID
	next.ControlledBy = &ctrl
	if err := fn(now, &next, ch, phases); err != nil {
		return nil, err
	}
	saved, err := s.repo.SaveSession(ctx, baseRev, &next)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, action, "evaluation_session", saved.ID, map[string]any{
		"challenge_id": challengeID, "revision": saved.Revision, "state": saved.State,
	})
	s.hub.Publish(challengeID, LiveEvent{Type: "session.updated", Revision: saved.Revision, State: saved.State})
	s.hub.Touch(challengeID, a.UserID)
	return s.buildSnapshot(ctx, ch)
}

func applyPhase(sess *Session, phase *PhaseTemplate, now time.Time) {
	id := phase.ID
	sess.CurrentPhaseID = &id
	sess.State = phase.MapsToState
	t := now
	sess.PhaseStartedAt = &t
	sess.PhaseDurationSeconds = phase.DurationSeconds
	sess.PausedAt = nil
	sess.AccumulatedPauseSeconds = 0
}

func clearPhaseTimer(sess *Session) {
	sess.PhaseStartedAt = nil
	sess.PhaseDurationSeconds = nil
	sess.PausedAt = nil
	sess.AccumulatedPauseSeconds = 0
	sess.CurrentPhaseID = nil
}

func holdBetweenContestants(sess *Session) {
	clearPhaseTimer(sess)
	sess.State = StatePaused
}

func nextContestantID(list []LiveContestant, currentID string) string {
	for i, c := range list {
		if c.UserID == currentID && i+1 < len(list) {
			return list[i+1].UserID
		}
	}
	return ""
}

func firstSpeechPhase(phases []PhaseTemplate) *PhaseTemplate {
	for i := range phases {
		if phases[i].MapsToState == StateLive {
			return &phases[i]
		}
	}
	if len(phases) > 0 {
		return &phases[0]
	}
	return nil
}

func visiblePhases(phases []PhaseTemplate) []PhaseTemplate {
	out := make([]PhaseTemplate, 0, len(phases))
	for _, p := range phases {
		if p.MapsToState == StateScoring || p.MapsToState == StatePostScoring {
			continue
		}
		p.ScoringAllowed = true
		out = append(out, p)
	}
	return out
}

func (s *Service) phasesForChallenge(ctx context.Context, challengeID string) ([]PhaseTemplate, error) {
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if errors.Is(err, ErrNotFound) || scheme == nil {
		return []PhaseTemplate{}, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.repo.SeedDefaultPhases(ctx, scheme.ID); err != nil {
		return nil, err
	}
	list, err := s.repo.ListPhaseTemplates(ctx, scheme.ID)
	if err != nil {
		return nil, err
	}
	return visiblePhases(list), nil
}

func (s *Service) buildSnapshot(ctx context.Context, ch *challengeRef) (*LiveSnapshot, error) {
	sess, err := s.repo.EnsureSession(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	phases, err := s.phasesForChallenge(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	contestants, err := s.repo.ListLiveContestants(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return nil, err
	}
	s.decorateAvatars(ctx, contestants)
	drawn := false
	for i := range contestants {
		if contestants[i].DrawNumber != nil {
			drawn = true
			break
		}
	}
	if sess.CurrentContestantUserID == nil && len(contestants) > 0 && !drawLocked(sess.State) {
		filled, err := s.repo.FillCurrentIfEmpty(ctx, ch.ID, contestants[0].UserID)
		if err != nil {
			return nil, err
		}
		sess = filled
	}
	snap := &LiveSnapshot{
		ChallengeID:    ch.ID,
		ContestID:      ch.ContestID,
		ChallengeTitle: ch.Title,
		Session:        *sess,
		Phases:         phases,
		Contestants:    contestants,
		JuryOnline:     s.hub.Online(ch.ID),
		ServerTime:     time.Now().UTC(),
		Drawn:          drawn,
	}
	snap.TimerRemaining = timerRemaining(snap.ServerTime, sess.PhaseDurationSeconds, sess.PhaseStartedAt, sess.PausedAt, sess.AccumulatedPauseSeconds)
	if sess.CurrentPerformanceID != nil {
		p, err := s.repo.PerformanceByID(ctx, *sess.CurrentPerformanceID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		snap.Performance = p
	}
	if sess.CurrentContestantUserID != nil {
		for i := range contestants {
			if contestants[i].UserID == *sess.CurrentContestantUserID {
				c := contestants[i]
				snap.Current = &c
				break
			}
		}
	}
	if err := s.attachLives(ctx, snap, ""); err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Service) Subscribe(challengeID string) (<-chan LiveEvent, func()) {
	return s.hub.Subscribe(challengeID)
}

func (s *Service) Touch(challengeID, userID string) {
	s.hub.Touch(challengeID, userID)
}

func (s *Service) SetDurations(ctx context.Context, a Actor, challengeID string, speechSeconds, questionsSeconds, baseRev int) (*LiveSnapshot, error) {
	if speechSeconds < 30 || speechSeconds > 7200 || questionsSeconds < 30 || questionsSeconds > 7200 {
		return nil, ErrValidation
	}
	return s.command(ctx, a, challengeID, baseRev, "EVALUATION_LIVE_DURATIONS", func(_ time.Time, sess *Session, ch *challengeRef, _ []PhaseTemplate) error {
		scheme, err := s.ensureLiveScheme(ctx, a, ch)
		if err != nil {
			return err
		}
		if err := s.repo.SeedDefaultPhases(ctx, scheme.ID); err != nil {
			return err
		}
		if err := s.repo.UpdatePhaseDuration(ctx, scheme.ID, StateLive, "Выступление", speechSeconds, 0); err != nil {
			return err
		}
		if err := s.repo.UpdatePhaseDuration(ctx, scheme.ID, StateQuestions, "Вопросы", questionsSeconds, 1); err != nil {
			return err
		}
		updated, err := s.repo.ListPhaseTemplates(ctx, scheme.ID)
		if err != nil {
			return err
		}
		if sess.CurrentPhaseID != nil {
			for i := range updated {
				if updated[i].ID == *sess.CurrentPhaseID {
					sess.PhaseDurationSeconds = updated[i].DurationSeconds
					break
				}
			}
		}
		return nil
	})
}

func (s *Service) ensureLiveScheme(ctx context.Context, a Actor, ch *challengeRef) (*Scheme, error) {
	scheme, err := s.repo.SchemeByChallenge(ctx, ch.ID)
	if err == nil && scheme != nil {
		return scheme, nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	in, err := normalizeScheme(SchemeInput{Type: TypeCriteriaScoring}, ch.Title)
	if err != nil {
		return nil, err
	}
	return s.repo.UpsertScheme(ctx, ch.ID, a.UserID, in)
}

func (s *Service) ShuffleDraw(ctx context.Context, a Actor, challengeID string) (*LiveSnapshot, error) {
	return s.replaceDraw(ctx, a, challengeID, nil, true)
}

func (s *Service) ReorderDraw(ctx context.Context, a Actor, challengeID string, userIDs []string) (*LiveSnapshot, error) {
	if len(userIDs) == 0 {
		return nil, ErrValidation
	}
	return s.replaceDraw(ctx, a, challengeID, userIDs, false)
}

func (s *Service) replaceDraw(ctx context.Context, a Actor, challengeID string, requested []string, shuffle bool) (*LiveSnapshot, error) {
	if err := s.rejectIfNoLive(ctx, challengeID); err != nil {
		return nil, err
	}
	ch, err := s.challengeForEdit(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	sess, err := s.repo.EnsureSession(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if drawLocked(sess.State) {
		return nil, ErrValidation
	}
	contestants, err := s.repo.ListLiveContestants(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(contestants))
	for _, c := range contestants {
		ids = append(ids, c.UserID)
	}
	if len(ids) == 0 {
		return nil, ErrValidation
	}
	var order []string
	if shuffle {
		order = append([]string(nil), ids...)
		if err := shuffleStrings(order); err != nil {
			return nil, err
		}
	} else {
		order, err = mergeDrawOrder(ids, requested)
		if err != nil {
			return nil, err
		}
	}
	if err := s.repo.ReplaceDraw(ctx, ch.ID, order); err != nil {
		return nil, err
	}
	if len(order) > 0 {
		if err := s.repo.SetCurrentContestant(ctx, ch.ID, order[0]); err != nil {
			return nil, err
		}
	}
	saved, err := s.repo.BumpSessionRevision(ctx, ch.ID, a.UserID)
	if err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "EVALUATION_DRAW", "evaluation_session", saved.ID, map[string]any{
		"challenge_id": challengeID, "revision": saved.Revision, "shuffled": shuffle, "count": len(order),
	})
	s.hub.Publish(challengeID, LiveEvent{Type: "draw.updated", Revision: saved.Revision, State: saved.State})
	s.hub.Touch(challengeID, a.UserID)
	return s.buildSnapshot(ctx, ch)
}

func (s *Service) ContestantDraw(ctx context.Context, a Actor, challengeID string) (*ContestantDraw, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	ch, err := s.repo.ChallengeByID(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if ch.Status != "PUBLISHED" && ch.Status != "CLOSED" && !a.IsMega && !a.IsSuper {
		return nil, ErrNotFound
	}
	ok, err := s.repo.ContestantInContest(ctx, ch.ContestID, a.UserID)
	if err != nil {
		return nil, err
	}
	if !ok && !a.IsMega && !a.IsSuper {
		return nil, ErrForbidden
	}
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if scheme != nil && !HasLiveSession(scheme.Type) {
		return &ContestantDraw{Order: []ContestantDrawEntry{}}, nil
	}
	contestants, err := s.repo.ListLiveContestants(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return nil, err
	}
	out := &ContestantDraw{Order: []ContestantDrawEntry{}}
	for _, c := range contestants {
		if c.DrawNumber == nil {
			continue
		}
		out.Drawn = true
		entry := ContestantDrawEntry{
			DrawNumber:   *c.DrawNumber,
			FullName:     c.FullName,
			Organization: c.Organization,
			IsMe:         c.UserID == a.UserID,
		}
		out.Order = append(out.Order, entry)
		if entry.IsMe {
			n := *c.DrawNumber
			out.MyDrawNumber = &n
		}
	}
	out.Total = len(out.Order)
	return out, nil
}

func (s *Service) ContestantContestDraws(ctx context.Context, a Actor, contestID string) ([]MyDrawSummary, error) {
	if err := s.ensureEnabled(); err != nil {
		return nil, err
	}
	ok, err := s.repo.ContestantInContest(ctx, contestID, a.UserID)
	if err != nil {
		return nil, err
	}
	if !ok && !a.IsMega && !a.IsSuper {
		return nil, ErrForbidden
	}
	return s.repo.ListMyDraws(ctx, contestID, a.UserID)
}
