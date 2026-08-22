package contests

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/eazytech/student-leader-cabinet/internal/platform/filevalidation"
)

type ImageUpload struct {
	OriginalName string
	ContentType  string
	Size         int64
	Reader       io.Reader
	KeySuffix    string
}

func (s *Service) SetAvatar(
	ctx context.Context,
	a Actor,
	contestID, userID string,
	upload ImageUpload,
	store ImageStore,
) (string, error) {
	if err := s.ensureOwnerOrMega(ctx, a, contestID); err != nil {
		return "", err
	}
	ok, err := s.repo.IsActiveContestant(ctx, contestID, userID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNotFound
	}
	if store == nil {
		return "", ErrStorageDisabled
	}
	upload, err = s.validateImage(upload)
	if err != nil {
		return "", err
	}
	objectKey := avatarObjectKey(userID, upload)
	if err := store.Put(ctx, objectKey, upload.Reader, upload.Size, upload.ContentType); err != nil {
		return "", err
	}
	prev, err := s.repo.SetUserAvatarKey(ctx, userID, &objectKey)
	if err != nil {
		_ = store.Remove(ctx, objectKey)
		return "", err
	}
	if prev != nil && *prev != objectKey {
		_ = store.Remove(ctx, *prev)
	}
	s.audit.Log(ctx, a.UserID, "CONTESTANT_AVATAR_UPDATED", "user", userID,
		map[string]any{"contest_id": contestID})
	return objectKey, nil
}

func (s *Service) DeleteAvatar(ctx context.Context, a Actor, contestID, userID string, store ImageStore) error {
	if err := s.ensureOwnerOrMega(ctx, a, contestID); err != nil {
		return err
	}
	ok, err := s.repo.IsActiveContestant(ctx, contestID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	prev, err := s.repo.SetUserAvatarKey(ctx, userID, nil)
	if err != nil {
		return err
	}
	if prev != nil && store != nil {
		_ = store.Remove(ctx, *prev)
	}
	s.audit.Log(ctx, a.UserID, "CONTESTANT_AVATAR_DELETED", "user", userID,
		map[string]any{"contest_id": contestID})
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

func avatarObjectKey(userID string, upload ImageUpload) string {
	return fmt.Sprintf("avatars/%s/%s-%s", userID, upload.KeySuffix, safeFileName(upload.OriginalName))
}

func safeFileName(name string) string {
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
		return "avatar"
	}
	return name
}
