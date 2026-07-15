package outbox

import (
	"fmt"
	"html"
	"strings"
)

// eventKindTitle — заголовок сообщения по типу события (SITE.md §15).
func eventKindTitle(action string) string {
	switch action {
	case "RESUBMIT":
		return "🔄 Обновление отправки"
	default:
		return "📥 Новая отправка"
	}
}

// formatSubmission строит HTML-текст уведомления о submission по данным резолвера
// (SITE.md §15, пример сообщения). baseURL — публичный адрес админки для ссылки.
func formatSubmission(v *SubmissionView, action, baseURL string) string {
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString("<b>" + eventKindTitle(action) + "</b>\n\n")
	b.WriteString("Конкурс: " + esc(v.ContestName) + "\n")
	b.WriteString("Испытание: " + esc(v.Challenge) + "\n")
	b.WriteString("Конкурсант: " + esc(v.FullName) + "\n")
	if v.Organization != nil && *v.Organization != "" {
		b.WriteString("Организация: " + esc(*v.Organization) + "\n")
	}
	b.WriteString(fmt.Sprintf("Ревизия: %d\n", v.Revision))
	if v.SubmittedAt != nil {
		b.WriteString("Отправлено: " + esc(*v.SubmittedAt) + "\n")
	}
	b.WriteString(fmt.Sprintf("Файлов: %d\n", v.FileCount))

	url := strings.TrimRight(baseURL, "/") + "/admin/submissions/" + v.SubmissionID
	b.WriteString("\nОткрыть форму: " + esc(url))
	return b.String()
}
