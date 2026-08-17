package merch

import (
	"context"
	"path"
	"strings"

	"github.com/eazytech/student-leader-cabinet/internal/platform/filevalidation"
)

func (s *Service) AddImage(
	ctx context.Context,
	actor Actor,
	contestID, productID string,
	upload ImageUpload,
	store FileStore,
) (*ProductImage, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManageProducts); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, ErrStorageDisabled
	}
	upload, err := s.validateImage(upload)
	if err != nil {
		return nil, err
	}
	objectKey := productImageKey(contestID, productID, upload)
	if err := store.Put(ctx, objectKey, upload.Reader, upload.Size, upload.ContentType); err != nil {
		return nil, err
	}
	image, err := s.repo.AddImage(ctx, contestID, productID, ProductImage{
		ObjectKey: objectKey, OriginalName: strings.TrimSpace(upload.OriginalName),
		MimeType: upload.ContentType, SizeBytes: upload.Size, SortOrder: upload.SortOrder,
	})
	if err != nil {
		_ = store.Remove(ctx, objectKey)
		return nil, err
	}
	if s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "MERCH_PRODUCT_IMAGE_ADDED", "merch_product", productID,
			map[string]any{"contest_id": contestID, "image_id": image.ID})
	}
	if s.presign != nil {
		if value, signErr := s.presign(ctx, image.ObjectKey); signErr == nil {
			image.URL = &value
		}
	}
	return image, nil
}

func (s *Service) DeleteImage(
	ctx context.Context,
	actor Actor,
	contestID, productID, imageID string,
	store FileStore,
) error {
	if err := s.ensure(ctx, actor, contestID, PermissionManageProducts); err != nil {
		return err
	}
	image, err := s.repo.DeleteImage(ctx, contestID, productID, imageID)
	if err != nil {
		return err
	}
	if store != nil {
		_ = store.Remove(ctx, image.ObjectKey)
	}
	if s.audit != nil {
		s.audit.Log(ctx, actor.UserID, "MERCH_PRODUCT_IMAGE_DELETED", "merch_product", productID,
			map[string]any{"contest_id": contestID, "image_id": image.ID})
	}
	return nil
}

func (s *Service) validateImage(upload ImageUpload) (ImageUpload, error) {
	if upload.Size <= 0 || upload.Size > s.maxImage ||
		strings.TrimSpace(upload.OriginalName) == "" || strings.TrimSpace(upload.KeySuffix) == "" {
		return ImageUpload{}, ErrValidation
	}
	reader, mime, err := filevalidation.InspectImage(upload.Reader, upload.ContentType, upload.OriginalName)
	if err != nil {
		return ImageUpload{}, ErrValidation
	}
	upload.Reader = reader
	upload.ContentType = mime
	return upload, nil
}

func safeName(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, name)
	if name == "" || name == "." {
		return "image"
	}
	return name
}
