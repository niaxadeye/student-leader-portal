package submissions

import (
	"context"
	"time"
)

type Service struct {
	repo   *Repo
	source ChallengeSource
	audit  Auditor
	// presign превращает object_key в скачиваемую ссылку (nil → отдаём key как есть).
	presign func(ctx context.Context, objectKey string) (string, error)
	// now — для тестируемости; в проде time.Now.
	now func() time.Time
}

func NewService(repo *Repo, source ChallengeSource, audit Auditor) *Service {
	return &Service{repo: repo, source: source, audit: audit, now: time.Now}
}

// SetPresigner подключает функцию подписи ссылок на файлы (из storage).
func (s *Service) SetPresigner(fn func(ctx context.Context, objectKey string) (string, error)) {
	s.presign = fn
}

// GetOrCreateDraft — контестант открывает испытание: гарантируем черновик и отдаём с файлами.
// Требует активного участия в конкурсе и опубликованного испытания.
func (s *Service) GetOrCreateDraft(ctx context.Context, a Actor, challengeID string) (*Submission, error) {
	info, err := s.source.ChallengeInfo(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureParticipant(ctx, a, info.ContestID); err != nil {
		return nil, err
	}
	// Черновик можно открыть только на опубликованном (или закрытом — читать/просматривать) испытании.
	if info.Status == "DRAFT" || info.Status == "ARCHIVED" {
		return nil, ErrNotFound
	}
	sub, err := s.repo.EnsureDraft(ctx, challengeID, a.UserID, info.CurrentSchemaVersion)
	if err != nil {
		return nil, err
	}
	return s.withFiles(ctx, sub)
}

// SaveDraft сохраняет ответы черновика (без создания ревизии).
func (s *Service) SaveDraft(ctx context.Context, a Actor, challengeID string, answers map[string]any) (*Submission, error) {
	info, sub, err := s.loadForWrite(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureWindowOpen(info, sub); err != nil {
		return nil, err
	}
	if err := s.repo.SaveAnswers(ctx, sub.ID, answers); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "submission.save_draft", "submission", sub.ID, nil)
	fresh, err := s.repo.ByID(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	return s.withFiles(ctx, fresh)
}

// ensureParticipant — доступ конкурсанта по активному участию (SUPER_ADMIN проходит всегда).
func (s *Service) ensureParticipant(ctx context.Context, a Actor, contestID string) error {
	if a.IsSuper {
		return nil
	}
	ok, err := s.source.IsParticipant(ctx, a.UserID, contestID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// loadForWrite — общий префикс мутаций: инфо об испытании + работа конкурсанта.
func (s *Service) loadForWrite(ctx context.Context, a Actor, challengeID string) (*ChallengeInfo, *Submission, error) {
	info, err := s.source.ChallengeInfo(ctx, challengeID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.ensureParticipant(ctx, a, info.ContestID); err != nil {
		return nil, nil, err
	}
	sub, err := s.repo.EnsureDraft(ctx, challengeID, a.UserID, info.CurrentSchemaVersion)
	if err != nil {
		return nil, nil, err
	}
	return info, sub, nil
}

// ensureWindowOpen проверяет, что подача сейчас разрешена (SITE.md §7.5).
func (s *Service) ensureWindowOpen(info *ChallengeInfo, sub *Submission) error {
	if sub.Status == StatusLocked || sub.LockedAt != nil {
		return ErrLocked
	}
	if info.Status != "PUBLISHED" {
		return ErrClosed
	}
	now := s.now()
	if info.OpenAt != nil && now.Before(*info.OpenAt) {
		return ErrClosed
	}
	if info.DeadlineAt != nil && now.After(*info.DeadlineAt) {
		// Поздняя отправка допускается, только если разрешена в настройках испытания.
		if allow, _ := info.Settings["allow_late_submission"].(bool); !allow {
			return ErrDeadline
		}
	}
	return nil
}

// withFiles присоединяет файлы к работе и подписывает download-URL (object_key → presigned).
func (s *Service) withFiles(ctx context.Context, sub *Submission) (*Submission, error) {
	files, err := s.repo.LoadFiles(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	// Ключ поля по field_id: подтягиваем схему для отображения.
	fields, _ := s.source.SchemaFields(ctx, sub.ChallengeID)
	keyByID := make(map[string]string, len(fields))
	for _, f := range fields {
		keyByID[f.ID] = f.Key
	}
	for i := range files {
		if files[i].FieldID != nil {
			files[i].FieldKey = keyByID[*files[i].FieldID]
		}
		if s.presign != nil {
			if url, err := s.presign(ctx, files[i].DownloadURL); err == nil {
				files[i].DownloadURL = url
			}
		}
	}
	sub.Files = files
	return sub, nil
}
