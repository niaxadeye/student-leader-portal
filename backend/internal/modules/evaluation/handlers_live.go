package evaluation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type liveCmdReq struct {
	BaseRevision int    `json:"base_revision"`
	ContestantID string `json:"contestant_user_id"`
	PhaseID      string `json:"phase_id"`
	Delta        int    `json:"delta"`
}

func (h *Handler) AdminLive(w http.ResponseWriter, r *http.Request) {
	snap, err := h.svc.Live(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), false)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func (h *Handler) AdminStart(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.Start(r.Context(), a, id, req.BaseRevision)
	})
}

func (h *Handler) AdminPause(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.Pause(r.Context(), a, id, req.BaseRevision)
	})
}

func (h *Handler) AdminFinish(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.Finish(r.Context(), a, id, req.BaseRevision)
	})
}

func (h *Handler) AdminRestart(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.Restart(r.Context(), a, id, req.BaseRevision)
	})
}

func (h *Handler) AdminRestartTimer(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.RestartTimer(r.Context(), a, id, req.BaseRevision)
	})
}

func (h *Handler) AdminCompleteContestant(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.CompleteContestant(r.Context(), a, id, req.BaseRevision)
	})
}

func (h *Handler) AdminEndSpeech(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.EndSpeech(r.Context(), a, id, req.BaseRevision)
	})
}

func (h *Handler) AdminSetDurations(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BaseRevision     int `json:"base_revision"`
		SpeechSeconds    int `json:"speech_seconds"`
		QuestionsSeconds int `json:"questions_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	snap, err := h.svc.SetDurations(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.SpeechSeconds, req.QuestionsSeconds, req.BaseRevision)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func (h *Handler) AdminSetContestant(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.SetContestant(r.Context(), a, id, req.ContestantID, req.BaseRevision)
	})
}

func (h *Handler) AdminSetPhase(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.SetPhase(r.Context(), a, id, req.PhaseID, req.BaseRevision)
	})
}

func (h *Handler) AdminStepQuestion(w http.ResponseWriter, r *http.Request) {
	h.liveCmd(w, r, func(a Actor, id string, req liveCmdReq) (*LiveSnapshot, error) {
		return h.svc.StepQuestion(r.Context(), a, id, req.Delta, req.BaseRevision)
	})
}

func (h *Handler) AdminSetCorrectAnswer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CorrectAnswer string `json:"correct_answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	snap, err := h.svc.SetCorrectAnswer(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.CorrectAnswer)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func (h *Handler) AdminSetQuestionPlan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		QuestionCount int `json:"question_count"`
		Answers       []struct {
			QuestionNumber int    `json:"question_number"`
			CorrectAnswer  string `json:"correct_answer"`
		} `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	items := make([]questionPlanItem, 0, len(req.Answers))
	for _, a := range req.Answers {
		items = append(items, questionPlanItem{Number: a.QuestionNumber, Answer: a.CorrectAnswer})
	}
	snap, err := h.svc.SetQuestionPlan(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.QuestionCount, items)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func (h *Handler) AdminShuffleDraw(w http.ResponseWriter, r *http.Request) {
	snap, err := h.svc.ShuffleDraw(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

type reorderDrawReq struct {
	ContestantUserIDs []string `json:"contestant_user_ids"`
}

func (h *Handler) AdminReorderDraw(w http.ResponseWriter, r *http.Request) {
	var req reorderDrawReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	snap, err := h.svc.ReorderDraw(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.ContestantUserIDs)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func (h *Handler) ContestantDraw(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.ContestantDraw(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, contestantDrawJSON(d), nil)
}

func (h *Handler) ContestantContestDraws(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ContestantContestDraws(r.Context(), actorOf(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, m := range list {
		items = append(items, map[string]any{
			"challenge_id": m.ChallengeID, "my_draw_number": m.MyDrawNumber, "total": m.Total,
		})
	}
	httpserver.WriteJSON(w, r, http.StatusOK, items, nil)
}

func contestantDrawJSON(d *ContestantDraw) map[string]any {
	if d == nil {
		return map[string]any{"drawn": false, "my_draw_number": nil, "total": 0, "order": []any{}}
	}
	order := make([]map[string]any, 0, len(d.Order))
	for _, e := range d.Order {
		order = append(order, map[string]any{
			"draw_number": e.DrawNumber, "full_name": e.FullName,
			"organization": e.Organization, "is_me": e.IsMe,
		})
	}
	return map[string]any{
		"drawn": d.Drawn, "my_draw_number": d.MyDrawNumber, "total": d.Total, "order": order,
	}
}

func (h *Handler) liveCmd(w http.ResponseWriter, r *http.Request, fn func(Actor, string, liveCmdReq) (*LiveSnapshot, error)) {
	var req liveCmdReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	snap, err := fn(actorOf(r), chi.URLParam(r, "challengeId"), req)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func (h *Handler) JuryLive(w http.ResponseWriter, r *http.Request) {
	snap, err := h.svc.JuryLive(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func (h *Handler) AdminLiveStream(w http.ResponseWriter, r *http.Request) {
	h.stream(w, r, true)
}

func (h *Handler) JuryLiveStream(w http.ResponseWriter, r *http.Request) {
	h.stream(w, r, false)
}

func (h *Handler) stream(w http.ResponseWriter, r *http.Request, admin bool) {
	id := chi.URLParam(r, "challengeId")
	a := actorOf(r)
	var err error
	if admin {
		_, err = h.svc.Live(r.Context(), a, id, false)
	} else {
		_, err = h.svc.JuryLive(r.Context(), a, id)
	}
	if err != nil {
		writeErr(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	events, cancel := h.svc.Subscribe(id)
	defer cancel()
	h.svc.Touch(id, a.UserID)
	_, _ = fmt.Fprintf(w, "event: hello\ndata: {\"ok\":true}\n\n")
	_ = rc.Flush()
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			h.svc.Touch(id, a.UserID)
			_, _ = fmt.Fprintf(w, ": keepalive\n\n")
			if err := rc.Flush(); err != nil {
				return
			}
		case ev, ok := <-events:
			if !ok {
				return
			}
			body, _ := json.Marshal(ev)
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, body)
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}

func liveJSON(s *LiveSnapshot) map[string]any {
	if s == nil {
		return map[string]any{}
	}
	contestants := make([]map[string]any, 0, len(s.Contestants))
	byUser := livesRowByUser(s.Lives)
	for _, c := range s.Contestants {
		row := liveContestantJSON(c)
		if st, ok := byUser[c.UserID]; ok {
			row["lives"] = st.Lives
			row["eliminated"] = st.Eliminated
			row["eliminated_question"] = st.EliminatedQuestion
			row["rank"] = st.Rank
			row["restore_questions"] = st.RestoreQuestions
			row["answer"] = st.Answer
			row["mismatch"] = st.Mismatch
			row["can_reveal"] = st.CanReveal
		}
		contestants = append(contestants, row)
	}
	phases := make([]map[string]any, 0, len(s.Phases))
	for _, p := range s.Phases {
		phases = append(phases, map[string]any{
			"id": p.ID, "title": p.Title, "duration_seconds": p.DurationSeconds,
			"scoring_allowed": p.ScoringAllowed, "maps_to_state": p.MapsToState, "sort_order": p.SortOrder,
		})
	}
	var perf any
	if s.Performance != nil {
		perf = map[string]any{
			"id": s.Performance.ID, "contestant_user_id": s.Performance.ContestantUserID,
			"status": s.Performance.Status, "sequence_number": s.Performance.SequenceNumber,
			"started_at": s.Performance.StartedAt, "finished_at": s.Performance.FinishedAt,
		}
	}
	var current any
	if s.Current != nil {
		current = liveContestantJSON(*s.Current)
	}
	sess := s.Session
	q := s.CurrentQuestionNumber
	if q < 1 {
		q = sess.CurrentQuestionNumber
	}
	if q < 1 {
		q = 1
	}
	out := map[string]any{
		"challenge_id": s.ChallengeID, "contest_id": s.ContestID, "challenge_title": s.ChallengeTitle,
		"session_revision": sess.Revision, "state": sess.State,
		"current_contestant_user_id": sess.CurrentContestantUserID,
		"current_performance_id":     sess.CurrentPerformanceID,
		"current_phase_id":           sess.CurrentPhaseID,
		"started_at":                 sess.StartedAt,
		"finished_at":                sess.FinishedAt,
		"phase_started_at":           sess.PhaseStartedAt,
		"phase_duration_seconds":     sess.PhaseDurationSeconds,
		"paused_at":                  sess.PausedAt,
		"accumulated_pause_seconds":  sess.AccumulatedPauseSeconds,
		"timer_remaining_seconds":    s.TimerRemaining,
		"jury_online":                s.JuryOnline,
		"server_time":                s.ServerTime,
		"current":                    current,
		"performance":                perf,
		"phases":                     phases,
		"contestants":                contestants,
		"drawn":                      s.Drawn,
		"draw_locked":                drawLocked(sess.State),
		"scheme_type":                s.SchemeType,
		"starting_lives":             s.StartingLives,
		"current_question_number":    q,
		"operator_user_id":           s.OperatorUserID,
		"question_count":             s.QuestionCount,
		"lives":                      livesJSON(s.Lives),
	}
	if s.ShowAnswerKey {
		out["correct_answer"] = s.CorrectAnswer
		keys := make([]map[string]any, 0, len(s.QuestionKeys))
		for _, k := range s.QuestionKeys {
			keys = append(keys, map[string]any{
				"question_number": k.QuestionNumber,
				"correct_answer":  k.CorrectAnswer,
			})
		}
		out["question_keys"] = keys
	}
	return out
}

func livesJSON(b *LivesBoard) any {
	if b == nil {
		return nil
	}
	questions := make([]map[string]any, 0, len(b.Questions))
	for _, q := range b.Questions {
		questions = append(questions, questionLogJSON(q))
	}
	rows := make([]map[string]any, 0, len(b.Rows))
	for _, row := range b.Rows {
		rows = append(rows, livesRowJSON(row))
	}
	return map[string]any{
		"starting_lives":   b.StartingLives,
		"current_question": b.CurrentQuestion,
		"operator_user_id": b.OperatorUserID,
		"viewer_user_id":   b.ViewerUserID,
		"official":         b.Official,
		"correct_answer":   b.CorrectAnswer,
		"questions":        questions,
		"rows":             rows,
	}
}

func questionLogJSON(q QuestionLogEntry) map[string]any {
	losses := make([]map[string]any, 0, len(q.Losses))
	for _, l := range q.Losses {
		losses = append(losses, map[string]any{
			"contestant_user_id": l.ContestantUserID,
			"full_name":          l.FullName,
			"organization":       l.Organization,
			"avatar_url":         l.AvatarURL,
		})
	}
	answers := make([]map[string]any, 0, len(q.Answers))
	for _, a := range q.Answers {
		answers = append(answers, map[string]any{
			"contestant_user_id": a.ContestantUserID,
			"full_name":          a.FullName,
			"answer":             a.Answer,
			"mismatch":           a.Mismatch,
		})
	}
	return map[string]any{
		"question_number": q.QuestionNumber,
		"current":         q.Current,
		"correct_answer":  q.CorrectAnswer,
		"losses":          losses,
		"answers":         answers,
	}
}

func livesRowJSON(row LivesRow) map[string]any {
	return map[string]any{
		"user_id":             row.UserID,
		"lives":               row.Lives,
		"eliminated":          row.Eliminated,
		"eliminated_question": row.EliminatedQuestion,
		"rank":                row.Rank,
		"restore_questions":   row.RestoreQuestions,
		"answer":              row.Answer,
		"mismatch":            row.Mismatch,
		"can_reveal":          row.CanReveal,
	}
}

func liveContestantJSON(c LiveContestant) map[string]any {
	return map[string]any{
		"user_id": c.UserID, "login": c.Login, "full_name": c.FullName,
		"organization": c.Organization, "performance_id": c.PerformanceID,
		"performance_status": c.PerformanceStatus, "draw_number": c.DrawNumber,
		"speech_duration_seconds": c.SpeechDurationSeconds,
		"avatar_url":              c.AvatarURL,
	}
}
