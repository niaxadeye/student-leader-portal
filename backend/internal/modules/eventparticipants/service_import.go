package eventparticipants

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (s *Service) Import(
	ctx context.Context,
	actor Actor,
	contestID string,
	records []ImportRecord,
) (*ImportResult, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	result := &ImportResult{Rows: make([]ImportRowResult, 0, len(records))}
	directions := map[string]*Direction{}
	vkIDs := s.resolveImportedVKIDs(ctx, records)
	for _, record := range records {
		row := s.importOne(ctx, contestID, record, directions, vkIDs)
		switch row.Status {
		case "added":
			result.Added++
		case "updated":
			result.Updated++
		case "duplicate":
			result.Duplicates++
		default:
			result.Errors++
		}
		result.Rows = append(result.Rows, row)
	}
	s.audit.Log(ctx, actor.UserID, "EVENT_PARTICIPANTS_IMPORTED", "contest", contestID, map[string]any{
		"added": result.Added, "updated": result.Updated,
		"errors": result.Errors, "duplicates": result.Duplicates,
	})
	return result, nil
}

// resolveImportedVKIDs разрешает все ссылки файла заранее: построчные запросы
// к VK упёрлись бы в лимит частоты и растянули импорт.
func (s *Service) resolveImportedVKIDs(ctx context.Context, records []ImportRecord) vkIDCache {
	slugs := make([]string, 0, len(records))
	for _, record := range records {
		normalized, err := normalizeOptionalSocialURL(socialVK, optionalString(record.VKURL))
		if err != nil {
			continue
		}
		if slug := vkSlugFromURL(normalized); slug != "" {
			slugs = append(slugs, slug)
		}
	}
	if len(slugs) == 0 {
		return nil
	}
	return s.resolveVKSlugs(ctx, slugs)
}

func (s *Service) importOne(
	ctx context.Context,
	contestID string,
	record ImportRecord,
	directions map[string]*Direction,
	vkIDs vkIDCache,
) ImportRowResult {
	row := ImportRowResult{Line: record.Line, FullName: cleanFullName(record.FullName)}
	birthDate, err := parseBirthDate(record.BirthDate)
	if err != nil {
		row.Status, row.Message = "error", "Некорректная дата рождения"
		return row
	}
	union := optionalString(record.UnionCardNumber)
	sks := optionalString(record.SKSBarcode)
	vkURL := optionalString(record.VKURL)
	telegramURL := optionalString(record.TelegramURL)
	incoming, err := normalizeParticipantInput(contestID, "", CreateInput{
		FullName: record.FullName, BirthDate: birthDate,
		UnionCardNumber: union, SKSBarcode: sks,
		VKURL: vkURL, TelegramURL: telegramURL,
	}, s.now())
	if err != nil {
		if vkURL != nil || telegramURL != nil {
			if _, vkErr := normalizeOptionalSocialURL(socialVK, vkURL); vkErr != nil {
				row.Status, row.Message = "error", "Некорректная ссылка ВКонтакте"
				return row
			}
			if _, tgErr := normalizeOptionalSocialURL(socialTelegram, telegramURL); tgErr != nil {
				row.Status, row.Message = "error", "Некорректная ссылка Telegram"
				return row
			}
		}
		row.Status, row.Message = "error", "Не заполнены обязательные поля или дата некорректна"
		return row
	}
	if directionName := strings.TrimSpace(record.Direction); directionName != "" {
		direction, err := s.resolveImportedDirection(ctx, contestID, directionName, directions)
		if err != nil {
			if errors.Is(err, ErrValidation) {
				row.Status, row.Message = "error", "Некорректное название направления"
				return row
			}
			return importFailure(row, err)
		}
		incoming.DirectionID = &direction.ID
		incoming.DirectionName = &direction.Name
		row.Direction = direction.Name
	}

	byUnion, err := s.importLookup(ctx, contestID, union, s.repo.FindByUnionCard)
	if err != nil {
		return importFailure(row, err)
	}
	bySKS, err := s.importLookup(ctx, contestID, sks, s.repo.FindBySKSBarcode)
	if err != nil {
		return importFailure(row, err)
	}
	if byUnion != nil && bySKS != nil && byUnion.ID != bySKS.ID {
		row.Status, row.Message = "error", "Профбилет и barcode принадлежат разным участникам"
		return row
	}

	existing := byUnion
	if existing == nil {
		existing = bySKS
	}
	if existing == nil {
		matches, err := s.repo.FindByNameBirth(ctx, contestID, incoming.FullNameNormalized, incoming.BirthDate)
		if err != nil {
			return importFailure(row, err)
		}
		if len(matches) > 1 {
			row.Status, row.Message = "duplicate", "Найдено несколько участников с такими ФИО и датой рождения"
			return row
		}
		if len(matches) == 1 {
			existing = &matches[0]
		}
	}

	if existing == nil {
		s.fillVKUserID(ctx, incoming, nil, vkIDs)
		id, err := s.repo.Create(ctx, incoming)
		if err != nil {
			return importFailure(row, err)
		}
		row.Status, row.ParticipantID = "added", id
		return row
	}
	row.ParticipantID = existing.ID
	if existing.Status == StatusArchived {
		row.Status, row.Message = "error", "Участник архивирован и не может быть обновлён импортом"
		return row
	}

	// Пустая optional-ячейка не стирает уже известный идентификатор.
	if incoming.UnionCardNumber == nil {
		incoming.UnionCardNumber = existing.UnionCardNumber
	}
	if incoming.SKSBarcode == nil {
		incoming.SKSBarcode = existing.SKSBarcode
	}
	if incoming.VKURL == nil {
		incoming.VKURL = existing.VKURL
	}
	if incoming.TelegramURL == nil {
		incoming.TelegramURL = existing.TelegramURL
	}
	if incoming.DirectionID == nil {
		incoming.DirectionID = existing.DirectionID
		incoming.DirectionName = existing.DirectionName
	}
	if incoming.DirectionName != nil {
		row.Direction = *incoming.DirectionName
	}
	incoming.ID = existing.ID
	if sameImportedParticipant(existing, incoming) {
		row.Status, row.Message = "duplicate", "Участник уже существует без изменений"
		return row
	}
	s.fillVKUserID(ctx, incoming, existing, vkIDs)
	if err := s.repo.Update(ctx, incoming); err != nil {
		return importFailure(row, err)
	}
	row.Status = "updated"
	return row
}

func (s *Service) resolveImportedDirection(
	ctx context.Context,
	contestID, name string,
	cache map[string]*Direction,
) (*Direction, error) {
	display, err := normalizeDirectionName(name)
	if err != nil {
		return nil, err
	}
	key := strings.ToLower(display)
	if cached := cache[key]; cached != nil {
		return cached, nil
	}
	direction, err := s.repo.EnsureDirection(ctx, contestID, display)
	if err != nil {
		return nil, err
	}
	cache[key] = direction
	return direction, nil
}

func (s *Service) importLookup(
	ctx context.Context,
	contestID string,
	value *string,
	lookup identifierLookup,
) (*Participant, error) {
	if value == nil {
		return nil, nil
	}
	participant, err := lookup(ctx, contestID, *value)
	if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return participant, err
}

func importFailure(row ImportRowResult, err error) ImportRowResult {
	row.Status = "error"
	switch {
	case errors.Is(err, ErrIdentifierTaken):
		row.Message = "Профбилет или barcode уже используется"
	case errors.Is(err, ErrDirectionTaken):
		row.Message = "Название направления уже используется"
	case errors.Is(err, ErrValidation):
		row.Message = "Некорректные данные"
	default:
		row.Message = "Не удалось обработать строку"
	}
	return row
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func sameImportedParticipant(current, incoming *Participant) bool {
	return current.FullName == incoming.FullName &&
		current.BirthDate.Equal(incoming.BirthDate) &&
		equalOptionalIdentifier(current.UnionCardNumber, incoming.UnionCardNumber) &&
		equalOptionalIdentifier(current.SKSBarcode, incoming.SKSBarcode) &&
		equalOptionalIdentifier(current.VKURL, incoming.VKURL) &&
		equalOptionalIdentifier(current.TelegramURL, incoming.TelegramURL) &&
		equalOptionalIdentifier(current.DirectionID, incoming.DirectionID)
}

func equalOptionalIdentifier(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return strings.EqualFold(*left, *right)
}

func (s *Service) Export(ctx context.Context, actor Actor, contestID, format string) (*ExportFile, error) {
	if err := s.ensureManage(ctx, actor, contestID); err != nil {
		return nil, err
	}
	participants, err := s.repo.All(ctx, contestID)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "csv":
		return exportCSV(participants)
	case "xlsx":
		return exportXLSX(participants)
	default:
		return nil, fmt.Errorf("%w: неизвестный формат экспорта", ErrValidation)
	}
}
