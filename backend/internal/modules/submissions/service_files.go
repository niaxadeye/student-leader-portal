package submissions

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
)

// FileStore — часть storage, нужная модулю (запись/удаление объектов).
type FileStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, key string) error
}

// UploadInput — параметры загрузки файла в поле формы.
type UploadInput struct {
	ChallengeID  string
	FieldID      string
	OriginalName string
	ContentType  string
	Size         int64
	Reader       io.Reader
	KeySuffix    string // уникальный суффикс ключа (передаётся хендлером: без Date/rand в модуле)
}

// UploadFile сохраняет файл в MinIO и привязывает его к черновику работы (SITE.md §13.3).
// Разрешено только на открытом окне подачи; тип/размер валидируются по настройкам поля.
func (s *Service) UploadFile(ctx context.Context, a Actor, in UploadInput, store FileStore) (*SubmissionFile, error) {
	info, sub, err := s.loadForWrite(ctx, a, in.ChallengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureWindowOpen(info, sub); err != nil {
		return nil, err
	}

	field, err := s.findField(ctx, in.ChallengeID, in.FieldID)
	if err != nil {
		return nil, err
	}
	if field.Type != "FILE_GROUP" {
		return nil, fmt.Errorf("%w: поле не принимает файлы", ErrValidation)
	}
	ext := strings.TrimPrefix(strings.ToLower(path.Ext(in.OriginalName)), ".")
	if err := validateFile(field, ext, in.Size); err != nil {
		return nil, err
	}

	// Ключ объекта: contest/challenge/submission/field/suffix-имя. Без секретов.
	objectKey := fmt.Sprintf("submissions/%s/%s/%s/%s-%s",
		info.ContestID, in.ChallengeID, sub.ID, in.KeySuffix, safeName(in.OriginalName))
	if err := store.Put(ctx, objectKey, in.Reader, in.Size, in.ContentType); err != nil {
		return nil, err
	}

	fieldID := field.ID
	row := &FileRow{
		OwnerUserID:  a.UserID,
		ContestID:    info.ContestID,
		ChallengeID:  in.ChallengeID,
		Bucket:       "", // фактический bucket знает storage; храним ключ (bucket в конфиге)
		ObjectKey:    objectKey,
		OriginalName: in.OriginalName,
		SafeName:     safeName(in.OriginalName),
		MimeType:     strPtr(in.ContentType),
		SizeBytes:    in.Size,
	}
	if ext != "" {
		row.Extension = &ext
	}
	fileID, err := s.repo.InsertFile(ctx, row, sub.ID, &fieldID)
	if err != nil {
		_ = store.Remove(ctx, objectKey) // откат объекта при сбое БД
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "submission.file_upload", "submission", sub.ID,
		map[string]any{"file_id": fileID, "field_id": fieldID})

	out := &SubmissionFile{
		FileID: fileID, FieldID: &fieldID, FieldKey: field.Key,
		OriginalName: in.OriginalName, SizeBytes: &in.Size, MimeType: strPtr(in.ContentType),
	}
	return out, nil
}

// DeleteFile удаляет файл из черновика (soft) и объект из MinIO. Только владелец, только открытое окно.
func (s *Service) DeleteFile(ctx context.Context, a Actor, challengeID, fileID string, store FileStore) error {
	info, sub, err := s.loadForWrite(ctx, a, challengeID)
	if err != nil {
		return err
	}
	if err := s.ensureWindowOpen(info, sub); err != nil {
		return err
	}
	ownerID, objectKey, err := s.repo.FileByID(ctx, fileID)
	if err != nil {
		return ErrNotFound
	}
	if ownerID != a.UserID && !a.IsSuper {
		return ErrForbidden
	}
	if err := s.repo.SoftDeleteFile(ctx, sub.ID, fileID); err != nil {
		return err
	}
	_ = store.Remove(ctx, objectKey)
	s.audit.Log(ctx, a.UserID, "submission.file_delete", "submission", sub.ID, map[string]any{"file_id": fileID})
	return nil
}

// PresignFile отдаёт ссылку на скачивание файла (проверяет доступ: владелец или админ конкурса).
func (s *Service) PresignFile(ctx context.Context, a Actor, submissionID, fileID string) (string, error) {
	sub, err := s.repo.ByID(ctx, submissionID)
	if err != nil {
		return "", err
	}
	if sub.ContestantUserID != a.UserID {
		if err := s.ensureAdmin(ctx, a, sub.ChallengeID); err != nil {
			return "", err
		}
	}
	_, objectKey, err := s.repo.FileByID(ctx, fileID)
	if err != nil {
		return "", ErrNotFound
	}
	if s.presign == nil {
		return "", ErrNotFound
	}
	return s.presign(ctx, objectKey)
}

func (s *Service) findField(ctx context.Context, challengeID, fieldID string) (*FieldInfo, error) {
	fields, err := s.source.SchemaFields(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	for i := range fields {
		if fields[i].ID == fieldID {
			return &fields[i], nil
		}
	}
	return nil, fmt.Errorf("%w: поле не найдено", ErrValidation)
}

// validateFile проверяет расширение и размер по настройкам поля FILE_GROUP.
func validateFile(field *FieldInfo, ext string, size int64) error {
	if exts, ok := field.Settings["allowed_extensions"].([]any); ok && len(exts) > 0 {
		allowed := false
		for _, e := range exts {
			if s, _ := e.(string); strings.ToLower(s) == ext {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("%w: недопустимое расширение .%s", ErrValidation, ext)
		}
	}
	if maxMB, ok := field.Settings["max_file_size_mb"].(float64); ok && maxMB > 0 {
		if size > int64(maxMB)*1024*1024 {
			return fmt.Errorf("%w: файл больше %.0f МБ", ErrValidation, maxMB)
		}
	}
	return nil
}

// safeName очищает имя файла от путей и небезопасных символов.
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
		return "file"
	}
	return name
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
