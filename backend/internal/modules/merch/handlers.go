package merch

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/eazytech/student-leader-cabinet/internal/middleware"
	"github.com/eazytech/student-leader-cabinet/internal/modules/eventparticipants"
	"github.com/eazytech/student-leader-cabinet/internal/platform/httpserver"
)

type Handler struct {
	svc            *Service
	store          FileStore
	maxUploadBytes int64
}

func NewHandler(service *Service, store FileStore, maxImageBytes int64) *Handler {
	if maxImageBytes <= 0 {
		maxImageBytes = 20 << 20
	}
	return &Handler{svc: service, store: store, maxUploadBytes: maxImageBytes + (1 << 20)}
}

func actorFrom(r *http.Request) Actor {
	principal := middleware.PrincipalFrom(r.Context())
	if principal == nil {
		return Actor{}
	}
	return Actor{UserID: principal.UserID, IsMega: principal.Role == "MEGA_ADMIN"}
}

func participantFrom(r *http.Request) (*eventparticipants.Principal, bool) {
	principal := eventparticipants.PrincipalFrom(r.Context())
	return principal, principal != nil
}

func (h *Handler) AdminProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.AdminProducts(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, products, map[string]any{"count": len(products)})
}

func (h *Handler) AdminProduct(w http.ResponseWriter, r *http.Request) {
	product, err := h.svc.AdminProduct(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "productId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, product, nil)
}

func (h *Handler) AdminCreateProduct(w http.ResponseWriter, r *http.Request) {
	var input ProductInput
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeError(w, r, ErrValidation)
		return
	}
	product, err := h.svc.CreateProduct(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, product, nil)
}

func (h *Handler) AdminUpdateProduct(w http.ResponseWriter, r *http.Request) {
	var input ProductInput
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeError(w, r, ErrValidation)
		return
	}
	product, err := h.svc.UpdateProduct(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "productId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, product, nil)
}

func (h *Handler) AdminTransitionProduct(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		product, err := h.svc.TransitionProduct(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
			chi.URLParam(r, "productId"), action)
		if err != nil {
			writeError(w, r, err)
			return
		}
		httpserver.WriteJSON(w, r, http.StatusOK, product, nil)
	}
}

func (h *Handler) AdminDeleteProduct(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteProduct(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "productId"), h.store)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

func (h *Handler) AdminAddImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes)
	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, r, ErrValidation)
		return
	}
	defer file.Close()
	sortOrder, _ := strconv.Atoi(r.FormValue("sort_order"))
	image, err := h.svc.AddImage(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "productId"), ImageUpload{
			OriginalName: header.Filename, ContentType: header.Header.Get("Content-Type"),
			Size: header.Size, Reader: file, KeySuffix: uuid.NewString(), SortOrder: sortOrder,
		}, h.store)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusCreated, image, nil)
}

func (h *Handler) AdminDeleteImage(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteImage(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "productId"), chi.URLParam(r, "imageId"), h.store)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

func (h *Handler) AdminOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.svc.AdminOrders(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, orders, map[string]any{"count": len(orders)})
}

func (h *Handler) AdminOrder(w http.ResponseWriter, r *http.Request) {
	order, err := h.svc.AdminOrder(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "orderId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, order, nil)
}

func writeOrderResult(w http.ResponseWriter, r *http.Request, result *OrderResult) {
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	httpserver.WriteJSON(w, r, status, result, nil)
}

func (h *Handler) AdminIssue(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.Issue(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "orderId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeOrderResult(w, r, result)
}

func (h *Handler) AdminReject(w http.ResponseWriter, r *http.Request) {
	var input RejectInput
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeError(w, r, ErrValidation)
		return
	}
	result, err := h.svc.Reject(r.Context(), actorFrom(r), chi.URLParam(r, "contestId"),
		chi.URLParam(r, "orderId"), input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeOrderResult(w, r, result)
}

func (h *Handler) ParticipantProducts(w http.ResponseWriter, r *http.Request) {
	principal, ok := participantFrom(r)
	if !ok {
		writeError(w, r, ErrForbidden)
		return
	}
	products, err := h.svc.ParticipantProducts(r.Context(), principal.Event.ID, principal.Participant.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, products, map[string]any{"count": len(products)})
}

func (h *Handler) ParticipantProduct(w http.ResponseWriter, r *http.Request) {
	principal, ok := participantFrom(r)
	if !ok {
		writeError(w, r, ErrForbidden)
		return
	}
	product, err := h.svc.ParticipantProduct(r.Context(), principal.Event.ID,
		principal.Participant.ID, chi.URLParam(r, "productSlug"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, product, nil)
}

type savingTargetInput struct {
	ProductID string `json:"product_id"`
}

func (h *Handler) ParticipantSetTarget(w http.ResponseWriter, r *http.Request) {
	principal, ok := participantFrom(r)
	if !ok {
		writeError(w, r, ErrForbidden)
		return
	}
	var input savingTargetInput
	if json.NewDecoder(r.Body).Decode(&input) != nil || strings.TrimSpace(input.ProductID) == "" {
		writeError(w, r, ErrValidation)
		return
	}
	product, err := h.svc.SetSavingTarget(r.Context(), principal.Event.ID,
		principal.Participant.ID, input.ProductID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, product, nil)
}

func (h *Handler) ParticipantDeleteTarget(w http.ResponseWriter, r *http.Request) {
	principal, ok := participantFrom(r)
	if !ok {
		writeError(w, r, ErrForbidden)
		return
	}
	if err := h.svc.DeleteSavingTarget(r.Context(), principal.Event.ID, principal.Participant.ID); err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, map[string]bool{"deleted": true}, nil)
}

func (h *Handler) ParticipantReserve(w http.ResponseWriter, r *http.Request) {
	principal, ok := participantFrom(r)
	if !ok {
		writeError(w, r, ErrForbidden)
		return
	}
	var input ReserveInput
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		writeError(w, r, ErrValidation)
		return
	}
	if key := strings.TrimSpace(r.Header.Get("Idempotency-Key")); key != "" {
		input.IdempotencyKey = key
	}
	result, err := h.svc.Reserve(r.Context(), principal.Event.ID, principal.Participant.ID, input)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeOrderResult(w, r, result)
}

func (h *Handler) ParticipantOrders(w http.ResponseWriter, r *http.Request) {
	principal, ok := participantFrom(r)
	if !ok {
		writeError(w, r, ErrForbidden)
		return
	}
	orders, err := h.svc.ParticipantOrders(r.Context(), principal.Event.ID, principal.Participant.ID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, orders, map[string]any{"count": len(orders)})
}

func (h *Handler) ParticipantOrder(w http.ResponseWriter, r *http.Request) {
	principal, ok := participantFrom(r)
	if !ok {
		writeError(w, r, ErrForbidden)
		return
	}
	order, err := h.svc.ParticipantOrder(r.Context(), principal.Event.ID,
		principal.Participant.ID, chi.URLParam(r, "orderId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, order, nil)
}

func (h *Handler) ParticipantCancel(w http.ResponseWriter, r *http.Request) {
	principal, ok := participantFrom(r)
	if !ok {
		writeError(w, r, ErrForbidden)
		return
	}
	result, err := h.svc.Cancel(r.Context(), principal.Event.ID,
		principal.Participant.ID, chi.URLParam(r, "orderId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeOrderResult(w, r, result)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "MERCH_NOT_FOUND", "Товар или заказ не найден", nil)
	case errors.Is(err, ErrForbidden):
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав", nil)
	case errors.Is(err, ErrValidation):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Проверьте данные товара или заказа", nil)
	case errors.Is(err, ErrInsufficientStock):
		httpserver.WriteError(w, r, http.StatusConflict, "INSUFFICIENT_STOCK", "Недостаточно товара на складе", nil)
	case errors.Is(err, ErrInsufficientPoints):
		httpserver.WriteError(w, r, http.StatusConflict, "INSUFFICIENT_POINTS", "Недостаточно доступных баллов", nil)
	case errors.Is(err, ErrIdempotencyConflict):
		httpserver.WriteError(w, r, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Ключ запроса уже использован для другого заказа", nil)
	case errors.Is(err, ErrInvalidTransition):
		httpserver.WriteError(w, r, http.StatusConflict, "INVALID_MERCH_TRANSITION", "Операция недоступна в текущем статусе", nil)
	case errors.Is(err, ErrStorageDisabled):
		httpserver.WriteError(w, r, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Хранилище изображений временно недоступно", nil)
	default:
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", nil)
	}
}
