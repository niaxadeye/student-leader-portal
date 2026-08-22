package evaluation

import (
	"sort"
	"strings"
)

func applyLifeEvents(starting int, contestants []LiveContestant, events []LifeEvent) []contestantLifeState {
	if starting < 1 {
		starting = DefaultStartingLives
	}
	states := make([]contestantLifeState, 0, len(contestants))
	idx := map[string]int{}
	for i, c := range contestants {
		draw := 9999
		if c.DrawNumber != nil {
			draw = *c.DrawNumber
		}
		states = append(states, contestantLifeState{
			ID:    c.UserID,
			Lives: starting,
			Draw:  draw,
		})
		idx[c.UserID] = i
	}
	for _, e := range events {
		i, ok := idx[e.ContestantUserID]
		if !ok {
			continue
		}
		states[i].Lives += e.Delta
		if states[i].Lives < 0 {
			states[i].Lives = 0
		}
		if states[i].Lives == 0 {
			q := e.QuestionNumber
			states[i].EliminatedQuestion = &q
		} else {
			states[i].EliminatedQuestion = nil
		}
	}
	return states
}

// eliminationRanks: выбывшие получают места с конца. Несколько выбывших
// на одном вопросе делят последний свободный номер (12, 12 → следующий 10).
// Оставшиеся сравниваются по жизням: больше жизней — выше, ничьи олимпийские сверху.
func eliminationRanks(states []contestantLifeState) map[string]int {
	n := len(states)
	ranks := map[string]int{}
	if n == 0 {
		return ranks
	}
	waves := map[int][]string{}
	var remaining []contestantLifeState
	for _, s := range states {
		if s.EliminatedQuestion != nil {
			q := *s.EliminatedQuestion
			waves[q] = append(waves[q], s.ID)
		} else {
			remaining = append(remaining, s)
		}
	}
	qs := make([]int, 0, len(waves))
	for q := range waves {
		qs = append(qs, q)
	}
	sort.Ints(qs)
	already := 0
	for _, q := range qs {
		rank := n - already
		if rank < 1 {
			rank = 1
		}
		for _, id := range waves[q] {
			ranks[id] = rank
		}
		already += len(waves[q])
	}
	sort.SliceStable(remaining, func(i, j int) bool {
		if remaining[i].Lives != remaining[j].Lives {
			return remaining[i].Lives > remaining[j].Lives
		}
		return remaining[i].Draw < remaining[j].Draw
	})
	i := 0
	for i < len(remaining) {
		j := i + 1
		for j < len(remaining) && remaining[j].Lives == remaining[i].Lives {
			j++
		}
		place := i + 1
		for k := i; k < j; k++ {
			ranks[remaining[k].ID] = place
		}
		i = j
	}
	return ranks
}

func openWrongByQuestion(events []LifeEvent) map[string]map[int]string {
	reversed := map[string]bool{}
	for _, e := range events {
		if e.ReversesLifeEventID != nil {
			reversed[*e.ReversesLifeEventID] = true
		}
	}
	out := map[string]map[int]string{}
	for _, e := range events {
		if e.Reason != ReasonWrongAnswer || reversed[e.ID] {
			continue
		}
		if out[e.ContestantUserID] == nil {
			out[e.ContestantUserID] = map[int]string{}
		}
		out[e.ContestantUserID][e.QuestionNumber] = e.ID
	}
	return out
}

func netLossByQuestion(events []LifeEvent) map[string]map[int]int {
	out := map[string]map[int]int{}
	for _, e := range events {
		if out[e.ContestantUserID] == nil {
			out[e.ContestantUserID] = map[int]int{}
		}
		out[e.ContestantUserID][e.QuestionNumber] += e.Delta
	}
	return out
}

func buildLivesBoard(starting, currentQ, planned int, operatorID, viewerID *string, contestants []LiveContestant, events []LifeEvent) *LivesBoard {
	if currentQ < 1 {
		currentQ = 1
	}
	states := applyLifeEvents(starting, contestants, events)
	ranks := eliminationRanks(states)
	stateBy := map[string]contestantLifeState{}
	for _, st := range states {
		stateBy[st.ID] = st
	}
	open := openWrongByQuestion(events)
	net := netLossByQuestion(events)
	maxQ := currentQ
	if planned > maxQ {
		maxQ = planned
	}
	for _, e := range events {
		if e.QuestionNumber > maxQ {
			maxQ = e.QuestionNumber
		}
	}
	byID := map[string]LiveContestant{}
	for _, c := range contestants {
		byID[c.UserID] = c
	}
	questions := make([]QuestionLogEntry, 0, maxQ)
	for q := 1; q <= maxQ; q++ {
		entry := QuestionLogEntry{QuestionNumber: q, Current: q == currentQ, Losses: []QuestionLoss{}, Answers: []QuestionAnswerMark{}}
		for _, c := range contestants {
			if net[c.UserID][q] >= 0 {
				continue
			}
			entry.Losses = append(entry.Losses, QuestionLoss{
				ContestantUserID: c.UserID,
				FullName:         c.FullName,
				Organization:     c.Organization,
				AvatarURL:        c.AvatarURL,
			})
		}
		questions = append(questions, entry)
	}
	rows := make([]LivesRow, 0, len(contestants))
	for _, c := range contestants {
		st := stateBy[c.UserID]
		rank := ranks[c.UserID]
		restore := make([]int, 0)
		for q := range open[c.UserID] {
			restore = append(restore, q)
		}
		sort.Ints(restore)
		row := LivesRow{
			UserID:             c.UserID,
			Lives:              st.Lives,
			Eliminated:         st.EliminatedQuestion != nil,
			EliminatedQuestion: st.EliminatedQuestion,
			RestoreQuestions:   restore,
		}
		if rank > 0 {
			r := rank
			row.Rank = &r
		}
		rows = append(rows, row)
	}
	official := operatorID != nil && viewerID != nil && *operatorID == *viewerID
	return &LivesBoard{
		StartingLives:   starting,
		CurrentQuestion: currentQ,
		OperatorUserID:  operatorID,
		ViewerUserID:    viewerID,
		Official:        official,
		Questions:       questions,
		Rows:            rows,
	}
}

func livesRowByUser(board *LivesBoard) map[string]LivesRow {
	out := map[string]LivesRow{}
	if board == nil {
		return out
	}
	for _, row := range board.Rows {
		out[row.UserID] = row
	}
	return out
}

func normalizeAnswer(raw string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case AnswerYes, "DA", "ДА":
		return AnswerYes, nil
	case AnswerNo, "NET", "НЕТ":
		return AnswerNo, nil
	default:
		return "", ErrValidation
	}
}

// lifeSyncAction: после отметки ответственного и ключа вопроса.
// miss — снять жизнь, restore — вернуть, пусто — ничего не делать.
func lifeSyncAction(mark, correct string, hasOpenWrong bool, lives int) string {
	if mark == "" || correct == "" {
		return ""
	}
	if mark != correct && !hasOpenWrong && lives > 0 {
		return "miss"
	}
	if mark == correct && hasOpenWrong {
		return "restore"
	}
	return ""
}

func applyQuestionKeys(board *LivesBoard, keys map[int]string) {
	if board == nil {
		return
	}
	if a, ok := keys[board.CurrentQuestion]; ok {
		v := a
		board.CorrectAnswer = &v
	}
	for i := range board.Questions {
		if a, ok := keys[board.Questions[i].QuestionNumber]; ok {
			v := a
			board.Questions[i].CorrectAnswer = &v
		}
	}
}

func decorateViewerMarks(board *LivesBoard, contestants []LiveContestant, marks []AnswerMark, currentQ int, viewerIsOfficial bool) {
	if board == nil {
		return
	}
	byQ := map[int][]AnswerMark{}
	current := map[string]string{}
	for _, m := range marks {
		byQ[m.QuestionNumber] = append(byQ[m.QuestionNumber], m)
		if m.QuestionNumber == currentQ {
			current[m.ContestantUserID] = m.Answer
		}
	}
	names := map[string]string{}
	for _, c := range contestants {
		names[c.UserID] = c.FullName
	}
	correct := ""
	if board.CorrectAnswer != nil {
		correct = *board.CorrectAnswer
	}
	for i := range board.Rows {
		ans, ok := current[board.Rows[i].UserID]
		if !ok {
			continue
		}
		v := ans
		board.Rows[i].Answer = &v
		if correct != "" {
			board.Rows[i].Mismatch = ans != correct
			board.Rows[i].CanReveal = viewerIsOfficial && ans != correct
		}
	}
	for i := range board.Questions {
		qCorrect := ""
		if board.Questions[i].CorrectAnswer != nil {
			qCorrect = *board.Questions[i].CorrectAnswer
		}
		list := byQ[board.Questions[i].QuestionNumber]
		answers := make([]QuestionAnswerMark, 0, len(list))
		for _, m := range list {
			answers = append(answers, QuestionAnswerMark{
				ContestantUserID: m.ContestantUserID,
				FullName:         names[m.ContestantUserID],
				Answer:           m.Answer,
				Mismatch:         qCorrect != "" && m.Answer != qCorrect,
			})
		}
		board.Questions[i].Answers = answers
	}
}

func questionKeysList(keys map[int]string, planned int) []QuestionKey {
	max := planned
	for n := range keys {
		if n > max {
			max = n
		}
	}
	out := make([]QuestionKey, 0, max)
	for n := 1; n <= max; n++ {
		if a, ok := keys[n]; ok {
			out = append(out, QuestionKey{QuestionNumber: n, CorrectAnswer: a})
		}
	}
	return out
}
