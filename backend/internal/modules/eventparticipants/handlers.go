package eventparticipants

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
	"github.com/eazytech/student-leader-cabinet/internal/platform/security"
)

type CookieConfig struct {
	Name     string
	Domain   string
	Path     string
	Secure   bool
	SameSite http.SameSite
}

type Handler struct {
	svc     *Service
	cookie  CookieConfig
	limiter RateLimiter
}

func NewHandler(svc *Service, cookie CookieConfig, limiter RateLimiter) *Handler {
	return &Handler{svc: svc, cookie: cookie, limiter: limiter}
}

func staffActor(r *http.Request) Actor {
	p := middleware.PrincipalFrom(r.Context())
	if p == nil {
		return Actor{}
	}
	return Actor{UserID: p.UserID, IsMega: p.Role == "MEGA_ADMIN"}
}

func participantJSON(p *Participant) map[string]any {
	return map[string]any{
		"id": p.ID, "event_id": p.ContestID, "full_name": p.FullName,
		"birth_date":        p.BirthDate.Format("2006-01-02"),
		"union_card_number": p.UnionCardNumber, "sks_barcode": p.SKSBarcode,
		"vk_url": p.VKURL, "telegram_url": p.TelegramURL,
		"direction_id": p.DirectionID, "direction_name": p.DirectionName,
		"status": p.Status, "created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
		"archived_at": p.ArchivedAt,
	}
}

func (h *Handler) AdminList(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	result, err := h.svc.List(r.Context(), staffActor(r), chi.URLParam(r, "contestId"), ListFilter{
		Search: r.URL.Query().Get("search"), Status: r.URL.Query().Get("status"),
		DirectionID: r.URL.Query().Get("direction_id"),
		Limit:       limit, Offset: offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	data := make([]map[string]any, 0, len(result.Participants))
	for i := range result.Participants {
		data = append(data, participantJSON(&result.Participants[i]))
	}
	httpserver.WriteJSON(w, r, http.StatusOK, data, map[string]any{
		"total": result.Total, "limit": result.Limit, "offset": result.Offset,
	})
}

func (h *Handler) AdminGet(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), staffActor(r), chi.URLParam(r, "contestId"), chi.URLParam(r, "participantId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, participantJSON(p), nil)
}

type participantRequest struct {
	FullName        string  `json:"full_name"`
	BirthDate       string  `json:"birth_date"`
	UnionCardNumber *string `json:"union_card_number"`
	SKSBarcode      *string `json:"sks_barcode"`
	VKURL           *string `json:"vk_url"`
	TelegramURL     *string `json:"telegram_url"`
	DirectionID     *string `json:"direction_id"`
}

func parseParticipantRequest(req participantRequest) (CreateInput, error) {
	birthDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return CreateInput{}, ErrValidation
	}
	return CreateInput{
		FullName: req.FullName, BirthDate: birthDate,
		UnionCardNumber: req.UnionCardNumber, SKSBarcode: req.SKSBarcode,
		VKURL: req.VKURL, TelegramURL: req.TelegramURL,
		DirectionID: req.DirectionID,
	}, nil
}

func (h *Handler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var req participantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	input, err := parseParticipantRequest(req)
	if err != nil {
		writeError(w, r, err)
		return
	}
	p, err := h.svc.Create(r.Context(), staffActor(r), chi.URLParam(r, "contestId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, participantJSON(p), nil)
}

func (h *Handler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	var req participantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	input, err := parseParticipantRequest(req)
	if err != nil {
		writeError(w, r, err)
		return
	}
	p, err := h.svc.Update(r.Context(), staffActor(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "participantId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, participantJSON(p), nil)
}

const maxParticipantImportBytes = 16 << 20

func (h *Handler) AdminImport(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxParticipantImportBytes)
	filename := "event-participants.csv"
	var reader io.Reader = r.Body
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		if err := r.ParseMultipartForm(maxParticipantImportBytes); err != nil {
			writeError(w, r, ErrValidation)
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, r, ErrValidation)
			return
		}
		defer file.Close()
		filename, reader = header.Filename, file
	}
	records, err := ParseImportFile(filename, reader)
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := h.svc.Import(r.Context(), staffActor(r), chi.URLParam(r, "contestId"), records)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, result, map[string]any{
		"added": result.Added, "updated": result.Updated,
		"errors": result.Errors, "duplicates": result.Duplicates,
	})
}

func (h *Handler) AdminExport(w http.ResponseWriter, r *http.Request) {
	file, err := h.svc.Export(r.Context(), staffActor(r), chi.URLParam(r, "contestId"), r.URL.Query().Get("format"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Data)
}

func (h *Handler) status(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := h.svc.SetStatus(r.Context(), staffActor(r), chi.URLParam(r, "contestId"),
			chi.URLParam(r, "participantId"), target)
		if err != nil {
			writeError(w, r, err)
			return
		}
		httpserver.WriteJSON(w, r, http.StatusOK, participantJSON(p), nil)
	}
}

func (h *Handler) AdminBlock() http.HandlerFunc   { return h.status(StatusBlocked) }
func (h *Handler) AdminUnblock() http.HandlerFunc { return h.status(StatusActive) }
func (h *Handler) AdminArchive() http.HandlerFunc { return h.status(StatusArchived) }

func (h *Handler) AdminListDirections(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListDirections(r.Context(), staffActor(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, list, map[string]any{"count": len(list)})
}

type directionRequest struct {
	Name string `json:"name"`
}

func (h *Handler) AdminCreateDirection(w http.ResponseWriter, r *http.Request) {
	var req directionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	direction, err := h.svc.CreateDirection(r.Context(), staffActor(r), chi.URLParam(r, "contestId"), req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, direction, nil)
}

func (h *Handler) AdminUpdateDirection(w http.ResponseWriter, r *http.Request) {
	var req directionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	direction, err := h.svc.UpdateDirection(r.Context(), staffActor(r),
		chi.URLParam(r, "contestId"), chi.URLParam(r, "directionId"), req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, direction, nil)
}

func (h *Handler) AdminDeleteDirection(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteDirection(r.Context(), staffActor(r),
		chi.URLParam(r, "contestId"), chi.URLParam(r, "directionId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

type loginByNameRequest struct {
	FullName  string `json:"full_name"`
	BirthDate string `json:"birth_date"`
}

type loginIdentifierRequest struct {
	Value           string `json:"value"`
	UnionCardNumber string `json:"union_card_number"`
	SKSBarcode      string `json:"sks_barcode"`
}

func (h *Handler) LoginByName(w http.ResponseWriter, r *http.Request) {
	client := requestClientInfo(r)
	if !h.allowLogin(r, client) {
		writeError(w, r, ErrRateLimited)
		return
	}
	var req loginByNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrInvalidCredentials)
		return
	}
	birthDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		writeError(w, r, ErrInvalidCredentials)
		return
	}
	result, err := h.svc.LoginByName(r.Context(), chi.URLParam(r, "eventSlug"), req.FullName,
		birthDate, client)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.writeLoginResult(w, r, result)
}

func (h *Handler) LoginByUnionCard(w http.ResponseWriter, r *http.Request) {
	h.loginByIdentifier(w, r, "union_card_number", h.svc.LoginByUnionCard)
}

func (h *Handler) LoginBySKSBarcode(w http.ResponseWriter, r *http.Request) {
	h.loginByIdentifier(w, r, "sks_barcode", h.svc.LoginBySKSBarcode)
}

type loginIdentifierFunc func(context.Context, string, string, ClientInfo) (*SessionResult, error)

func (h *Handler) loginByIdentifier(w http.ResponseWriter, r *http.Request, field string, login loginIdentifierFunc) {
	client := requestClientInfo(r)
	if !h.allowLogin(r, client) {
		writeError(w, r, ErrRateLimited)
		return
	}
	var req loginIdentifierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, ErrInvalidCredentials)
		return
	}
	value := req.Value
	if field == "union_card_number" && value == "" {
		value = req.UnionCardNumber
	}
	if field == "sks_barcode" && value == "" {
		value = req.SKSBarcode
	}
	result, err := login(r.Context(), chi.URLParam(r, "eventSlug"), value, client)
	if err != nil {
		writeError(w, r, err)
		return
	}
	h.writeLoginResult(w, r, result)
}

func (h *Handler) writeLoginResult(w http.ResponseWriter, r *http.Request, result *SessionResult) {
	http.SetCookie(w, &http.Cookie{
		Name: h.cookie.Name, Value: result.Token, Path: h.cookie.Path, Domain: h.cookie.Domain,
		Expires: result.ExpiresAt, HttpOnly: true, Secure: h.cookie.Secure, SameSite: h.cookie.SameSite,
	})
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"participant": participantJSON(&result.Participant),
		"event": map[string]any{"id": result.Event.ID, "slug": result.Event.Slug, "name": result.Event.Name,
			"timezone": result.Event.Timezone},
		"expires_at": result.ExpiresAt,
	}, nil)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	p := PrincipalFrom(r.Context())
	if p == nil {
		writeError(w, r, ErrSessionExpired)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]any{
		"participant": participantJSON(&p.Participant),
		"event": map[string]any{"id": p.Event.ID, "slug": p.Event.Slug, "name": p.Event.Name,
			"timezone": p.Event.Timezone},
	}, nil)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := readCookie(r, h.cookie.Name)
	if err := h.svc.Logout(r.Context(), token); err != nil {
		writeError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: h.cookie.Name, Value: "", Path: h.cookie.Path, Domain: h.cookie.Domain,
		Expires: time.Unix(0, 0), MaxAge: -1, HttpOnly: true,
		Secure: h.cookie.Secure, SameSite: h.cookie.SameSite,
	})
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"}, nil)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Участник не найден", nil)
	case errors.Is(err, ErrForbidden):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав", nil)
	case errors.Is(err, ErrIdentifierTaken):
		httpserver.WriteError(w, r, http.StatusConflict, "PARTICIPANT_IDENTIFIER_TAKEN", "Идентификатор уже используется", nil)
	case errors.Is(err, ErrAmbiguousIdentity):
		httpserver.WriteError(w, r, http.StatusConflict, "PARTICIPANT_IDENTITY_AMBIGUOUS",
			"Найдено несколько совпадений. Используйте номер профсоюзного билета или barcode СКС", nil)
	case errors.Is(err, ErrSocialUnavailable):
		httpserver.WriteError(w, r, http.StatusServiceUnavailable, "SOCIAL_AUTH_UNAVAILABLE",
			"Вход через соцсеть не настроен", nil)
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrEventUnavailable):
		httpserver.WriteError(w, r, http.StatusUnauthorized, "PARTICIPANT_AUTH_FAILED", "Не удалось войти по указанным данным", nil)
	case errors.Is(err, ErrSessionExpired):
		httpserver.WriteError(w, r, http.StatusUnauthorized, "PARTICIPANT_SESSION_EXPIRED", "Сессия участника завершена", nil)
	case errors.Is(err, ErrRateLimited):
		httpserver.WriteError(w, r, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Слишком много попыток, попробуйте позже", nil)
	case errors.Is(err, ErrDirectionNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "DIRECTION_NOT_FOUND", "Направление не найдено", nil)
	case errors.Is(err, ErrDirectionTaken):
		httpserver.WriteError(w, r, http.StatusConflict, "DIRECTION_NAME_TAKEN", "Направление с таким названием уже есть", nil)
	case errors.Is(err, ErrDirectionInUse):
		httpserver.WriteError(w, r, http.StatusConflict, "DIRECTION_IN_USE",
			"Нельзя удалить направление: оно назначено участникам или лекциям", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Проверьте заполнение полей", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}

func (h *Handler) allowLogin(r *http.Request, client ClientInfo) bool {
	if h.limiter == nil {
		return true
	}
	key := security.HashToken(chi.URLParam(r, "eventSlug") + "|" + client.IP + "|" + client.UserAgent)
	return h.limiter.Allow(key)
}

func readCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func requestClientInfo(r *http.Request) ClientInfo {
	ip := r.Header.Get("X-Forwarded-For")
	if comma := strings.IndexByte(ip, ','); comma >= 0 {
		ip = ip[:comma]
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = r.RemoteAddr
		}
	}
	return ClientInfo{UserAgent: r.UserAgent(), IP: ip}
}
