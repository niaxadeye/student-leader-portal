package challenges

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type fieldReq struct {
	Key         string         `json:"key"`
	Type        string         `json:"type"`
	Label       string         `json:"label"`
	Description *string        `json:"description"`
	HelpText    *string        `json:"help_text"`
	Placeholder *string        `json:"placeholder"`
	Required    bool           `json:"required"`
	Settings    map[string]any `json:"settings"`
	Validation  map[string]any `json:"validation"`
	Visibility  map[string]any `json:"visibility"`
}

func (req fieldReq) toInput() FieldInput {
	return FieldInput{
		Key: req.Key, Type: req.Type, Label: req.Label, Description: req.Description,
		HelpText: req.HelpText, Placeholder: req.Placeholder, Required: req.Required,
		Settings: req.Settings, Validation: req.Validation, Visibility: req.Visibility,
	}
}

func (h *Handler) ListFields(w http.ResponseWriter, r *http.Request) {
	fields, err := h.svc.ListFields(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeFieldList(w, r, fields)
}

func (h *Handler) AddField(w http.ResponseWriter, r *http.Request) {
	var req fieldReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	f, err := h.svc.AddField(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.toInput())
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, FieldMap(f), nil)
}

func (h *Handler) UpdateField(w http.ResponseWriter, r *http.Request) {
	var req fieldReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	err := h.svc.UpdateField(r.Context(), actorOf(r),
		chi.URLParam(r, "challengeId"), chi.URLParam(r, "fieldId"), req.toInput())
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *Handler) DeleteField(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteField(r.Context(), actorOf(r),
		chi.URLParam(r, "challengeId"), chi.URLParam(r, "fieldId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

type reorderReq struct {
	FieldIDs []string `json:"field_ids"`
}

func (h *Handler) ReorderFields(w http.ResponseWriter, r *http.Request) {
	var req reorderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.FieldIDs) == 0 {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", nil)
		return
	}
	err := h.svc.ReorderFields(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"), req.FieldIDs)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func (h *Handler) SchemaPreview(w http.ResponseWriter, r *http.Request) {
	schema, err := h.svc.AdminSchemaPreview(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, schema, nil)
}

// --- Контестант (чтение) ---

func (h *Handler) ContestantList(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ContestantList(r.Context(), actorOf(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeChallengeList(w, r, list)
}

func (h *Handler) ContestantGet(w http.ResponseWriter, r *http.Request) {
	c, fields, err := h.svc.ContestantGet(r.Context(), actorOf(r), chi.URLParam(r, "challengeId"))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	out := challengeJSON(c)
	items := make([]map[string]any, 0, len(fields))
	for i := range fields {
		items = append(items, FieldMap(&fields[i]))
	}
	out["fields"] = items
	httpserver.WriteJSON(w, r, http.StatusOK, out, nil)
}

func writeFieldList(w http.ResponseWriter, r *http.Request, fields []Field) {
	out := make([]map[string]any, 0, len(fields))
	for i := range fields {
		out = append(out, FieldMap(&fields[i]))
	}
	httpserver.WriteJSON(w, r, http.StatusOK, out, map[string]any{"count": len(out)})
}
