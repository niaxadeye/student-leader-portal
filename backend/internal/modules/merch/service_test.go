package merch

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

type serviceRepo struct {
	Repository
	allowed   bool
	created   ProductInput
	slugBase  string
	reserve   ReserveParams
	product   Product
	createErr error
}

func (f *serviceRepo) Can(context.Context, string, string, string) (bool, error) {
	return f.allowed, nil
}

func (f *serviceRepo) CreateProduct(
	_ context.Context,
	_ string,
	slugBase string,
	input ProductInput,
) (*Product, error) {
	f.slugBase, f.created = slugBase, input
	if f.createErr != nil {
		return nil, f.createErr
	}
	product := f.product
	if product.ID == "" {
		product.ID = "product-1"
	}
	return &product, nil
}

func (f *serviceRepo) Reserve(_ context.Context, params ReserveParams) (*OrderResult, error) {
	f.reserve = params
	return &OrderResult{Order: Order{ID: "order-1"}}, nil
}

func TestCreateProductValidatesAndSlugifies(t *testing.T) {
	t.Parallel()
	repo := &serviceRepo{allowed: true}
	service := NewService(repo, nil, 1024)
	discount := int64(75)
	product, err := service.CreateProduct(context.Background(), Actor{UserID: "staff"}, "event-1", ProductInput{
		Title: "  Футболка лидера  ", Description: "  Хлопок  ", PricePoints: 100,
		DiscountPricePoints: &discount, StockQuantity: 12,
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if product.ID != "product-1" || repo.slugBase != "futbolka-lidera" {
		t.Fatalf("product=%#v slug=%q", product, repo.slugBase)
	}
	if repo.created.Title != "Футболка лидера" || repo.created.Description != "Хлопок" {
		t.Fatalf("input was not normalized: %#v", repo.created)
	}
}

func TestCreateProductRejectsInvalidDiscountAndPermission(t *testing.T) {
	t.Parallel()
	tooHigh := int64(100)
	_, err := NewService(&serviceRepo{allowed: true}, nil, 1024).CreateProduct(
		context.Background(), Actor{UserID: "staff"}, "event-1",
		ProductInput{Title: "Товар", Description: "Описание", PricePoints: 100, DiscountPricePoints: &tooHigh},
	)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("error=%v, want ErrValidation", err)
	}
	_, err = NewService(&serviceRepo{}, nil, 1024).CreateProduct(
		context.Background(), Actor{UserID: "staff"}, "event-1",
		ProductInput{Title: "Товар", Description: "Описание", PricePoints: 100},
	)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error=%v, want ErrForbidden", err)
	}
}

func TestReserveCanonicalizesItemsAndFingerprint(t *testing.T) {
	t.Parallel()
	repo := &serviceRepo{}
	service := NewService(repo, nil, 1024)
	input := ReserveInput{IdempotencyKey: " order-key-001 ", Items: []OrderItemInput{
		{ProductID: "b", Quantity: 1},
		{ProductID: "a", Quantity: 2},
		{ProductID: "b", Quantity: 3},
	}}
	if _, err := service.Reserve(context.Background(), "event-1", "participant-1", input); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if repo.reserve.IdempotencyKey != "order-key-001" || len(repo.reserve.Items) != 2 ||
		repo.reserve.Items[0].ProductID != "a" || repo.reserve.Items[0].Quantity != 2 ||
		repo.reserve.Items[1].ProductID != "b" || repo.reserve.Items[1].Quantity != 4 ||
		len(repo.reserve.RequestFingerprint) != 64 {
		t.Fatalf("canonical params=%#v", repo.reserve)
	}

	repo2 := &serviceRepo{}
	permuted := ReserveInput{IdempotencyKey: "order-key-001", Items: []OrderItemInput{
		{ProductID: "b", Quantity: 4}, {ProductID: "a", Quantity: 2},
	}}
	if _, err := NewService(repo2, nil, 1024).Reserve(
		context.Background(), "event-1", "participant-1", permuted,
	); err != nil {
		t.Fatalf("permuted Reserve: %v", err)
	}
	if repo.reserve.RequestFingerprint != repo2.reserve.RequestFingerprint {
		t.Fatalf("fingerprints differ: %s != %s", repo.reserve.RequestFingerprint, repo2.reserve.RequestFingerprint)
	}
}

func TestValidateImage(t *testing.T) {
	t.Parallel()
	service := NewService(&serviceRepo{}, nil, 1024)
	upload, err := service.validateImage(ImageUpload{
		OriginalName: "cover.webp", ContentType: "image/webp; charset=binary",
		Size: 512, Reader: bytes.NewReader([]byte("RIFF1234WEBPpayload")), KeySuffix: "nonce",
	})
	if err != nil || upload.ContentType != "image/webp" {
		t.Fatalf("valid image: mime=%q err=%v", upload.ContentType, err)
	}
	_, err = service.validateImage(ImageUpload{
		OriginalName: "cover.exe", ContentType: "image/webp", Size: 512,
		Reader: bytes.NewReader([]byte("RIFF1234WEBPpayload")), KeySuffix: "nonce",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("error=%v, want ErrValidation", err)
	}
}
