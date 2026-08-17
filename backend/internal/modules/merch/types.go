// Package merch implements the event points shop and atomic reservations.
package merch

import (
	"errors"
	"io"
	"time"
)

var (
	ErrNotFound            = errors.New("merch entity not found")
	ErrForbidden           = errors.New("no access to event merch")
	ErrValidation          = errors.New("merch validation error")
	ErrInvalidTransition   = errors.New("invalid merch transition")
	ErrInsufficientStock   = errors.New("insufficient merch stock")
	ErrInsufficientPoints  = errors.New("insufficient participant points")
	ErrIdempotencyConflict = errors.New("merch idempotency conflict")
	ErrStorageDisabled     = errors.New("merch image storage unavailable")
)

const (
	ProductDraft   = "DRAFT"
	ProductActive  = "ACTIVE"
	ProductHidden  = "HIDDEN"
	ProductSoldOut = "SOLD_OUT"

	OrderReserved  = "RESERVED"
	OrderIssued    = "ISSUED"
	OrderRejected  = "REJECTED"
	OrderCancelled = "CANCELLED"

	HoldActive   = "ACTIVE"
	HoldCaptured = "CAPTURED"
	HoldReleased = "RELEASED"

	PermissionManageProducts = "event.merch.manage"
	PermissionManageOrders   = "event.merch.orders.manage"
)

type Actor struct {
	UserID string
	IsMega bool
}

type ProductImage struct {
	ID           string    `json:"id"`
	ProductID    string    `json:"product_id"`
	ObjectKey    string    `json:"-"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	SortOrder    int       `json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`
	URL          *string   `json:"url"`
}

type Product struct {
	ID                  string         `json:"id"`
	ContestID           string         `json:"event_id"`
	Title               string         `json:"title"`
	Slug                string         `json:"slug"`
	Description         string         `json:"description"`
	PricePoints         int64          `json:"price_points"`
	DiscountPricePoints *int64         `json:"discount_price_points"`
	StockQuantity       int            `json:"stock_quantity"`
	ReservedQuantity    int            `json:"reserved_quantity"`
	AvailableQuantity   int            `json:"available_quantity"`
	EffectivePrice      int64          `json:"effective_price_points"`
	InterestedCount     int            `json:"interested_count"`
	IsSavingTarget      bool           `json:"is_saving_target"`
	Status              string         `json:"status"`
	Images              []ProductImage `json:"images"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

type ProductInput struct {
	Title               string `json:"title"`
	Description         string `json:"description"`
	PricePoints         int64  `json:"price_points"`
	DiscountPricePoints *int64 `json:"discount_price_points"`
	StockQuantity       int    `json:"stock_quantity"`
}

type ImageUpload struct {
	OriginalName string
	ContentType  string
	Size         int64
	Reader       io.Reader
	KeySuffix    string
	SortOrder    int
}

type OrderItemInput struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type ReserveInput struct {
	Items          []OrderItemInput `json:"items"`
	IdempotencyKey string           `json:"idempotency_key"`
}

type ReserveParams struct {
	ContestID          string
	EventParticipantID string
	Items              []OrderItemInput
	IdempotencyKey     string
	RequestFingerprint string
	Now                time.Time
}

type OrderItem struct {
	ID           string `json:"id"`
	ProductID    string `json:"product_id"`
	ProductTitle string `json:"product_title"`
	Quantity     int    `json:"quantity"`
	PricePoints  int64  `json:"price_points"`
	TotalPoints  int64  `json:"total_points"`
}

type Order struct {
	ID                 string      `json:"id"`
	ContestID          string      `json:"event_id"`
	EventParticipantID string      `json:"participant_id"`
	ParticipantName    string      `json:"participant_name,omitempty"`
	Status             string      `json:"status"`
	PointsTotal        int64       `json:"points_total"`
	RejectionReason    *string     `json:"rejection_reason"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
	IssuedAt           *time.Time  `json:"issued_at"`
	RejectedAt         *time.Time  `json:"rejected_at"`
	CancelledAt        *time.Time  `json:"cancelled_at"`
	IssuedByUserID     *string     `json:"issued_by_user_id"`
	RejectedByUserID   *string     `json:"rejected_by_user_id"`
	Items              []OrderItem `json:"items"`
}

type OrderResult struct {
	Order    Order `json:"order"`
	Replayed bool  `json:"replayed"`
}

type RejectInput struct {
	Reason string `json:"reason"`
}
