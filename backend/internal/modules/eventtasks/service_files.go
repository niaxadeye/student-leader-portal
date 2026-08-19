package eventtasks

import (
	"context"
	"fmt"
	"strings"

	"github.com/eazytech/student-leader-cabinet/internal/platform/filevalidation"
)

const maxSubmissionAssets = 10

func (s *Service) Submit(
	ctx context.Context,
	contestID, participantID, taskID string,
	input SubmitInput,
	store FileStore,
) (*Submission, error) {
	task, err := s.repo.Get(ctx, contestID, taskID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	if task.Status != StatusActive ||
		(task.StartsAt != nil && now.Before(*task.StartsAt)) ||
		(task.EndsAt != nil && now.After(*task.EndsAt)) {
		return nil, ErrSubmissionClosed
	}
	if existing, findErr := s.repo.ParticipantSubmission(ctx, contestID, taskID, participantID); findErr == nil {
		if existing.Status != SubmissionRejected {
			return nil, ErrInvalidTransition
		}
	} else if findErr != ErrNotFound {
		return nil, findErr
	}

	comment := strings.TrimSpace(input.ParticipantComment)
	if len(comment) > 2000 || len(input.Images) > maxSubmissionAssets || len(input.Links) > maxSubmissionAssets ||
		len(input.Images)+len(input.Links) == 0 {
		return nil, ErrValidation
	}
	allowed := make(map[string]bool, len(task.AllowedSubmissionTypes))
	for _, assetType := range task.AllowedSubmissionTypes {
		allowed[assetType] = true
	}
	if len(input.Images) > 0 && !allowed[AssetImage] || len(input.Links) > 0 && !allowed[AssetLink] {
		return nil, ErrValidation
	}

	assets := make([]StoredAsset, 0, len(input.Images)+len(input.Links))
	for _, raw := range input.Links {
		link, err := validateLink(raw)
		if err != nil {
			return nil, err
		}
		linkCopy := link
		assets = append(assets, StoredAsset{Type: AssetLink, ExternalURL: &linkCopy, SortOrder: len(assets)})
	}

	if len(input.Images) > 0 && store == nil {
		return nil, ErrStorageDisabled
	}
	uploadedKeys := make([]string, 0, len(input.Images))
	cleanup := func() {
		for _, key := range uploadedKeys {
			_ = store.Remove(ctx, key)
		}
	}
	for _, image := range input.Images {
		image, err = s.validateImage(image)
		if err != nil {
			cleanup()
			return nil, err
		}
		objectKey := fmt.Sprintf("event-tasks/%s/%s/%s/%s-%s",
			contestID, taskID, participantID, image.KeySuffix, safeName(image.OriginalName))
		if err := store.Put(ctx, objectKey, image.Reader, image.Size, image.ContentType); err != nil {
			cleanup()
			return nil, err
		}
		uploadedKeys = append(uploadedKeys, objectKey)
		keyCopy, nameCopy, typeCopy, sizeCopy := objectKey, image.OriginalName, image.ContentType, image.Size
		assets = append(assets, StoredAsset{
			Type: AssetImage, ObjectKey: &keyCopy, OriginalName: &nameCopy,
			MimeType: &typeCopy, SizeBytes: &sizeCopy, SortOrder: len(assets),
		})
	}

	var commentPtr *string
	if comment != "" {
		commentPtr = &comment
	}
	submission, err := s.repo.SubmitAttempt(ctx, SubmitParams{
		ContestID: contestID, TaskID: taskID, EventParticipantID: participantID,
		ParticipantComment: commentPtr, Assets: assets, Now: now,
	})
	if err != nil {
		cleanup()
		return nil, err
	}
	decorateParticipantAssetPaths(submission)
	return submission, nil
}

func (s *Service) SetImage(
	ctx context.Context,
	actor Actor,
	contestID, taskID string,
	image ImageUpload,
	store FileStore,
) (*Task, error) {
	return s.replaceTaskFile(ctx, actor, contestID, taskID, image, store, "cover", s.repo.SetImage, "EVENT_TASK_IMAGE_UPDATED")
}

func (s *Service) DeleteImage(
	ctx context.Context,
	actor Actor,
	contestID, taskID string,
	store FileStore,
) (*Task, error) {
	return s.clearTaskFile(ctx, actor, contestID, taskID, store, s.repo.SetImage, "EVENT_TASK_IMAGE_DELETED")
}

func (s *Service) SetIcon(
	ctx context.Context,
	actor Actor,
	contestID, taskID string,
	image ImageUpload,
	store FileStore,
) (*Task, error) {
	return s.replaceTaskFile(ctx, actor, contestID, taskID, image, store, "icon", s.repo.SetIcon, "EVENT_TASK_ICON_UPDATED")
}

func (s *Service) DeleteIcon(
	ctx context.Context,
	actor Actor,
	contestID, taskID string,
	store FileStore,
) (*Task, error) {
	return s.clearTaskFile(ctx, actor, contestID, taskID, store, s.repo.SetIcon, "EVENT_TASK_ICON_DELETED")
}

func (s *Service) replaceTaskFile(
	ctx context.Context,
	actor Actor,
	contestID, taskID string,
	image ImageUpload,
	store FileStore,
	kind string,
	set func(context.Context, string, string, *string) (*Task, *string, error),
	auditAction string,
) (*Task, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, ErrStorageDisabled
	}
	image, err := s.validateImage(image)
	if err != nil {
		return nil, err
	}
	objectKey := fmt.Sprintf("event-tasks/%s/%s/%s-%s-%s",
		contestID, taskID, kind, image.KeySuffix, safeName(image.OriginalName))
	if err := store.Put(ctx, objectKey, image.Reader, image.Size, image.ContentType); err != nil {
		return nil, err
	}
	task, previous, err := set(ctx, contestID, taskID, &objectKey)
	if err != nil {
		_ = store.Remove(ctx, objectKey)
		return nil, err
	}
	if previous != nil && *previous != objectKey {
		_ = store.Remove(ctx, *previous)
	}
	if s.audit != nil {
		s.audit.Log(ctx, actor.UserID, auditAction, "event_task", taskID,
			map[string]any{"contest_id": contestID})
	}
	s.decorateTask(ctx, task)
	return task, nil
}

func (s *Service) clearTaskFile(
	ctx context.Context,
	actor Actor,
	contestID, taskID string,
	store FileStore,
	set func(context.Context, string, string, *string) (*Task, *string, error),
	auditAction string,
) (*Task, error) {
	if err := s.ensure(ctx, actor, contestID, PermissionManage); err != nil {
		return nil, err
	}
	task, previous, err := set(ctx, contestID, taskID, nil)
	if err != nil {
		return nil, err
	}
	if previous != nil && store != nil {
		_ = store.Remove(ctx, *previous)
	}
	if s.audit != nil {
		s.audit.Log(ctx, actor.UserID, auditAction, "event_task", taskID,
			map[string]any{"contest_id": contestID})
	}
	s.decorateTask(ctx, task)
	return task, nil
}

func (s *Service) validateImage(image ImageUpload) (ImageUpload, error) {
	if image.Size <= 0 || image.Size > s.maxImage || strings.TrimSpace(image.OriginalName) == "" {
		return ImageUpload{}, ErrValidation
	}
	if strings.TrimSpace(image.KeySuffix) == "" {
		return ImageUpload{}, ErrValidation
	}
	reader, mime, err := filevalidation.InspectImage(image.Reader, image.ContentType, image.OriginalName)
	if err != nil {
		return ImageUpload{}, ErrValidation
	}
	image.Reader = reader
	image.ContentType = mime
	return image, nil
}
