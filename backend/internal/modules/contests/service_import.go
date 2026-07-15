package contests

import (
	"context"
	"strings"
)

// ImportRow — результат обработки одной строки импорта.
type ImportRow struct {
	Line         int    `json:"line"`
	Login        string `json:"login"`
	Status       string `json:"status"` // created | error
	TempPassword string `json:"temp_password,omitempty"`
	Error        string `json:"error,omitempty"`
}

// ImportResult — сводка импорта конкурсантов.
type ImportResult struct {
	Created int         `json:"created"`
	Failed  int         `json:"failed"`
	Rows    []ImportRow `json:"rows"`
}

// ImportContestants парсит CSV (login,full_name,organization; заголовок опционален)
// и добавляет конкурсантов построчно. Скелет Этапа 2: синхронно, без файлов/фоновых задач.
func (s *Service) ImportContestants(ctx context.Context, a Actor, contestID, csv string) (*ImportResult, error) {
	if err := s.ensureAccess(ctx, a, contestID); err != nil {
		return nil, err
	}
	res := &ImportResult{Rows: []ImportRow{}}
	lines := strings.Split(strings.ReplaceAll(csv, "\r\n", "\n"), "\n")
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		cols := splitCSV(line)
		// Пропускаем строку-заголовок.
		if i == 0 && strings.EqualFold(strings.TrimSpace(cols[0]), "login") {
			continue
		}
		row := ImportRow{Line: i + 1}
		if len(cols) < 2 || strings.TrimSpace(cols[0]) == "" || strings.TrimSpace(cols[1]) == "" {
			row.Status, row.Error = "error", "нужны колонки login и full_name"
			res.Rows = append(res.Rows, row)
			res.Failed++
			continue
		}
		org := ""
		if len(cols) >= 3 {
			org = cols[2]
		}
		row.Login = strings.TrimSpace(cols[0])
		out, err := s.AddContestant(ctx, a, contestID, AddContestantInput{
			Login: cols[0], FullName: cols[1], Organization: org,
		})
		if err != nil {
			row.Status, row.Error = "error", "не удалось создать"
			res.Failed++
		} else {
			row.Status, row.TempPassword = "created", out.TempPassword
			res.Created++
		}
		res.Rows = append(res.Rows, row)
	}
	return res, nil
}

// splitCSV — простой разбор строки по запятой с trim (без экранирования кавычек;
// достаточно для скелета, полноценный CSV — на Этапе 8 с экспортом).
func splitCSV(line string) []string {
	parts := strings.Split(line, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// ExportContestants формирует CSV активных конкурсантов конкурса.
func (s *Service) ExportContestants(ctx context.Context, a Actor, contestID string) (string, error) {
	if err := s.ensureAccess(ctx, a, contestID); err != nil {
		return "", err
	}
	list, err := s.repo.Participants(ctx, contestID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("login,full_name,organization,status,joined_at\n")
	for _, p := range list {
		org := ""
		if p.Organization != nil {
			org = *p.Organization
		}
		b.WriteString(csvField(p.Login) + "," + csvField(p.FullName) + "," +
			csvField(org) + "," + csvField(p.UserStatus) + "," +
			p.JoinedAt.Format("2006-01-02T15:04:05Z07:00") + "\n")
	}
	return b.String(), nil
}

// csvField экранирует поле по RFC 4180, если содержит запятую/кавычку/перевод строки.
func csvField(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}
