package evaluation

import (
	"context"
	"errors"
	"math"
	"strings"
)

const scoringUICriteria = "CRITERIA"
const scoringUILives = "LIVES"
const scoringUINone = "NONE"

func (s *Service) JuryScorecard(ctx context.Context, a Actor, challengeID, contestantUserID string) (*Scorecard, error) {
	ch, err := s.juryChallenge(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	card := &Scorecard{Criteria: []ScorecardCriterion{}, ScoringUI: scoringUINone}
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if errors.Is(err, ErrNotFound) {
		return card, nil
	}
	if err != nil {
		return nil, err
	}
	sess, err := s.repo.EnsureSession(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	card.Configured = true
	card.SchemeType = scheme.Type
	card.Editable = scoringEditable(scheme.EditPolicy, sess.State)
	if UsesCriteria(scheme.Type) {
		card.ScoringUI = scoringUICriteria
	} else {
		card.ScoringUI = scoringUINone
		return card, nil
	}

	contestantUserID = resolveScorecardContestant(contestantUserID, sess.CurrentContestantUserID)
	if contestantUserID != "" {
		ok, err := s.repo.ContestantInContest(ctx, ch.ContestID, contestantUserID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrValidation
		}
		perf, err := s.repo.EnsurePerformance(ctx, ch.ID, contestantUserID)
		if err != nil {
			return nil, err
		}
		card.PerformanceID = &perf.ID
		list, err := s.repo.ListLiveContestants(ctx, ch.ContestID, ch.ID)
		if err != nil {
			return nil, err
		}
		s.decorateAvatars(ctx, list)
		for i := range list {
			if list[i].UserID == contestantUserID {
				c := list[i]
				card.Contestant = &c
				break
			}
		}
	}

	values := map[string]ScoreValue{}
	if card.PerformanceID != nil {
		rows, err := s.repo.ListScoreValues(ctx, *card.PerformanceID, a.UserID)
		if err != nil {
			return nil, err
		}
		for _, v := range rows {
			values[v.CriterionID] = v
		}
	}

	var total float64
	filled := 0
	for i := range scheme.Criteria {
		item := ScorecardCriterion{Criterion: scheme.Criteria[i]}
		if v, ok := values[item.ID]; ok {
			score := v.Score
			item.Score = &score
			item.Revision = v.Revision
			total += v.Score * item.Weight
			filled++
		}
		card.Criteria = append(card.Criteria, item)
	}
	card.Filled = filled
	if filled > 0 {
		card.Total = &total
	}
	return card, nil
}

func (s *Service) JurySetScore(ctx context.Context, a Actor, challengeID string, in ScoreMutation) (*ScoreWriteResult, error) {
	ch, err := s.juryChallenge(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	in.PerformanceID = strings.TrimSpace(in.PerformanceID)
	in.CriterionID = strings.TrimSpace(in.CriterionID)
	in.MutationID = strings.TrimSpace(in.MutationID)
	if in.PerformanceID == "" || in.CriterionID == "" {
		return nil, ErrValidation
	}
	scheme, err := s.repo.SchemeByChallenge(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if !UsesCriteria(scheme.Type) {
		return nil, ErrValidation
	}
	sess, err := s.repo.EnsureSession(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if !scoringEditable(scheme.EditPolicy, sess.State) {
		return nil, ErrScoringClosed
	}
	perf, err := s.repo.PerformanceByID(ctx, in.PerformanceID)
	if err != nil {
		return nil, err
	}
	if perf.ChallengeID != ch.ID {
		return nil, ErrValidation
	}
	if err := s.criterionBelongs(ctx, challengeID, in.CriterionID); err != nil {
		return nil, err
	}
	var criterion *Criterion
	for i := range scheme.Criteria {
		if scheme.Criteria[i].ID == in.CriterionID {
			criterion = &scheme.Criteria[i]
			break
		}
	}
	if criterion == nil {
		return nil, ErrNotFound
	}
	if !scoreInRange(in.Score, criterion.MinScore, criterion.MaxScore) {
		return nil, ErrValidation
	}
	if scheme.CorridorMode == CorridorStrict {
		if scheme.MinScore != nil && in.Score < *scheme.MinScore {
			return nil, ErrValidation
		}
		if scheme.MaxScore != nil && in.Score > *scheme.MaxScore {
			return nil, ErrValidation
		}
	}
	if in.MutationID != "" {
		existing, err := s.repo.ScoreValueByMutation(ctx, in.MutationID)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		if err == nil {
			total, err := s.repo.RefreshSheetTotal(ctx, existing.ScoreSheetID)
			if err != nil {
				return nil, err
			}
			return &ScoreWriteResult{
				CriterionID: existing.CriterionID,
				Score:       existing.Score,
				Revision:    existing.Revision,
				Total:       total,
			}, nil
		}
	}
	sheetID, err := s.repo.EnsureScoreSheet(ctx, perf.ID, a.UserID)
	if err != nil {
		return nil, err
	}
	var mut *string
	if in.MutationID != "" {
		mut = &in.MutationID
	}
	saved, err := s.repo.UpsertScoreValue(ctx, sheetID, in.CriterionID, in.Score, mut, a.UserID)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.RefreshSheetTotal(ctx, sheetID)
	if err != nil {
		return nil, err
	}
	return &ScoreWriteResult{
		CriterionID: saved.CriterionID,
		Score:       saved.Score,
		Revision:    saved.Revision,
		Total:       total,
	}, nil
}

func scoringEditable(policy, state string) bool {
	if policy == EditAlways {
		return true
	}
	return state != StateFinished
}

func resolveScorecardContestant(requested string, sessionCurrent *string) string {
	requested = strings.TrimSpace(requested)
	if requested != "" {
		return requested
	}
	if sessionCurrent != nil {
		return strings.TrimSpace(*sessionCurrent)
	}
	return ""
}

func scoreInRange(score, min, max float64) bool {
	if score < min || score > max {
		return false
	}
	if isWhole(min) && isWhole(max) && !isWhole(score) {
		return false
	}
	return true
}

func isWhole(v float64) bool {
	return math.Abs(v-math.Round(v)) < 1e-9
}

func (s *Service) AdminScoreboard(ctx context.Context, a Actor, challengeID string) (*Scoreboard, error) {
	ch, err := s.challengeForView(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	board, err := s.buildScoreboard(ctx, ch)
	if err != nil {
		return nil, err
	}
	if err := s.attachCombined(ctx, ch, board); err != nil {
		return nil, err
	}
	if err := s.attachCorrections(ctx, ch.ID, board); err != nil {
		return nil, err
	}
	board.CanOverride = a.IsMega
	return board, nil
}

func (s *Service) buildScoreboard(ctx context.Context, ch *challengeRef) (*Scoreboard, error) {
	board := &Scoreboard{
		ScoringUI:   scoringUINone,
		Criteria:    []Criterion{},
		Jury:        []JuryPerson{},
		Contestants: []ScoreboardContestant{},
	}
	scheme, err := s.repo.SchemeByChallenge(ctx, ch.ID)
	if errors.Is(err, ErrNotFound) {
		return board, nil
	}
	if err != nil {
		return nil, err
	}
	board.Configured = true
	board.SchemeType = scheme.Type
	if scheme.Type == TypeEliminationLives {
		return s.livesScoreboard(ctx, ch, scheme, board)
	}
	if scheme.Type == TypeNumericResult {
		return s.numericScoreboard(ctx, ch, scheme, board)
	}
	if !UsesCriteria(scheme.Type) {
		return board, nil
	}
	board.ScoringUI = scoringUICriteria
	board.Criteria = scheme.Criteria
	sess, err := s.repo.EnsureSession(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	board.CurrentContestantUserID = sess.CurrentContestantUserID
	jury, err := s.repo.ListContestJury(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return nil, err
	}
	board.Jury = jury
	contestants, err := s.repo.ListLiveContestants(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return nil, err
	}
	s.decorateAvatars(ctx, contestants)
	rows, err := s.repo.ListChallengeScoreRows(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	idx := map[string]map[string]map[string]float64{}
	for _, row := range rows {
		if idx[row.ContestantUserID] == nil {
			idx[row.ContestantUserID] = map[string]map[string]float64{}
		}
		if idx[row.ContestantUserID][row.JuryUserID] == nil {
			idx[row.ContestantUserID][row.JuryUserID] = map[string]float64{}
		}
		idx[row.ContestantUserID][row.JuryUserID][row.CriterionID] = row.Score
	}
	for _, c := range contestants {
		item := ScoreboardContestant{LiveContestant: c, Sheets: make([]ScoreboardSheet, 0, len(jury))}
		var totals []float64
		for _, j := range jury {
			sheet := ScoreboardSheet{
				JuryUserID: j.UserID,
				Values:     make([]ScoreboardValue, 0, len(scheme.Criteria)),
			}
			vals := idx[c.UserID][j.UserID]
			var total float64
			for k := range scheme.Criteria {
				crit := scheme.Criteria[k]
				cell := ScoreboardValue{CriterionID: crit.ID}
				if s, ok := vals[crit.ID]; ok {
					score := s
					cell.Score = &score
					sheet.Filled++
					total += s * crit.Weight
				}
				sheet.Values = append(sheet.Values, cell)
			}
			if sheet.Filled > 0 {
				t := total
				sheet.Total = &t
				totals = append(totals, total)
			}
			item.Sheets = append(item.Sheets, sheet)
		}
		item.Average = mean(totals)
		item.Sum = sumFloats(totals)
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

func (s *Service) livesScoreboard(ctx context.Context, ch *challengeRef, scheme *Scheme, board *Scoreboard) (*Scoreboard, error) {
	board.ScoringUI = scoringUILives
	starting := DefaultStartingLives
	if v := startingLivesOf(scheme); v != nil {
		starting = *v
	}
	board.StartingLives = &starting
	sess, err := s.repo.EnsureSession(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	q := sess.CurrentQuestionNumber
	if q < 1 {
		q = 1
	}
	board.CurrentQuestionNumber = q
	board.QuestionCount = questionCountOf(sess)
	board.CurrentContestantUserID = sess.CurrentContestantUserID
	op, err := s.repo.ChallengeOperator(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	if op == nil {
		if id := operatorIDOf(scheme); id != "" {
			op = &id
		}
	}
	board.OperatorUserID = op
	jury, err := s.repo.ListContestJury(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return nil, err
	}
	board.Jury = jury
	contestants, err := s.repo.ListLiveContestants(ctx, ch.ContestID, ch.ID)
	if err != nil {
		return nil, err
	}
	s.decorateAvatars(ctx, contestants)
	all, err := s.repo.ListLifeEvents(ctx, ch.ID, nil)
	if err != nil {
		return nil, err
	}
	byJury := map[string][]LifeEvent{}
	for _, e := range all {
		byJury[e.CreatedByUserID] = append(byJury[e.CreatedByUserID], e)
	}
	marks, err := s.repo.ListAnswerMarks(ctx, ch.ID, nil, nil)
	if err != nil {
		return nil, err
	}
	byJuryMarks := map[string][]AnswerMark{}
	for _, m := range marks {
		byJuryMarks[m.JuryUserID] = append(byJuryMarks[m.JuryUserID], m)
	}
	keys, err := s.repo.ListQuestionKeys(ctx, ch.ID)
	if err != nil {
		return nil, err
	}
	var officialEvents []LifeEvent
	if op != nil {
		officialEvents = byJury[*op]
	}
	official := buildLivesBoard(starting, q, board.QuestionCount, op, op, contestants, officialEvents)
	applyQuestionKeys(official, keys)
	rows := livesRowByUser(official)
	for _, c := range contestants {
		item := ScoreboardContestant{LiveContestant: c, Sheets: []ScoreboardSheet{}}
		if row, ok := rows[c.UserID]; ok {
			lives := row.Lives
			item.Lives = &lives
			item.Eliminated = row.Eliminated
			item.EliminatedQuestion = row.EliminatedQuestion
			item.Rank = row.Rank
		}
		board.Contestants = append(board.Contestants, item)
	}
	board.LifeLogs = make([]JuryLifeLog, 0, len(jury))
	for _, j := range jury {
		viewer := j.UserID
		lb := buildLivesBoard(starting, q, board.QuestionCount, op, &viewer, contestants, byJury[j.UserID])
		applyQuestionKeys(lb, keys)
		decorateViewerMarks(lb, contestants, byJuryMarks[j.UserID], q, lb.Official)
		board.LifeLogs = append(board.LifeLogs, JuryLifeLog{
			JuryUserID: j.UserID,
			Official:   lb.Official,
			Questions:  lb.Questions,
			Rows:       lb.Rows,
		})
	}
	return board, nil
}

func mean(xs []float64) *float64 {
	if len(xs) == 0 {
		return nil
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	v := s / float64(len(xs))
	return &v
}

func sumFloats(xs []float64) *float64 {
	if len(xs) == 0 {
		return nil
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return &s
}

// competitionRanks — олимпийская система: равные баллы делят место, следующие номера пропускаются (1,2,2,4).
func competitionRanks(scores []*float64) []*int {
	ranks := make([]*int, len(scores))
	for i, s := range scores {
		if s == nil {
			continue
		}
		better := 0
		for _, o := range scores {
			if o == nil {
				continue
			}
			if *o > *s+1e-9 {
				better++
			}
		}
		r := better + 1
		ranks[i] = &r
	}
	return ranks
}
