package evaluation

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func actorOf(r *http.Request) Actor {
	return actorFromPrincipal(middleware.PrincipalFrom(r.Context()))
}

func actorFromPrincipal(p *middleware.Principal) Actor {
	if p == nil {
		return Actor{}
	}
	return Actor{UserID: p.UserID, IsSuper: p.Role == "SUPER_ADMIN", IsMega: p.Role == "MEGA_ADMIN"}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "challengeId")
	scheme, err := h.svc.Get(r.Context(), actorOf(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	h.writeEvaluation(w, r, id, scheme)
}

func (h *Handler) AdminScoreboard(w http.ResponseWriter, r *http.Request) {
	board, err := h.svc.AdminScoreboard(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, scoreboardJSON(board), nil)
}

func (h *Handler) SetNumericResult(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContestantUserID string   `json:"contestant_user_id"`
		Score            *float64 `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	board, err := h.svc.SetNumericResult(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.ContestantUserID, req.Score)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, scoreboardJSON(board), nil)
}

func (h *Handler) OverrideScore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kind             string   `json:"kind"`
		ContestantUserID string   `json:"contestant_user_id"`
		JuryUserID       string   `json:"jury_user_id"`
		CriterionID      string   `json:"criterion_id"`
		Score            *float64 `json:"score"`
		Reason           string   `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	board, err := h.svc.OverrideScore(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), ScoreOverrideInput{
		Kind:             req.Kind,
		ContestantUserID: req.ContestantUserID,
		JuryUserID:       req.JuryUserID,
		CriterionID:      req.CriterionID,
		Score:            req.Score,
		Reason:           req.Reason,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, scoreboardJSON(board), nil)
}

func (h *Handler) Put(w http.ResponseWriter, r *http.Request) {
	var req schemeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	id := chi.URLParam(r, "challengeId")
	scheme, err := h.svc.Put(r.Context(), actorOf(r), id, req.toInput())
	if err != nil {
		writeErr(w, r, err)
		return
	}
	h.writeEvaluation(w, r, id, scheme)
}

func (h *Handler) writeEvaluation(w http.ResponseWriter, r *http.Request, challengeID string, scheme *Scheme) {
	a := actorOf(r)
	jury, err := h.svc.ContestJury(r.Context(), a, challengeID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	contestJury, err := h.svc.ContestRoleJury(r.Context(), a, challengeID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	remoteJury, err := h.svc.ContestRemoteJury(r.Context(), a, challengeID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	scope, err := h.svc.JuryScope(r.Context(), a, challengeID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	stage, err := h.svc.StageLinkView(r.Context(), a, challengeID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, evaluationJSON(scheme, jury, contestJury, remoteJury, scope, stage), nil)
}

func (h *Handler) PutStageLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RemoteChallengeID *string `json:"remote_challenge_id"`
		MainWeight        float64 `json:"main_weight"`
		RemoteWeight      float64 `json:"remote_weight"`
		CombineMode       string  `json:"combine_mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	id := chi.URLParam(r, "challengeId")
	if _, err := h.svc.PutStageLink(r.Context(), actorOf(r), id, StageLinkInput{
		RemoteChallengeID: req.RemoteChallengeID,
		MainWeight:        req.MainWeight,
		RemoteWeight:      req.RemoteWeight,
		CombineMode:       req.CombineMode,
	}); err != nil {
		writeErr(w, r, err)
		return
	}
	scheme, err := h.svc.Get(r.Context(), actorOf(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	h.writeEvaluation(w, r, id, scheme)
}

func (h *Handler) ResetResults(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	if err := h.svc.ResetResults(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.Password); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true}, nil)
}

func (h *Handler) ReplaceJury(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password    string   `json:"password"`
		JuryUserIDs []string `json:"jury_user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	if err := h.svc.ReplaceJury(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.Password, req.JuryUserIDs); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true}, nil)
}

func (h *Handler) PutRemoteJury(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JuryUserIDs []string `json:"jury_user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	id := chi.URLParam(r, "challengeId")
	if err := h.svc.SetRemoteJury(r.Context(), actorOf(r), id, req.JuryUserIDs); err != nil {
		writeErr(w, r, err)
		return
	}
	scheme, err := h.svc.Get(r.Context(), actorOf(r), id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	h.writeEvaluation(w, r, id, scheme)
}

func (h *Handler) SearchRemoteJury(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.SearchRemoteJury(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, j := range list {
		items = append(items, map[string]any{
			"user_id": j.UserID, "login": j.Login, "full_name": j.FullName,
		})
	}
	httpserver.WriteJSON(w, r, http.StatusOK, items, map[string]any{"count": len(items)})
}

func (h *Handler) AddCriterion(w http.ResponseWriter, r *http.Request) {
	var req criterionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	c, err := h.svc.AddCriterion(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.toInput())
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, criterionJSON(c), nil)
}

func (h *Handler) UpdateCriterion(w http.ResponseWriter, r *http.Request) {
	var req criterionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	c, err := h.svc.UpdateCriterion(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), chi.URLParam(r, "criterionId"), req.toInput())
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, criterionJSON(c), nil)
}

func (h *Handler) DeleteCriterion(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteCriterion(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), chi.URLParam(r, "criterionId")); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true}, nil)
}

func (h *Handler) ReorderCriteria(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	if err := h.svc.ReorderCriteria(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.IDs); err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true}, nil)
}

func (h *Handler) JuryContests(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFrom(r.Context())
	if p == nil {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Требуется авторизация", nil)
		return
	}
	list, err := h.svc.JuryContests(r.Context(), p.UserID)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		chs := make([]map[string]any, 0, len(list[i].Challenges))
		for j := range list[i].Challenges {
			ch := list[i].Challenges[j]
			chs = append(chs, map[string]any{
				"id": ch.ID, "title": ch.Title, "slug": ch.Slug,
				"status": ch.Status, "has_scheme": ch.HasScheme, "scheme_type": ch.SchemeType,
			})
		}
		out = append(out, map[string]any{
			"id": list[i].ID, "name": list[i].Name, "slug": list[i].Slug, "challenges": chs,
		})
	}
	httpserver.WriteJSON(w, r, http.StatusOK, out, map[string]any{"count": len(out)})
}

type schemeReq struct {
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	ScoringUnit      string   `json:"scoring_unit"`
	MinScore         *float64 `json:"min_score"`
	MaxScore         *float64 `json:"max_score"`
	CorridorMode     string   `json:"corridor_mode"`
	ResultVisibility string   `json:"result_visibility"`
	EditPolicy       string   `json:"edit_policy"`
	StartingLives    *int     `json:"starting_lives"`
	OperatorUserID   *string  `json:"operator_user_id"`
}

func (req schemeReq) toInput() SchemeInput {
	return SchemeInput{
		Name: req.Name, Type: req.Type, ScoringUnit: req.ScoringUnit,
		MinScore: req.MinScore, MaxScore: req.MaxScore,
		CorridorMode: req.CorridorMode, ResultVisibility: req.ResultVisibility, EditPolicy: req.EditPolicy,
		StartingLives: req.StartingLives, OperatorUserID: req.OperatorUserID,
	}
}

type bandReq struct {
	MinScore    float64 `json:"min_score"`
	MaxScore    float64 `json:"max_score"`
	Description string  `json:"description"`
}

type criterionReq struct {
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	MinScore    float64   `json:"min_score"`
	MaxScore    float64   `json:"max_score"`
	Weight      float64   `json:"weight"`
	IsRequired  *bool     `json:"is_required"`
	Bands       []bandReq `json:"bands"`
}

func (req criterionReq) toInput() CriterionInput {
	bands := make([]ScaleBandInput, 0, len(req.Bands))
	for _, b := range req.Bands {
		bands = append(bands, ScaleBandInput{MinScore: b.MinScore, MaxScore: b.MaxScore, Description: b.Description})
	}
	return CriterionInput{
		Title: req.Title, Description: req.Description,
		MinScore: req.MinScore, MaxScore: req.MaxScore, Weight: req.Weight,
		IsRequired: req.IsRequired, Bands: bands,
	}
}

func evaluationJSON(s *Scheme, jury, contestJury, remoteJury []JuryPerson, scope string, stage *StageLinkView) map[string]any {
	out, _ := schemeJSON(s).(map[string]any)
	if out == nil {
		out = map[string]any{"configured": false, "scheme": nil}
	}
	items := make([]map[string]any, 0, len(jury))
	for _, j := range jury {
		items = append(items, map[string]any{
			"user_id": j.UserID, "login": j.Login, "full_name": j.FullName,
		})
	}
	contestItems := make([]map[string]any, 0, len(contestJury))
	for _, j := range contestJury {
		contestItems = append(contestItems, map[string]any{
			"user_id": j.UserID, "login": j.Login, "full_name": j.FullName,
		})
	}
	remoteItems := make([]map[string]any, 0, len(remoteJury))
	for _, j := range remoteJury {
		remoteItems = append(remoteItems, map[string]any{
			"user_id": j.UserID, "login": j.Login, "full_name": j.FullName,
		})
	}
	if scope == "" {
		scope = "CONTEST"
	}
	out["jury"] = items
	out["contest_jury"] = contestItems
	out["remote_jury"] = remoteItems
	out["jury_scope"] = scope
	out["stage_link"] = nil
	out["linked_from"] = nil
	opts := make([]map[string]any, 0)
	if stage != nil {
		for _, o := range stage.RemoteOptions {
			opts = append(opts, map[string]any{"id": o.ID, "title": o.Title})
		}
		if stage.Link != nil && stage.LinkedFrom == nil {
			out["stage_link"] = map[string]any{
				"remote_challenge_id":    stage.Link.RemoteChallengeID,
				"remote_challenge_title": stage.Link.RemoteTitle,
				"main_weight":            stage.Link.MainWeight,
				"remote_weight":          stage.Link.RemoteWeight,
				"combine_mode":           stage.Link.CombineMode,
			}
		}
		if stage.LinkedFrom != nil {
			out["linked_from"] = map[string]any{
				"id": stage.LinkedFrom.ID, "title": stage.LinkedFrom.Title,
			}
		}
	}
	out["remote_stage_options"] = opts
	return out
}

func schemeJSON(s *Scheme) any {
	if s == nil {
		return map[string]any{"configured": false, "scheme": nil}
	}
	criteria := make([]map[string]any, 0, len(s.Criteria))
	for i := range s.Criteria {
		criteria = append(criteria, criterionJSON(&s.Criteria[i]))
	}
	op := s.OperatorUserID
	if op == nil {
		if id := operatorIDOf(s); id != "" {
			op = &id
		}
	}
	return map[string]any{
		"configured": true,
		"scheme": map[string]any{
			"id": s.ID, "challenge_id": s.ChallengeID, "contest_id": s.ContestID,
			"name": s.Name, "type": s.Type, "scoring_unit": s.ScoringUnit,
			"min_score": s.MinScore, "max_score": s.MaxScore,
			"corridor_mode": s.CorridorMode, "result_visibility": s.ResultVisibility,
			"edit_policy": s.EditPolicy, "active": s.Active,
			"starting_lives":   startingLivesOf(s),
			"operator_user_id": op,
			"settings":         schemeSettingsJSON(s),
			"created_at":       s.CreatedAt, "updated_at": s.UpdatedAt,
			"criteria": criteria,
		},
	}
}

func startingLivesOf(s *Scheme) *int {
	if s == nil || s.Type != TypeEliminationLives {
		return nil
	}
	lives := DefaultStartingLives
	var parsed struct {
		StartingLives *int `json:"starting_lives"`
	}
	if len(s.SettingsJSON) > 0 && json.Unmarshal(s.SettingsJSON, &parsed) == nil && parsed.StartingLives != nil {
		lives = *parsed.StartingLives
	}
	return &lives
}

func schemeSettingsJSON(s *Scheme) any {
	if s == nil || len(s.SettingsJSON) == 0 {
		return map[string]any{}
	}
	var raw any
	if json.Unmarshal(s.SettingsJSON, &raw) != nil || raw == nil {
		return map[string]any{}
	}
	return raw
}

func criterionJSON(c *Criterion) map[string]any {
	bands := make([]map[string]any, 0, len(c.Bands))
	for _, b := range c.Bands {
		bands = append(bands, map[string]any{
			"id": b.ID, "min_score": b.MinScore, "max_score": b.MaxScore,
			"description": b.Description, "sort_order": b.SortOrder,
		})
	}
	return map[string]any{
		"id": c.ID, "title": c.Title, "description": c.Description,
		"min_score": c.MinScore, "max_score": c.MaxScore, "weight": c.Weight,
		"is_required": c.IsRequired, "sort_order": c.SortOrder, "bands": bands,
	}
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	var scoreConflict *ScoreRevisionConflict
	switch {
	case errors.As(err, &scoreConflict):
		httpserver.WriteError(w, r, http.StatusConflict, "EVALUATION_REVISION_CONFLICT", "Оценка была изменена на другом устройстве", map[string]any{
			"current_score": scoreConflict.Score, "current_revision": scoreConflict.Revision,
		})
	case errors.Is(err, ErrDisabled), errors.Is(err, ErrNotFound), errors.Is(err, ErrChallenge):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Не найдено", nil)
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrNotAssigned):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Нет доступа", nil)
	case errors.Is(err, ErrRevision):
		httpserver.WriteError(w, r, http.StatusConflict, "EVALUATION_REVISION_CONFLICT", "Состояние сессии изменилось, обновите экран", nil)
	case errors.Is(err, ErrSchemeLocked):
		httpserver.WriteError(w, r, http.StatusConflict, "EVALUATION_SCHEME_LOCKED", "Схему нельзя менять после старта live-сессии", nil)
	case errors.Is(err, ErrScoringClosed):
		httpserver.WriteError(w, r, http.StatusConflict, "EVALUATION_SCORING_CLOSED", "Оценивание закрыто", nil)
	case errors.Is(err, ErrLifeGone):
		httpserver.WriteError(w, r, http.StatusConflict, "EVALUATION_LIFE_ELIMINATED", "Конкурсант уже выбыл", nil)
	case errors.Is(err, ErrLifeDuplicate):
		httpserver.WriteError(w, r, http.StatusConflict, "EVALUATION_LIFE_ALREADY_MARKED", "Ошибка на этом вопросе уже отмечена", nil)
	case errors.Is(err, ErrWrongPassword):
		httpserver.WriteError(w, r, http.StatusBadRequest, "AUTH_WRONG_PASSWORD", "Неверный пароль", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Проверьте заполнение полей", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
