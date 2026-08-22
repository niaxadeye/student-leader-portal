package evaluation

import "testing"

func TestEliminationRanksWaves(t *testing.T) {
	t.Parallel()
	ids := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"}
	states := make([]contestantLifeState, 0, 12)
	for i, id := range ids {
		states = append(states, contestantLifeState{ID: id, Lives: 3, Draw: i + 1})
	}
	q1 := 1
	q2 := 2
	states[0].Lives = 0
	states[0].EliminatedQuestion = &q1
	states[1].Lives = 0
	states[1].EliminatedQuestion = &q1
	states[2].Lives = 0
	states[2].EliminatedQuestion = &q2
	ranks := eliminationRanks(states)
	if ranks["a"] != 12 || ranks["b"] != 12 {
		t.Fatalf("wave 1: a=%d b=%d", ranks["a"], ranks["b"])
	}
	if ranks["c"] != 10 {
		t.Fatalf("wave 2: c=%d", ranks["c"])
	}
	if ranks["d"] != 1 {
		t.Fatalf("remaining top: d=%d", ranks["d"])
	}
}

func TestEliminationRanksSingleThenNext(t *testing.T) {
	t.Parallel()
	q1, q2 := 1, 2
	states := []contestantLifeState{
		{ID: "out1", Lives: 0, EliminatedQuestion: &q1, Draw: 1},
		{ID: "out2", Lives: 0, EliminatedQuestion: &q2, Draw: 2},
		{ID: "ok", Lives: 2, Draw: 3},
	}
	ranks := eliminationRanks(states)
	if ranks["out1"] != 3 || ranks["out2"] != 2 || ranks["ok"] != 1 {
		t.Fatalf("got %+v", ranks)
	}
}

func TestApplyLifeEventsRestore(t *testing.T) {
	t.Parallel()
	contestants := []LiveContestant{{UserID: "u1", FullName: "A"}}
	events := []LifeEvent{
		{ContestantUserID: "u1", QuestionNumber: 1, Delta: -1, Reason: ReasonWrongAnswer, ID: "e1"},
		{ContestantUserID: "u1", QuestionNumber: 2, Delta: -1, Reason: ReasonWrongAnswer, ID: "e2"},
		{ContestantUserID: "u1", QuestionNumber: 3, Delta: -1, Reason: ReasonWrongAnswer, ID: "e3"},
	}
	st := applyLifeEvents(3, contestants, events)
	if st[0].Lives != 0 || st[0].EliminatedQuestion == nil || *st[0].EliminatedQuestion != 3 {
		t.Fatalf("eliminated: %+v", st[0])
	}
	rev := "e2"
	events = append(events, LifeEvent{
		ContestantUserID: "u1", QuestionNumber: 2, Delta: 1, Reason: ReasonRestore, ReversesLifeEventID: &rev, ID: "e4",
	})
	st = applyLifeEvents(3, contestants, events)
	if st[0].Lives != 1 || st[0].EliminatedQuestion != nil {
		t.Fatalf("restored: %+v", st[0])
	}
}

func TestBuildLivesBoardNetLoss(t *testing.T) {
	t.Parallel()
	contestants := []LiveContestant{
		{UserID: "u1", FullName: "Иванов"},
		{UserID: "u2", FullName: "Петров"},
	}
	e1 := "e1"
	events := []LifeEvent{
		{ID: e1, ContestantUserID: "u1", QuestionNumber: 1, Delta: -1, Reason: ReasonWrongAnswer},
		{ID: "e2", ContestantUserID: "u1", QuestionNumber: 1, Delta: 1, Reason: ReasonRestore, ReversesLifeEventID: &e1},
		{ID: "e3", ContestantUserID: "u2", QuestionNumber: 1, Delta: -1, Reason: ReasonWrongAnswer},
	}
	op := "jury"
	board := buildLivesBoard(3, 2, 0, &op, &op, contestants, events)
	if len(board.Questions) != 2 {
		t.Fatalf("questions %d", len(board.Questions))
	}
	if len(board.Questions[0].Losses) != 1 || board.Questions[0].Losses[0].ContestantUserID != "u2" {
		t.Fatalf("q1 losses %+v", board.Questions[0].Losses)
	}
	if board.Rows[0].Lives != 3 || board.Rows[1].Lives != 2 {
		t.Fatalf("lives %+v", board.Rows)
	}
	if len(board.Rows[1].RestoreQuestions) != 1 || board.Rows[1].RestoreQuestions[0] != 1 {
		t.Fatalf("restore %+v", board.Rows[1].RestoreQuestions)
	}
	if board.Rows[1].EliminatedQuestion != nil {
		t.Fatalf("u2 still has lives, question=%v", board.Rows[1].EliminatedQuestion)
	}
	outEvents := append(events,
		LifeEvent{ID: "e4", ContestantUserID: "u2", QuestionNumber: 2, Delta: -1, Reason: ReasonWrongAnswer},
		LifeEvent{ID: "e5", ContestantUserID: "u2", QuestionNumber: 2, Delta: -1, Reason: ReasonWrongAnswer},
	)
	outBoard := buildLivesBoard(3, 2, 0, &op, &op, contestants, outEvents)
	if outBoard.Rows[1].EliminatedQuestion == nil || *outBoard.Rows[1].EliminatedQuestion != 2 {
		t.Fatalf("eliminated question: %+v", outBoard.Rows[1].EliminatedQuestion)
	}
}

func TestBuildLivesBoardPlannedCount(t *testing.T) {
	t.Parallel()
	contestants := []LiveContestant{{UserID: "u1", FullName: "A"}}
	board := buildLivesBoard(3, 1, 4, nil, nil, contestants, nil)
	if len(board.Questions) != 4 {
		t.Fatalf("planned questions %d", len(board.Questions))
	}
}

func TestLifeSyncAction(t *testing.T) {
	t.Parallel()
	if got := lifeSyncAction("YES", "YES", false, 3); got != "" {
		t.Fatalf("match: %q", got)
	}
	if got := lifeSyncAction("NO", "YES", false, 3); got != "miss" {
		t.Fatalf("mismatch: %q", got)
	}
	if got := lifeSyncAction("NO", "YES", true, 2); got != "" {
		t.Fatalf("already open: %q", got)
	}
	if got := lifeSyncAction("YES", "YES", true, 2); got != "restore" {
		t.Fatalf("correct after miss: %q", got)
	}
	if got := lifeSyncAction("NO", "YES", false, 0); got != "" {
		t.Fatalf("already out: %q", got)
	}
	if got := lifeSyncAction("NO", "", false, 3); got != "" {
		t.Fatalf("no key: %q", got)
	}
}

func TestNormalizeAnswer(t *testing.T) {
	t.Parallel()
	yes, err := normalizeAnswer("да")
	if err != nil || yes != AnswerYes {
		t.Fatalf("да: %q %v", yes, err)
	}
	no, err := normalizeAnswer("Нет")
	if err != nil || no != AnswerNo {
		t.Fatalf("нет: %q %v", no, err)
	}
	if _, err := normalizeAnswer("maybe"); err == nil {
		t.Fatal("expected validation")
	}
}
