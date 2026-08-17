package merch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type Repository interface {
	Can(ctx context.Context, userID, contestID, permission string) (bool, error)
	ListProducts(ctx context.Context, contestID string, participantID *string, admin bool) ([]Product, error)
	ProductByID(ctx context.Context, contestID, productID string) (*Product, error)
	ProductBySlug(ctx context.Context, contestID, slug, participantID string) (*Product, error)
	CreateProduct(ctx context.Context, contestID, slugBase string, input ProductInput) (*Product, error)
	UpdateProduct(ctx context.Context, contestID, productID string, input ProductInput) (*Product, error)
	TransitionProduct(ctx context.Context, contestID, productID, action string) (*Product, error)
	DeleteProduct(ctx context.Context, contestID, productID string) ([]string, error)
	AddImage(ctx context.Context, contestID, productID string, image ProductImage) (*ProductImage, error)
	DeleteImage(ctx context.Context, contestID, productID, imageID string) (*ProductImage, error)
	SetSavingTarget(ctx context.Context, contestID, participantID, productID string) (*Product, error)
	DeleteSavingTarget(ctx context.Context, contestID, participantID string) error
	Reserve(ctx context.Context, params ReserveParams) (*OrderResult, error)
	ParticipantOrders(ctx context.Context, contestID, participantID string) ([]Order, error)
	ParticipantOrder(ctx context.Context, contestID, participantID, orderID string) (*Order, error)
	AdminOrders(ctx context.Context, contestID, status string) ([]Order, error)
	AdminOrder(ctx context.Context, contestID, orderID string) (*Order, error)
	Issue(ctx context.Context, actor Actor, contestID, orderID string) (*OrderResult, error)
	Reject(ctx context.Context, actor Actor, contestID, orderID, reason string) (*OrderResult, error)
	Cancel(ctx context.Context, contestID, participantID, orderID string) (*OrderResult, error)
}

type Auditor interface {
	Log(ctx context.Context, actorUserID, action, entityType, entityID string, meta map[string]any)
}

type FileStore interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, key string) error
}

type Service struct {
	repo     Repository
	audit    Auditor
	presign  func(context.Context, string) (string, error)
	now      func() time.Time
	maxImage int64
}

func NewService(repo Repository, audit Auditor, maxImageBytes int64) *Service {
	if maxImageBytes <= 0 {
		maxImageBytes = 20 << 20
	}
	return &Service{repo: repo, audit: audit, now: time.Now, maxImage: maxImageBytes}
}

func (s *Service) SetPresigner(fn func(context.Context, string) (string, error)) {
	s.presign = fn
}

func (s *Service) ensure(ctx context.Context, actor Actor, contestID, permission string) error {
	if actor.IsMega {
		return nil
	}
	allowed, err := s.repo.Can(ctx, actor.UserID, contestID, permission)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *Service) AdminProducts(ctx context.Context, actor Actor, contestID string) ([]Product, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageProducts); err != nil {
		return nil, err
	}
	products, err := s.repo.ListProducts(ctx, contestID, nil, true)
	s.decorateProducts(ctx, products)
	return nonNilProducts(products), err
}

func (s *Service) ParticipantProducts(ctx context.Context, contestID, participantID string) ([]Product, error) {
	products, err := s.repo.ListProducts(ctx, contestID, &participantID, false)
	s.decorateProducts(ctx, products)
	return nonNilProducts(products), err
}

func (s *Service) AdminProduct(ctx context.Context, actor Actor, contestID, productID string) (*Product, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageProducts); err != nil {
		return nil, err
	}
	product, err := s.repo.ProductByID(ctx, contestID, productID)
	if err == nil {
		s.decorateProduct(ctx, product)
	}
	return product, err
}

func (s *Service) ParticipantProduct(ctx context.Context, contestID, participantID, slug string) (*Product, error) {
	product, err := s.repo.ProductBySlug(ctx, contestID, strings.TrimSpace(slug), participantID)
	if err == nil {
		s.decorateProduct(ctx, product)
	}
	return product, err
}

func (s *Service) CreateProduct(ctx context.Context, actor Actor, contestID string, input ProductInput) (*Product, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageProducts); err != nil {
		return nil, err
	}
	input, err := validateProductInput(input)
	if err != nil {
		return nil, err
	}
	base := slugify(input.Title)
	if base == "" {
		base = "product"
	}
	product, err := s.repo.CreateProduct(ctx, contestID, base, input)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "MERCH_PRODUCT_CREATED", "merch_product", product.ID,
			map[string]any{"contest_id": contestID})
	}
	return product, err
}

func (s *Service) UpdateProduct(ctx context.Context, actor Actor, contestID, productID string, input ProductInput) (*Product, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageProducts); err != nil {
		return nil, err
	}
	input, err := validateProductInput(input)
	if err != nil {
		return nil, err
	}
	product, err := s.repo.UpdateProduct(ctx, contestID, productID, input)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "MERCH_PRODUCT_UPDATED", "merch_product", productID,
			map[string]any{"contest_id": contestID, "stock_quantity": input.StockQuantity})
	}
	return product, err
}

func (s *Service) TransitionProduct(ctx context.Context, actor Actor, contestID, productID, action string) (*Product, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageProducts); err != nil {
		return nil, err
	}
	if action != "activate" && action != "hide" {
		return nil, ErrValidation
	}
	product, err := s.repo.TransitionProduct(ctx, contestID, productID, action)
	if err == nil && s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "MERCH_PRODUCT_STATUS_CHANGED", "merch_product", productID,
			map[string]any{"contest_id": contestID, "status": product.Status})
	}
	return product, err
}

func (s *Service) DeleteProduct(ctx context.Context, actor Actor, contestID, productID string, store FileStore) error {
	if err := s.ensure(ctx, actor, contestID, PermissionManageProducts); err != nil {
		return err
	}
	keys, err := s.repo.DeleteProduct(ctx, contestID, productID)
	if err != nil {
		return err
	}
	if store != nil {
		for _, key := range keys {
			_ = store.Remove(ctx, key)
		}
	}
	if s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "MERCH_PRODUCT_DELETED", "merch_product", productID,
			map[string]any{"contest_id": contestID})
	}
	return nil
}

func (s *Service) SetSavingTarget(ctx context.Context, contestID, participantID, productID string) (*Product, error) {
	product, err := s.repo.SetSavingTarget(ctx, contestID, participantID, productID)
	if err == nil {
		product.IsSavingTarget = true
		s.decorateProduct(ctx, product)
	}
	return product, err
}

func (s *Service) DeleteSavingTarget(ctx context.Context, contestID, participantID string) error {
	return s.repo.DeleteSavingTarget(ctx, contestID, participantID)
}

func (s *Service) Reserve(ctx context.Context, contestID, participantID string, input ReserveInput) (*OrderResult, error) {
	items, key, fingerprint, err := validateReserveInput(input)
	if err != nil {
		return nil, err
	}
	return s.repo.Reserve(ctx, ReserveParams{
		ContestID: contestID, EventParticipantID: participantID, Items: items,
		IdempotencyKey: key, RequestFingerprint: fingerprint, Now: s.now(),
	})
}

func (s *Service) ParticipantOrders(ctx context.Context, contestID, participantID string) ([]Order, error) {
	orders, err := s.repo.ParticipantOrders(ctx, contestID, participantID)
	return nonNilOrders(orders), err
}

func (s *Service) ParticipantOrder(ctx context.Context, contestID, participantID, orderID string) (*Order, error) {
	return s.repo.ParticipantOrder(ctx, contestID, participantID, orderID)
}

func (s *Service) Cancel(ctx context.Context, contestID, participantID, orderID string) (*OrderResult, error) {
	return s.repo.Cancel(ctx, contestID, participantID, orderID)
}

func (s *Service) AdminOrders(ctx context.Context, actor Actor, contestID, status string) ([]Order, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageOrders); err != nil {
		return nil, err
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "" && status != OrderReserved && status != OrderIssued && status != OrderRejected && status != OrderCancelled {
		return nil, ErrValidation
	}
	orders, err := s.repo.AdminOrders(ctx, contestID, status)
	return nonNilOrders(orders), err
}

func (s *Service) AdminOrder(ctx context.Context, actor Actor, contestID, orderID string) (*Order, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageOrders); err != nil {
		return nil, err
	}
	return s.repo.AdminOrder(ctx, contestID, orderID)
}

func (s *Service) Issue(ctx context.Context, actor Actor, contestID, orderID string) (*OrderResult, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageOrders); err != nil {
		return nil, err
	}
	return s.repo.Issue(ctx, actor, contestID, orderID)
}

func (s *Service) Reject(ctx context.Context, actor Actor, contestID, orderID string, input RejectInput) (*OrderResult, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageOrders); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(input.Reason)
	if len(reason) < 3 || len(reason) > 2000 {
		return nil, ErrValidation
	}
	return s.repo.Reject(ctx, actor, contestID, orderID, reason)
}

func validateProductInput(input ProductInput) (ProductInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" || input.Description == "" || len(input.Title) > 300 ||
		len(input.Description) > 20_000 || input.PricePoints <= 0 ||
		input.PricePoints > 1_000_000_000 || input.StockQuantity < 0 || input.StockQuantity > 1_000_000 {
		return ProductInput{}, ErrValidation
	}
	if input.DiscountPricePoints != nil && (*input.DiscountPricePoints <= 0 || *input.DiscountPricePoints >= input.PricePoints) {
		return ProductInput{}, ErrValidation
	}
	return input, nil
}

func validateReserveInput(input ReserveInput) ([]OrderItemInput, string, string, error) {
	key := strings.TrimSpace(input.IdempotencyKey)
	if len(key) < 8 || len(key) > 200 || len(input.Items) == 0 || len(input.Items) > 20 {
		return nil, "", "", ErrValidation
	}
	quantities := make(map[string]int, len(input.Items))
	for _, item := range input.Items {
		id := strings.TrimSpace(item.ProductID)
		if id == "" || item.Quantity <= 0 || item.Quantity > 99 {
			return nil, "", "", ErrValidation
		}
		quantities[id] += item.Quantity
		if quantities[id] > 99 {
			return nil, "", "", ErrValidation
		}
	}
	items := make([]OrderItemInput, 0, len(quantities))
	for id, quantity := range quantities {
		items = append(items, OrderItemInput{ProductID: id, Quantity: quantity})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ProductID < items[j].ProductID })
	payload, _ := json.Marshal(items)
	sum := sha256.Sum256(payload)
	return items, key, hex.EncodeToString(sum[:]), nil
}

func (s *Service) decorateProducts(ctx context.Context, products []Product) {
	for i := range products {
		s.decorateProduct(ctx, &products[i])
	}
}

func (s *Service) decorateProduct(ctx context.Context, product *Product) {
	product.AvailableQuantity = product.StockQuantity - product.ReservedQuantity
	product.EffectivePrice = product.PricePoints
	if product.DiscountPricePoints != nil {
		product.EffectivePrice = *product.DiscountPricePoints
	}
	if product.Images == nil {
		product.Images = []ProductImage{}
	}
	if s.presign == nil {
		return
	}
	for i := range product.Images {
		if value, err := s.presign(ctx, product.Images[i].ObjectKey); err == nil {
			product.Images[i].URL = &value
		}
	}
}

func nonNilProducts(products []Product) []Product {
	if products == nil {
		return []Product{}
	}
	return products
}

func nonNilOrders(orders []Order) []Order {
	if orders == nil {
		return []Order{}
	}
	return orders
}

func productImageKey(contestID, productID string, image ImageUpload) string {
	return fmt.Sprintf("merch/%s/%s/%s-%s", contestID, productID, image.KeySuffix, safeName(image.OriginalName))
}
