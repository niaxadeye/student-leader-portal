package evaluation

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

func (h *Handler) JuryScorecard(w http.ResponseWriter, r *http.Request) {
	contestantID := strings.TrimSpace(r.URL.Query().Get("contestant_user_id"))
	card, err := h.svc.JuryScorecard(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), contestantID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, scorecardJSON(card), nil)
}

func (h *Handler) JurySetScore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PerformanceID string  `json:"performance_id"`
		CriterionID   string  `json:"criterion_id"`
		Score         float64 `json:"score"`
		MutationID    string  `json:"mutation_id"`
		BaseRevision  *int    `json:"base_revision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	res, err := h.svc.JurySetScore(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), ScoreMutation{
		PerformanceID: req.PerformanceID,
		CriterionID:   req.CriterionID,
		Score:         req.Score,
		MutationID:    req.MutationID,
		BaseRevision:  req.BaseRevision,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"criterion_id": res.CriterionID,
		"score":        res.Score,
		"revision":     res.Revision,
		"total":        res.Total,
	}, nil)
}

func (h *Handler) JuryWrongAnswer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContestantUserID string `json:"contestant_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	snap, err := h.svc.JuryWrongAnswer(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.ContestantUserID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func (h *Handler) JuryRestoreLife(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContestantUserID string `json:"contestant_user_id"`
		QuestionNumber   int    `json:"question_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	snap, err := h.svc.JuryRestoreLife(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.ContestantUserID, req.QuestionNumber)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func (h *Handler) JurySetAnswer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContestantUserID string `json:"contestant_user_id"`
		Answer           string `json:"answer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	snap, err := h.svc.JurySetAnswer(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.ContestantUserID, req.Answer)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, liveJSON(snap), nil)
}

func scorecardJSON(c *Scorecard) map[string]any {
	if c == nil {
		return map[string]any{"configured": false, "scoring_ui": scoringUINone, "criteria": []any{}}
	}
	criteria := make([]map[string]any, 0, len(c.Criteria))
	for i := range c.Criteria {
		item := criterionJSON(&c.Criteria[i].Criterion)
		item["score"] = c.Criteria[i].Score
		item["revision"] = c.Criteria[i].Revision
		criteria = append(criteria, item)
	}
	var contestant any
	if c.Contestant != nil {
		contestant = liveContestantJSON(*c.Contestant)
	}
	return map[string]any{
		"configured":     c.Configured,
		"scheme_type":    c.SchemeType,
		"scoring_ui":     c.ScoringUI,
		"editable":       c.Editable,
		"performance_id": c.PerformanceID,
		"contestant":     contestant,
		"criteria":       criteria,
		"filled":         c.Filled,
		"total":          c.Total,
	}
}

func scoreboardJSON(b *Scoreboard) map[string]any {
	if b == nil {
		return map[string]any{"configured": false, "scoring_ui": scoringUINone, "jury": []any{}, "contestants": []any{}, "criteria": []any{}}
	}
	criteria := make([]map[string]any, 0, len(b.Criteria))
	for i := range b.Criteria {
		c := b.Criteria[i]
		criteria = append(criteria, map[string]any{
			"id": c.ID, "title": c.Title, "min_score": c.MinScore, "max_score": c.MaxScore, "weight": c.Weight,
		})
	}
	jury := make([]map[string]any, 0, len(b.Jury))
	for _, j := range b.Jury {
		jury = append(jury, map[string]any{
			"user_id": j.UserID, "login": j.Login, "full_name": j.FullName,
		})
	}
	contestants := make([]map[string]any, 0, len(b.Contestants))
	for i := range b.Contestants {
		c := b.Contestants[i]
		sheets := make([]map[string]any, 0, len(c.Sheets))
		for _, sh := range c.Sheets {
			values := make([]map[string]any, 0, len(sh.Values))
			for _, v := range sh.Values {
				values = append(values, map[string]any{"criterion_id": v.CriterionID, "score": v.Score})
			}
			sheets = append(sheets, map[string]any{
				"jury_user_id": sh.JuryUserID, "filled": sh.Filled, "total": sh.Total, "values": values,
			})
		}
		row := liveContestantJSON(c.LiveContestant)
		row["sheets"] = sheets
		row["average"] = c.Average
		row["sum"] = c.Sum
		row["rank"] = c.Rank
		row["lives"] = c.Lives
		row["eliminated"] = c.Eliminated
		row["eliminated_question"] = c.EliminatedQuestion
		row["numeric_score"] = c.NumericScore
		contestants = append(contestants, row)
	}
	logs := make([]map[string]any, 0, len(b.LifeLogs))
	for _, log := range b.LifeLogs {
		qs := make([]map[string]any, 0, len(log.Questions))
		for _, q := range log.Questions {
			qs = append(qs, questionLogJSON(q))
		}
		rows := make([]map[string]any, 0, len(log.Rows))
		for _, row := range log.Rows {
			rows = append(rows, livesRowJSON(row))
		}
		logs = append(logs, map[string]any{
			"jury_user_id": log.JuryUserID, "official": log.Official,
			"questions": qs, "rows": rows,
		})
	}
	return map[string]any{
		"configured":                 b.Configured,
		"scheme_type":                b.SchemeType,
		"scoring_ui":                 b.ScoringUI,
		"current_contestant_user_id": b.CurrentContestantUserID,
		"criteria":                   criteria,
		"jury":                       jury,
		"contestants":                contestants,
		"starting_lives":             b.StartingLives,
		"operator_user_id":           b.OperatorUserID,
		"current_question_number":    b.CurrentQuestionNumber,
		"question_count":             b.QuestionCount,
		"life_logs":                  logs,
		"min_score":                  b.MinScore,
		"max_score":                  b.MaxScore,
		"combined":                   combinedJSON(b.Combined),
		"can_override":               b.CanOverride,
		"corrections":                correctionsJSON(b.Corrections),
	}
}

func correctionsJSON(list []ScoreCorrection) []map[string]any {
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, map[string]any{
			"id":                 c.ID,
			"kind":               c.Kind,
			"actor_user_id":      c.ActorUserID,
			"actor_name":         c.ActorName,
			"contestant_user_id": c.ContestantUserID,
			"contestant_name":    c.ContestantName,
			"jury_user_id":       c.JuryUserID,
			"jury_name":          c.JuryName,
			"criterion_id":       c.CriterionID,
			"criterion_title":    c.CriterionTitle,
			"old_score":          c.OldScore,
			"new_score":          c.NewScore,
			"reason":             c.Reason,
			"created_at":         c.CreatedAt,
		})
	}
	return out
}

func combinedJSON(c *CombinedRanking) any {
	if c == nil {
		return nil
	}
	rows := make([]map[string]any, 0, len(c.Rows))
	for _, row := range c.Rows {
		rows = append(rows, map[string]any{
			"user_id":      row.UserID,
			"full_name":    row.FullName,
			"main_score":   row.MainScore,
			"main_rank":    row.MainRank,
			"remote_score": row.RemoteScore,
			"remote_rank":  row.RemoteRank,
			"combined":     row.Combined,
			"rank":         row.Rank,
		})
	}
	return map[string]any{
		"remote_challenge_id":    c.RemoteChallengeID,
		"remote_challenge_title": c.RemoteChallengeTitle,
		"main_weight":            c.MainWeight,
		"remote_weight":          c.RemoteWeight,
		"combine_mode":           c.CombineMode,
		"rows":                   rows,
	}
}
