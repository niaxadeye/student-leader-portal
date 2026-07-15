package app

import (
	"context"

	"github.com/eazytech/student-leader-cabinet/internal/modules/outbox"
)

// StartBackground запускает фоновые процессы приложения (диспетчер outbox →
// Telegram). Диспетчер живёт в горутине API-процесса (по договорённости Этапа 5:
// один инстанс, отдельный worker-бинарь оставлен тонким). Возврат — функция
// остановки для graceful-shutdown.
func (a *App) StartBackground(ctx context.Context) {
	sender := outbox.NewTelegramSender(outbox.TelegramConfig{
		Token:         a.cfg.Telegram.BotToken,
		DefaultChat:   a.cfg.Telegram.DefaultChatID,
		DefaultThread: a.cfg.Telegram.DefaultThreadID,
		Enabled:       a.cfg.Telegram.Enabled,
	})
	disp := outbox.NewDispatcher(outbox.NewRepo(a.pool), sender, a.log, outbox.Config{
		BaseURL: a.cfg.App.BaseURL,
	})
	go disp.Run(ctx)
	if a.cfg.Telegram.Enabled {
		a.log.Info("outbox dispatcher started", "telegram", "enabled")
	} else {
		a.log.Info("outbox dispatcher started", "telegram", "disabled (events accumulate)")
	}
}
