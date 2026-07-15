package submissions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Submit валидирует обязательные поля, создаёт immutable-ревизию и переводит работу в SUBMITTED.
// Повторная отправка (resubmit) допускается, если окно подачи открыто.
func (s *Service) Submit(ctx context.Context, a Actor, challengeID string, answers map[string]any) (*Submission, error) {
	info, sub, err := s.loadForWrite(ctx, a, challengeID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureWindowOpen(info, sub); err != nil {
		return nil, err
	}

	fields, err := s.source.SchemaFields(ctx, challengeID)
	if err != nil {
		return nil, err
	}
	files, err := s.repo.LoadFiles(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	if err := validateRequired(fields, answers, files); err != nil {
		return nil, err
	}

	// Снапшоты схемы и файлов для ревизии.
	schemaSnap, _ := json.Marshal(map[string]any{"fields": fieldsToSnapshot(fields)})
	filesSnap, _ := json.Marshal(filesToSnapshot(files))
	checksum := computeChecksum(answers, files)

	action := ActionSubmit
	if sub.CurrentRevisionNumber > 0 {
		action = ActionResubmit
	}
	revNum := sub.CurrentRevisionNumber + 1

	// Тип outbox-события: первая отправка vs обновление формы (SITE.md §15).
	// Payload минимальный — диспетчер дорезолвит человекочитаемые поля при отправке.
	eventType := "submission.submitted"
	if action == ActionResubmit {
		eventType = "submission.resubmitted"
	}
	outboxPayload, _ := json.Marshal(map[string]any{
		"submission_id": sub.ID, "revision": revNum, "action": action,
	})

	if err := s.repo.Submit(ctx, SubmitParams{
		SubmissionID:    sub.ID,
		Answers:         answers,
		SchemaVersion:   info.CurrentSchemaVersion,
		SchemaSnapshot:  schemaSnap,
		FilesSnapshot:   filesSnap,
		Checksum:        checksum,
		ActionType:      action,
		RevisionNumber:  revNum,
		ActorID:         a.UserID,
		OutboxEventType: eventType,
		OutboxPayload:   outboxPayload,
	}); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, a.UserID, "submission.submit", "submission", sub.ID, map[string]any{
		"action": action, "revision": revNum,
	})

	fresh, err := s.repo.ByID(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	return s.withFiles(ctx, fresh)
}

// validateRequired проверяет заполнение обязательных полей активной схемы (SITE.md §7.4).
func validateRequired(fields []FieldInfo, answers map[string]any, files []SubmissionFile) error {
	fileCountByField := map[string]int{}
	for _, f := range files {
		if f.FieldID != nil {
			fileCountByField[*f.FieldID]++
		}
	}
	var missing []string
	for _, f := range fields {
		if !f.Required || f.Type == "SECTION" || f.Type == "INFO_BLOCK" {
			continue
		}
		if f.Type == "FILE_GROUP" {
			if fileCountByField[f.ID] == 0 {
				missing = append(missing, f.Label)
			}
			continue
		}
		v, ok := answers[f.Key]
		if !ok || isEmpty(v) {
			missing = append(missing, f.Label)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(missing, ", "))
	}
	return nil
}

func isEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	case []any:
		return len(t) == 0
	case bool:
		return !t
	}
	return false
}

func fieldsToSnapshot(fields []FieldInfo) []map[string]any {
	out := make([]map[string]any, 0, len(fields))
	for _, f := range fields {
		out = append(out, map[string]any{
			"id": f.ID, "key": f.Key, "type": f.Type, "label": f.Label, "required": f.Required,
		})
	}
	return out
}

func filesToSnapshot(files []SubmissionFile) []map[string]any {
	out := make([]map[string]any, 0, len(files))
	for _, f := range files {
		out = append(out, map[string]any{
			"file_id": f.FileID, "field_id": f.FieldID, "name": f.OriginalName,
		})
	}
	return out
}

// computeChecksum — детерминированный sha256 по ответам и составу файлов.
func computeChecksum(answers map[string]any, files []SubmissionFile) string {
	h := sha256.New()
	keys := make([]string, 0, len(answers))
	for k := range answers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b, _ := json.Marshal(answers[k])
		fmt.Fprintf(h, "%s=%s;", k, b)
	}
	ids := make([]string, 0, len(files))
	for _, f := range files {
		ids = append(ids, f.FileID)
	}
	sort.Strings(ids)
	fmt.Fprintf(h, "files=%s", strings.Join(ids, ","))
	return hex.EncodeToString(h.Sum(nil))
}
